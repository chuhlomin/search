package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/pkg/errors"
)

type response struct {
	ID        string      `json:"id"`
	Score     float64     `json:"score"`
	Fragments []fragment  `json:"fragments"`
	Document  interface{} `json:"document,omitempty"`
}

type fragment struct {
	Field     string                `json:"field"`
	Locations map[string][]location `json:"locations"`
}

type location struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

func (s *server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryString := r.URL.Query().Get("q")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "error reading request body", http.StatusInternalServerError)
			return
		}

		fields, err := parseFields(body)
		if err != nil {
			log.Printf("Error parsing fields: %v", err)
			http.Error(w, "error parsing request body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		query := bleve.NewMatchQuery(queryString)
		search := bleve.NewSearchRequest(query)
		search.Highlight = bleve.NewHighlight()
		search.IncludeLocations = true
		search.Fields = fields
		log.Printf("Searching fields %s", fields)

		searchResults, err := s.index.Search(search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Took", fmt.Sprintf("%v", searchResults.Took))

		resp := formatResponse(searchResults)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func parseFields(body []byte) ([]string, error) {
	if len(body) == 0 {
		return nil, nil
	}

	// parse arbitrary json
	var document map[string]interface{}
	if err := json.Unmarshal(body, &document); err != nil {
		log.Println(err)
		return nil, errors.Wrap(err, "error parsing request body")
	}

	// extract fields
	var fields []string
	for field, value := range document {
		fields = append(fields, extractFields(field, value)...)
	}

	return fields, nil
}

func extractFields(field string, value interface{}) []string {
	switch value.(type) {
	case bool, float64, string:
		return []string{field}
	case map[string]interface{}:
		// parse nested fields
		var fields []string
		for nestedField, nestedValue := range value.(map[string]interface{}) {
			fields = append(
				fields,
				extractFields(
					fmt.Sprintf("%s.%s", field, nestedField),
					nestedValue,
				)...,
			)
		}
		return fields
	default:
		log.Printf("Unsupported type: %T", value)
	}
	return nil
}

func formatResponse(searchResults *bleve.SearchResult) []response {
	var resp []response
	for _, hit := range searchResults.Hits {
		var fragments []fragment
		for field, termLocations := range hit.Locations {
			fragment := fragment{
				Field:     field,
				Locations: map[string][]location{},
			}
			for term, locations := range termLocations {
				fragment.Locations[term] = []location{}
				for _, loc := range locations {
					fragment.Locations[term] = append(fragment.Locations[term], location{
						Start: loc.Start,
						End:   loc.End,
					})
				}
			}
			fragments = append(fragments, fragment)
		}

		resp = append(resp, response{
			ID:        hit.ID,
			Score:     hit.Score,
			Fragments: fragments,
			Document:  buildDocument(hit.Fields),
		})
	}
	return resp
}

func buildDocument(fields map[string]interface{}) interface{} {
	document := map[string]interface{}{}
	for field, value := range fields {
		path := strings.Split(field, ".")
		if len(path) == 1 {
			document[field] = value
		} else {
			var current interface{} = document
			found := true
			for _, part := range path[:len(path)-1] {
				if currentMap, ok := current.(map[string]interface{}); ok {
					if currentMap[part] == nil {
						currentMap[part] = map[string]interface{}{}
					}
					current = currentMap[part]
				} else {
					found = false
					break
				}
			}
			if found {
				log.Printf("%s: %v", path, value)
				current.(map[string]interface{})[path[len(path)-1]] = value
			}
		}
	}
	return document
}
