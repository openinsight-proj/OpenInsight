package client

import (
	"encoding/json"
	"net/http"
)

// SearchResult is the result of a search in Elasticsearch.
type SearchResult struct {
	Header          http.Header          `json:"-"`
	TookInMillis    int64                `json:"took,omitempty"`             // search time in milliseconds
	TerminatedEarly bool                 `json:"terminated_early,omitempty"` // request terminated early
	NumReducePhases int                  `json:"num_reduce_phases,omitempty"`
	Clusters        *SearchResultCluster `json:"_clusters,omitempty"`    // 6.1.0+
	ScrollId        string               `json:"_scroll_id,omitempty"`   // only used with Scroll and Scan operations
	Hits            *SearchHits          `json:"hits,omitempty"`         // the actual search hits
	Aggregations    Aggregations         `json:"aggregations,omitempty"` // results from aggregations
	Suggest         SearchSuggest        `json:"suggest,omitempty"`      // results from suggesters
	TimedOut        bool                 `json:"timed_out,omitempty"`    // true if the search timed out
	Error           *ErrorDetails        `json:"error,omitempty"`        // only used in MultiGet
	Status          int                  `json:"status,omitempty"`       // used in MultiSearch
}

// Aggregations is a list of aggregations that are part of a search result.
type Aggregations map[string]*json.RawMessage

// SearchSuggest is a map of suggestions.
// See https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-suggesters.html.
type SearchSuggest map[string][]SearchSuggestion

// SearchSuggestion is a single search suggestion.
// See https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-suggesters.html.
type SearchSuggestion struct {
	Text    string                   `json:"text"`
	Offset  int                      `json:"offset"`
	Length  int                      `json:"length"`
	Options []SearchSuggestionOption `json:"options"`
}

// SearchSuggestionOption is an option of a SearchSuggestion.
// See https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-suggesters.html.
type SearchSuggestionOption struct {
	Text            string              `json:"text"`
	Index           string              `json:"_index"`
	Type            string              `json:"_type"`
	Id              string              `json:"_id"`
	Score           float64             `json:"score"`  // term and phrase suggesters uses "score" as of 6.2.4
	ScoreUnderscore float64             `json:"_score"` // completion and context suggesters uses "_score" as of 6.2.4
	Highlighted     string              `json:"highlighted"`
	CollateMatch    bool                `json:"collate_match"`
	Freq            int                 `json:"freq"` // from TermSuggestion.Option in Java API
	Source          *json.RawMessage    `json:"_source"`
	Contexts        map[string][]string `json:"contexts,omitempty"`
}

// SearchResultCluster holds information about a search response
// from a cluster.
type SearchResultCluster struct {
	Successful int `json:"successful,omitempty"`
	Total      int `json:"total,omitempty"`
	Skipped    int `json:"skipped,omitempty"`
}

// SearchHit is a single hit.
type SearchHit struct {
	Score          *float64               `json:"_score,omitempty"`   // computed score
	Index          string                 `json:"_index,omitempty"`   // index name
	Type           string                 `json:"_type,omitempty"`    // type meta field
	Id             string                 `json:"_id,omitempty"`      // external or internal
	Uid            string                 `json:"_uid,omitempty"`     // uid meta field (see MapperService.java for all meta fields)
	Routing        string                 `json:"_routing,omitempty"` // routing meta field
	Parent         string                 `json:"_parent,omitempty"`  // parent meta field
	Version        *int64                 `json:"_version,omitempty"` // version number, when Version is set to true in SearchService
	SeqNo          *int64                 `json:"_seq_no"`
	PrimaryTerm    *int64                 `json:"_primary_term"`
	Sort           []interface{}          `json:"sort,omitempty"`            // sort information
	Highlight      SearchHitHighlight     `json:"highlight,omitempty"`       // highlighter information
	Source         *json.RawMessage       `json:"_source,omitempty"`         // stored document source
	Fields         map[string]interface{} `json:"fields,omitempty"`          // returned (stored) fields
	Explanation    *SearchExplanation     `json:"_explanation,omitempty"`    // explains how the score was computed
	MatchedQueries []string               `json:"matched_queries,omitempty"` // matched queries
	Nested         *NestedHit             `json:"_nested,omitempty"`         // for nested inner hits
	Shard          string                 `json:"_shard,omitempty"`          // used e.g. in Search Explain
	Node           string                 `json:"_node,omitempty"`           // used e.g. in Search Explain

	// HighlightFields
	// SortValues
	// MatchedFilters
}

// NestedHit is a nested innerhit
type NestedHit struct {
	Field  string     `json:"field"`
	Offset int        `json:"offset,omitempty"`
	Child  *NestedHit `json:"_nested,omitempty"`
}

// SearchHits specifies the list of search hits.
type SearchHits struct {
	//TotalHits int64        `json:"total"`               // total number of hits found
	MaxScore *float64     `json:"max_score,omitempty"` // maximum score of all hits
	Hits     []*SearchHit `json:"hits,omitempty"`      // the actual hits returned
}

// SearchHitHighlight is the highlight information of a search hit.
// See https://www.elastic.co/guide/en/elasticsearch/reference/6.8/search-request-highlighting.html
// for a general discussion of highlighting.
type SearchHitHighlight map[string][]string

type SearchExplanation struct {
	Value       float64             `json:"value"`             // e.g. 1.0
	Description string              `json:"description"`       // e.g. "boost" or "ConstantScore(*:*), product of:"
	Details     []SearchExplanation `json:"details,omitempty"` // recursive details
}
