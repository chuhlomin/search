package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFields(t *testing.T) {
	tt := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "empty",
			body: ``,
			want: nil,
		},
		{
			name: "simple",
			body: `{"field": true}`,
			want: []string{"field"},
		},
		{
			name: "values are ignored",
			body: `{"field": true, "other": false}`,
			want: []string{"field", "other"},
		},
		{
			name: "nested",
			body: `{
				"field": true,
				"other": {
					"field": true
				}
			}`,
			want: []string{"field", "other.field"},
		},
		{
			name: "deeply nested",
			body: `{
				"field": true,
				"other": {
					"field": true,
					"nested": {
						"field": true
					}
				}
			}`,
			want: []string{"field", "other.field", "other.nested.field"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFields([]byte(tc.body))
			sort.Strings(got)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBuildDocument(t *testing.T) {
	tt := []struct {
		name   string
		fields map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name: "simple",
			fields: map[string]interface{}{
				"field": true,
			},
			want: map[string]interface{}{
				"field": true,
			},
		},
		{
			name: "nested",
			fields: map[string]interface{}{
				"field.other": true,
			},
			want: map[string]interface{}{
				"field": map[string]interface{}{
					"other": true,
				},
			},
		},
		{
			name: "deeply nested",
			fields: map[string]interface{}{
				"path":                 "url",
				"metadata.title":       "Title",
				"metadata.tags":        []string{"tag1", "tag2"},
				"metadata.author.name": "John Doe",
			},
			want: map[string]interface{}{
				"path": "url",
				"metadata": map[string]interface{}{
					"title": "Title",
					"tags":  []string{"tag1", "tag2"},
					"author": map[string]interface{}{
						"name": "John Doe",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := buildDocument(tc.fields)
			require.Equal(t, tc.want, got)
		})
	}
}
