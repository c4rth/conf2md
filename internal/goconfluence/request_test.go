package goconfluence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestJSONRequestBodies(t *testing.T) {
	content := &Content{Title: "Example"}
	labels := &[]Label{{Name: "example"}}
	tests := []struct {
		name string
		body any
		send func(*API, *url.URL) error
	}{
		{"content", content, func(api *API, endpoint *url.URL) error {
			_, err := api.SendContentRequest(context.Background(), endpoint, http.MethodPost, content)
			return err
		}},
		{"labels", labels, func(api *API, endpoint *url.URL) error {
			_, err := api.SendLabelRequest(context.Background(), endpoint, http.MethodPost, labels)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatal(err)
			}
			api := &API{Client: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				got, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("request body = %q, want %q", got, want)
				}
				if req.Header.Get("Content-Type") != "application/json" || req.ContentLength != int64(len(want)) {
					t.Error("JSON content type or content length changed")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
			})}}
			endpoint, err := url.Parse("http://example.test")
			if err != nil {
				t.Fatal(err)
			}
			if err := tt.send(api, endpoint); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequestDebugLogging(t *testing.T) {
	tests := []struct {
		name        string
		level       zerolog.Level
		contentType string
		body        string
		wantDump    bool
	}{
		{"disabled", zerolog.InfoLevel, "application/json", `{"payload":"value"}`, false},
		{"text", zerolog.DebugLevel, "text/plain; charset=utf-8", "text payload", true},
		{"json", zerolog.DebugLevel, "application/problem+json", `{"payload":"value"}`, true},
		{"binary", zerolog.DebugLevel, "application/octet-stream", "binary payload", false},
		{"detected binary", zerolog.DebugLevel, "", "\x00binary payload", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous := log.Logger
			t.Cleanup(func() { log.Logger = previous })
			var output bytes.Buffer
			log.Logger = zerolog.New(&output).Level(tt.level)
			body := &countingReader{Reader: strings.NewReader(tt.body)}
			req, err := http.NewRequest(http.MethodPost, "http://example.test", body)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set("Authorization", "Bearer secret-token")
			api := &API{Client: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if !tt.wantDump && body.reads != 0 {
					t.Error("logging read the request body before transport")
				}
				sent, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if string(sent) != tt.body {
					t.Errorf("request body = %q, want %q", sent, tt.body)
				}
				if r.Header.Get("Authorization") != "Bearer secret-token" {
					t.Error("logging changed authorization on the outgoing request")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{tt.contentType}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})}}
			got, err := api.Request(req)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.body {
				t.Errorf("response body = %q, want %q", got, tt.body)
			}
			if strings.Contains(output.String(), "secret-token") {
				t.Error("debug output contains authorization token")
			}
			if tt.level == zerolog.InfoLevel && output.Len() != 0 {
				t.Errorf("unexpected debug output: %s", output.String())
			}
			if tt.wantDump && !strings.Contains(output.String(), "payload") {
				t.Errorf("text payload missing from debug output: %s", output.String())
			}
			if !tt.wantDump && strings.Contains(output.String(), "payload") {
				t.Errorf("payload unexpectedly logged: %s", output.String())
			}
		})
	}
}

type countingReader struct {
	io.Reader
	reads int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func TestRequestStatusHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantBody   string
		wantError  string
	}{
		{
			name:       "accepted response",
			statusCode: http.StatusAccepted,
			body:       `{"ok":true}`,
			wantBody:   `{"ok":true}`,
		},
		{
			name:       "no content response",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "client error includes body",
			statusCode: http.StatusNotFound,
			body:       "page does not exist",
			wantError:  `404 Not Found: "page does not exist"`,
		},
		{
			name:       "server error includes bounded body",
			statusCode: http.StatusInternalServerError,
			body:       strings.Repeat("x", 1100),
			wantError:  `internal server error: 500 Internal Server Error: "` + strings.Repeat("x", 1024) + `..."`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()

			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			api := &API{Client: server.Client()}

			got, err := api.Request(req)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("Request() error = %v, want it to contain %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Request() error = %v", err)
			}
			if string(got) != tt.wantBody {
				t.Errorf("Request() body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestRequestClosesBodyWhenReadingFails(t *testing.T) {
	tests := []struct {
		statusCode int
		status     string
		wantError  string
	}{
		{http.StatusOK, "200 OK", "read response body: read failed"},
		{http.StatusNotFound, "404 Not Found", "request failed: 404 Not Found: read response body: read failed"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			body := &failingReadCloser{err: errors.New("read failed")}
			api := &API{Client: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Status:     tt.status,
					Body:       body,
					Header:     make(http.Header),
				}, nil
			})}}
			req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
			if err != nil {
				t.Fatal(err)
			}

			_, err = api.Request(req)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Request() error = %v, want it to contain %q", err, tt.wantError)
			}
			if !body.closed {
				t.Fatal("response body was not closed after read failure")
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingReadCloser struct {
	err    error
	closed bool
}

func (r *failingReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (r *failingReadCloser) Close() error {
	r.closed = true
	return nil
}

func newStalledServer(t *testing.T) *httptest.Server {
	t.Helper()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})
	return server
}

func TestRequestHonorsContextCancellation(t *testing.T) {
	server := newStalledServer(t)
	api, err := NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	done := make(chan error, 1)
	go func() {
		_, err := api.GetContentByID(ctx, "1", ContentQuery{})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("GetContentByID() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request was not cancelled")
	}
}

func TestRequestHonorsClientTimeout(t *testing.T) {
	server := newStalledServer(t)
	api, err := NewAPI(server.URL, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	api.SetTimeout(50 * time.Millisecond)

	start := time.Now()
	_, err = api.GetContentByID(context.Background(), "1", ContentQuery{})
	var netErr interface{ Timeout() bool }
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("GetContentByID() error = %v, want timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("request took %s despite timeout", elapsed)
	}
}

func TestNewAPIUsesDefaultTimeoutAndVerifyTLSPreservesIt(t *testing.T) {
	api, err := NewAPI("https://example.test", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if api.Client.Timeout != DefaultTimeout {
		t.Fatalf("default timeout = %s, want %s", api.Client.Timeout, DefaultTimeout)
	}
	api.SetTimeout(5 * time.Second)
	api.VerifyTLS(false)
	if api.Client.Timeout != 5*time.Second {
		t.Fatalf("timeout after VerifyTLS = %s, want 5s", api.Client.Timeout)
	}
}
