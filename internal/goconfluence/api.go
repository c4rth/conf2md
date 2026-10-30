package goconfluence

import (
	"net/http"
	"net/url"
)

// API is the main api data structure
type API struct {
	BaseURL         *url.URL
	Subpath         string
	Client          *http.Client
	username, token string
}

func (a *API) getEndpoint(path string) (*url.URL, error) {
	return url.ParseRequestURI(a.BaseURL.String() + a.Subpath + path)
}
