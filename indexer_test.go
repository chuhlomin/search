package search

import (
	"os"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/stretchr/testify/require"
)

func searchText(index bleve.Index, query string, t *testing.T) []string {
	return search(index, bleve.NewQueryStringQuery(query), t)
}

func searchDate(index bleve.Index, start, end time.Time, t *testing.T) []string {
	return search(index, bleve.NewDateRangeQuery(start, end), t)
}

func search(index bleve.Index, query query.Query, t *testing.T) []string {
	result, err := index.Search(bleve.NewSearchRequest(query))
	if err != nil {
		t.Fatal(err, "failed to search")
	}

	hits := []string{}
	for _, hit := range result.Hits {
		hits = append(hits, hit.ID)
	}
	return hits
}

func TestIndexerSimple(t *testing.T) {
	type simple struct {
		Text string
	}

	path := "ignore/simple"
	os.RemoveAll(path)

	indexer, err := NewIndexer(path)
	require.NoError(t, err, "failed to create indexer")

	err = indexer.RegisterType(simple{})
	require.NoError(t, err, "failed to register type")

	err = indexer.Index("alice", simple{Text: "Ping"})
	require.NoError(t, err, "failed to index")

	err = indexer.Index("bob", simple{Text: "Pong"})
	require.NoError(t, err, "failed to index")

	err = indexer.Close()
	require.NoError(t, err, "failed to close indexer")

	index, err := bleve.Open(path)
	require.NoError(t, err, "failed to open index")

	result, err := index.Search(
		bleve.NewSearchRequest(
			bleve.NewQueryStringQuery("ping"),
		),
	)
	require.NoError(t, err, "failed to search")

	if result.Total != 1 {
		t.Errorf("expected 1 result, got %d", result.Total)
	}
	if result.Hits[0].ID != "alice" {
		t.Errorf("expected alice, got %s", result.Hits[0].ID)
	}
}

type tags struct {
	Text string `indexer:"text"`
	Date string `indexer:"date"`
	Temp string `indexer:"no_index"`
	Pass string `indexer:"no_store"`
}

func (t tags) Type() string {
	return "tags"
}

func TestIndexerTags(t *testing.T) {
	path := "ignore/tags"
	os.RemoveAll(path)

	indexer, err := NewIndexer(path)
	if err != nil {
		t.Fatal(err, "failed to create indexer")
	}

	err = indexer.RegisterType(tags{})
	require.NoError(t, err, "failed to register type")

	err = indexer.Index("one", tags{Text: "Ping", Date: "2006-01-02", Temp: "temp", Pass: "pass"})
	require.NoError(t, err, "failed to index")

	err = indexer.Close()
	require.NoError(t, err, "failed to close indexer")

	index, err := bleve.Open(path)
	require.NoError(t, err, "failed to open index")

	require.Equal(t, []string{"one"}, searchText(index, "ping", t))
	require.Equal(t, []string{}, searchText(index, "temp", t))
	require.Equal(t, []string{}, searchText(index, "pass", t))
	require.Equal(t, []string{"one"}, searchDate(index, time.Date(2006, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2006, 1, 3, 0, 0, 0, 0, time.UTC), t))
	require.Equal(t, []string{}, searchDate(index, time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2010, 1, 3, 0, 0, 0, 0, time.UTC), t))
}

type nested struct {
	Text string `indexer:"text"`
	Meta struct {
		Label string `indexer:"text"`
	}
	Data data
}

type data struct {
	Kicker string `indexer:"text"`
}

func (n nested) Type() string {
	return "nested"
}

func (n nested) Language() string {
	return "en"
}

func TestIndexerNested(t *testing.T) {
	path := "ignore/nested"
	os.RemoveAll(path)

	indexer, err := NewIndexer(path)
	if err != nil {
		t.Fatal(err, "failed to create indexer")
	}

	err = indexer.RegisterType(nested{})
	require.NoError(t, err, "failed to register type")

	err = indexer.Index(
		"one",
		nested{
			Text: "Cats",
			Meta: struct {
				Label string `indexer:"text"`
			}{Label: "chatting"},
			Data: data{Kicker: "Kicking"},
		},
	)
	require.NoError(t, err, "failed to index")

	err = indexer.Close()
	require.NoError(t, err, "failed to close indexer")

	index, err := bleve.Open(path)
	require.NoError(t, err, "failed to open index")

	require.Equal(t, []string{"one"}, searchText(index, "cat", t), "failed to search for cat")
	require.Equal(t, []string{"one"}, searchText(index, "chat", t), "failed to search for chat")
	require.Equal(t, []string{"one"}, searchText(index, "kick", t), "failed to search for kick")
}
