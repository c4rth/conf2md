package goconfluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

// Request implements the basic Request function
func (a *API) Request(req *http.Request) ([]byte, error) {
	req.Header.Add("Accept", "application/json, */*")

	// only auth if we can auth
	if (a.username != "") || (a.token != "") {
		a.Auth(req)
	}

	if event := log.Debug(); event.Enabled() {
		dumpReq := req.Clone(req.Context())
		dumpReq.Header.Del("Authorization")
		dumpReq.Header.Del("Proxy-Authorization")
		dumpReq.Header.Del("Cookie")
		requestDump, err := httputil.DumpRequest(dumpReq, isTextContentType(req.Header.Get("Content-Type")))
		// DumpRequest can replace the body after reading it.
		req.Body = dumpReq.Body
		if err != nil {
			event.Err(err).Msg("Failed to dump request")
		} else {
			event.Msg(string(requestDump))
		}
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Debug().Msgf("====== Response Status Code: %d ======", resp.StatusCode)

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusResetContent {
			return nil, nil
		}
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}

		if event := log.Debug(); event.Enabled() {
			contentType := resp.Header.Get("Content-Type")
			if contentType == "" {
				contentType = http.DetectContentType(res)
			}
			if isTextContentType(contentType) {
				event.Msg(string(res))
			} else {
				event.Int("bytes", len(res)).Msg("Binary response body omitted")
			}
		}
		return res, nil
	}

	message := "request failed"
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		message = "authentication failed"
	case http.StatusServiceUnavailable:
		message = "service is not available"
	case http.StatusInternalServerError:
		message = "internal server error"
	case http.StatusConflict:
		message = "conflict"
	}

	const responseBodySnippetLimit = 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseBodySnippetLimit+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %s: read response body: %w", message, resp.Status, err)
	}

	truncated := len(body) > responseBodySnippetLimit
	if truncated {
		body = body[:responseBodySnippetLimit]
	}
	errMsg := fmt.Sprintf("%s: %s", message, resp.Status)
	if len(body) > 0 {
		snippet := strings.TrimSpace(string(body))
		if truncated && snippet != "" {
			snippet += "..."
		}
		if snippet != "" {
			errMsg += fmt.Sprintf(": %q", snippet)
		}
	}
	return nil, fmt.Errorf("%s", errMsg)
}

func isTextContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.HasPrefix(mediaType, "text/") ||
		mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") ||
		mediaType == "application/xml" || strings.HasSuffix(mediaType, "+xml") ||
		mediaType == "application/x-www-form-urlencoded"
}

// SendContentRequest sends content related requests
// this function is used for getting, updating and deleting content
func (a *API) SendContentRequest(ctx context.Context, ep *url.URL, method string, c *Content) (*Content, error) {

	var body io.Reader
	if c != nil {
		js, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(js)
	}

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var content Content
	if len(res) != 0 {
		err = json.Unmarshal(res, &content)
		if err != nil {
			return nil, err
		}
	}
	return &content, nil
}

// SendUserRequest sends user related requests
func (a *API) SendUserRequest(ctx context.Context, ep *url.URL, method string) (*User, error) {

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var user User

	err = json.Unmarshal(res, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// SendSearchRequest sends search related requests
func (a *API) SendSearchRequest(ctx context.Context, ep *url.URL, method string) (*Search, error) {

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var search Search

	err = json.Unmarshal(res, &search)
	if err != nil {
		return nil, err
	}

	return &search, nil
}

// SendHistoryRequest requests history
func (a *API) SendHistoryRequest(ctx context.Context, ep *url.URL, method string) (*History, error) {

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var history History

	err = json.Unmarshal(res, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

// SendLabelRequest requests history
func (a *API) SendLabelRequest(ctx context.Context, ep *url.URL, method string, labels *[]Label) (*Labels, error) {

	var body io.Reader
	if labels != nil {
		js, err := json.Marshal(labels)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(js)
	}

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	if res != nil {
		var l Labels

		err = json.Unmarshal(res, &l)
		if err != nil {
			return nil, err
		}

		return &l, nil
	}

	return &Labels{}, nil
}

// SendAllSpacesRequest sends a request for all spaces
func (a *API) SendAllSpacesRequest(ctx context.Context, ep *url.URL, method string) (*AllSpaces, error) {

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var allSpaces AllSpaces

	err = json.Unmarshal(res, &allSpaces)
	if err != nil {
		return nil, err
	}

	return &allSpaces, nil
}

// SendContentVersionRequest requests a version of a specific content
func (a *API) SendContentVersionRequest(ctx context.Context, ep *url.URL, method string) (*ContentVersionResult, error) {

	req, err := http.NewRequestWithContext(ctx, method, ep.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := a.Request(req)
	if err != nil {
		return nil, err
	}

	var versionResult ContentVersionResult

	err = json.Unmarshal(res, &versionResult)
	if err != nil {
		return nil, err
	}

	return &versionResult, nil
}
