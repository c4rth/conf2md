package goconfluence

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// Search results
type Search struct {
	Results   []Results `json:"results"`
	Start     int       `json:"start,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Size      int       `json:"size,omitempty"`
	TotalSize int       `json:"totalSize,omitempty"`
	Links     *Links    `json:"_links,omitempty"`
}

// SearchLinks Parsing out the _links section to allow paging etc.
type SearchLinks struct {
	Base    string `json:"base,omitempty"`
	Context string `json:"content,omitempty"`
	Next    string `json:"next,omitempty"`
	Self    string `json:"self,omitempty"`
}

// SearchQuery defines query parameters used for searchng
// Query parameter values https://developer.atlassian.com/cloud/confluence/rest/#api-search-get
type SearchQuery struct {
	CQL                   string
	CQLContext            string
	IncludeArchivedSpaces bool
	Limit                 int
	Start                 int
	Expand                []string
}

// getContentEndpoint creates the correct api endpoint by given id
func (a *API) getSearchEndpoint() (*url.URL, error) {
	return a.getEndpoint("/search")
}

// addQueryParameters add the given url.Values into a query string without escaping (for Confluence CQL)
func addQueryParameters(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder
	// To minimize allocations, we eschew iterators and pre-size the slice in
	// which we collect v's keys.
	keys := make([]string, len(v))
	var i int
	for k := range v {
		keys[i] = k
		i++
	}
	slices.Sort(keys)
	for _, k := range keys {
		vs := v[k]
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}
	return buf.String()
}

// Search querys confluence using CQL
func (a *API) Search(ctx context.Context, query SearchQuery) (*Search, error) {
	ep, err := a.getSearchEndpoint()
	if err != nil {
		return nil, err
	}
	ep.RawQuery = addQueryParameters(*addSearchQueryParams(query))
	return a.SendSearchRequest(ctx, ep, "GET")
}

// SearchWithNext querys confluence using CQL, with the ability to pass in the Next header
// Empty Next header will run search like normal
func (a *API) SearchWithNext(ctx context.Context, query SearchQuery, next string) (*Search, error) {
	if next == "" {
		return a.Search(ctx, query)
	}
	ep, err := url.ParseRequestURI(a.BaseURL.String() + next)
	if err != nil {
		return nil, err
	}
	return a.SendSearchRequest(ctx, ep, "GET")
}

// addSearchQueryParams adds the defined query parameters
func addSearchQueryParams(query SearchQuery) *url.Values {

	data := url.Values{}
	if query.CQL != "" {
		data.Set("cql", query.CQL)
	}
	if query.CQLContext != "" {
		data.Set("cqlcontext", query.CQLContext)
	}
	if query.IncludeArchivedSpaces {
		data.Set("includeArchivedSpaces", "true")
	}
	if query.Limit != 0 {
		data.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Start != 0 {
		data.Set("start", strconv.Itoa(query.Start))
	}
	if len(query.Expand) != 0 {
		data.Set("expand", strings.Join(query.Expand, ","))
	}
	return &data
}
