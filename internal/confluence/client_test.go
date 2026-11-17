package confluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"conf2md/internal/goconfluence"
)

func TestSortChildPages(t *testing.T) {
	children := []goconfluence.Results{
		{ID: "1", Title: "Zeta", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`"none"`)}},
		{ID: "2", Title: "Third", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`30`)}},
		{ID: "3", Title: "alpha", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`"none"`)}},
		{ID: "4", Title: "First", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`-5`)}},
		{ID: "5", Title: "Second", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`10`)}},
		{ID: "6", Title: "Beta"},
	}

	SortChildPages(children)

	got := make([]string, len(children))
	for i, child := range children {
		got[i] = child.Title
	}
	want := []string{"First", "Second", "Third", "alpha", "Beta", "Zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SortChildPages() = %v, want %v", got, want)
	}
}

func TestGetConfluencePageOrdersExpandedChildrenWithoutExtraRequest(t *testing.T) {
	childEndpointCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/content/100":
			if got, want := r.URL.Query().Get("expand"), "body.storage,children.page.extensions.position"; got != want {
				t.Errorf("expand = %q, want %q", got, want)
			}
			_ = json.NewEncoder(w).Encode(goconfluence.Content{
				ID:    "100",
				Title: "Parent",
				Children: &goconfluence.Children{Page: &goconfluence.Search{
					Limit: 25,
					Results: []goconfluence.Results{
						{ID: "2", Title: "Second", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`20`)}},
						{ID: "1", Title: "First", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`10`)}},
					},
				}},
			})
		case "/content/100/child/page":
			childEndpointCalled = true
			http.NotFound(w, r)
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	page, err := GetConfluencePage(context.Background(), api, "100", true)
	if err != nil {
		t.Fatalf("GetConfluencePage() error = %v", err)
	}
	if childEndpointCalled {
		t.Error("expected inline expanded positions to avoid calling /child/page")
	}
	got := []string{page.Children.Page.Results[0].ID, page.Children.Page.Results[1].ID}
	want := []string{"1", "2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("child IDs = %v, want %v", got, want)
	}
}

func TestGetConfluencePageFallsBackToChildEndpointWhenPositionsMissingOrPaginated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/content/100":
			// Simulate an instance where nested children.page expansion omits extensions.position.
			_ = json.NewEncoder(w).Encode(goconfluence.Content{
				ID:    "100",
				Title: "Parent",
				Children: &goconfluence.Children{Page: &goconfluence.Search{
					Limit: 25,
					Results: []goconfluence.Results{
						{ID: "2", Title: "B Page"},
						{ID: "1", Title: "A Page"},
					},
				}},
			})
		case "/content/100/child/page":
			if got, want := r.URL.Query().Get("expand"), "extensions.position"; got != want {
				t.Errorf("child/page expand = %q, want %q", got, want)
			}
			_ = json.NewEncoder(w).Encode(goconfluence.Search{
				Limit: 25,
				Results: []goconfluence.Results{
					{ID: "2", Title: "B Page", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`5`)}},
					{ID: "1", Title: "A Page", Extensions: &goconfluence.Extensions{Position: json.RawMessage(`15`)}},
				},
			})
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	api, err := goconfluence.NewAPIWithClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	page, err := GetConfluencePage(context.Background(), api, "100", true)
	if err != nil {
		t.Fatalf("GetConfluencePage() error = %v", err)
	}
	got := []string{page.Children.Page.Results[0].ID, page.Children.Page.Results[1].ID}
	want := []string{"2", "1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("child IDs = %v, want %v", got, want)
	}
}
