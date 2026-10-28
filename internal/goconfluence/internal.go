package goconfluence

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// DefaultTimeout bounds each HTTP request, including reading the response body.
const DefaultTimeout = 2 * time.Minute

var (
	errEmptyHTTPClient = errors.New("empty http client")
)

// NewAPI implements API constructor
func NewAPI(baseURL string, subpath string, username string, token string) (*API, error) {
	if len(baseURL) == 0 {
		return nil, errors.New("url empty")
	}

	u, err := url.ParseRequestURI(baseURL)

	if err != nil {
		return nil, err
	}

	a := new(API)
	a.BaseURL = u
	a.Subpath = subpath
	a.token = token
	a.username = username
	a.Client = newHTTPClient(true, DefaultTimeout)

	return a, nil
}

// NewAPIWithClient creates a new API instance using an existing HTTP client.
// Useful when using oauth or other authentication methods.
func NewAPIWithClient(baseURL string, client *http.Client) (*API, error) {
	u, err := url.ParseRequestURI(baseURL)

	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, errEmptyHTTPClient
	}

	a := new(API)
	a.BaseURL = u
	a.Client = client

	return a, nil
}

// VerifyTLS to enable disable certificate checks
func (a *API) VerifyTLS(set bool) {
	timeout := DefaultTimeout
	if a.Client != nil {
		timeout = a.Client.Timeout
	}
	a.Client = newHTTPClient(set, timeout)
}

// SetTimeout sets the maximum duration of each HTTP request. Zero disables the timeout.
func (a *API) SetTimeout(timeout time.Duration) {
	if a.Client == nil {
		a.Client = newHTTPClient(true, timeout)
		return
	}
	a.Client.Timeout = timeout
}

func newHTTPClient(verifyTLS bool, timeout time.Duration) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	// #nosec G402
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: !verifyTLS}
	return &http.Client{Transport: tr, Timeout: timeout}
}
