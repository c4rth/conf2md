package gliffy

import (
	"encoding/json"
	"testing"
)

func TestResourceIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"integer", `123`, "123"},
		{"numeric string", `"123"`, "123"},
		{"string", `"resource-abc"`, "resource-abc"},
		{"large integer", `9007199254740993`, "9007199254740993"},
		{"null", `null`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resource Resource
			if err := json.Unmarshal([]byte(`{"id":`+tt.raw+`,"mimeType":"image/svg+xml","data":"<svg/>"}`), &resource); err != nil {
				t.Fatal(err)
			}
			if resource.ID != tt.want || resource.MimeType != "image/svg+xml" || resource.Data != "<svg/>" {
				t.Errorf("unexpected resource: %+v", resource)
			}
			var svg SVG
			if err := json.Unmarshal([]byte(`{"embeddedResourceId":`+tt.raw+`}`), &svg); err != nil {
				t.Fatal(err)
			}
			if tt.raw == "null" {
				if svg.EmbeddedResourceID != nil {
					t.Error("null reference should remain nil")
				}
			} else if svg.EmbeddedResourceID == nil || *svg.EmbeddedResourceID != tt.want {
				t.Errorf("unexpected reference: %+v", svg)
			}
		})
	}
}

func TestInvalidResourceIdentifiers(t *testing.T) {
	for _, raw := range []string{"true", "{}", "[]"} {
		t.Run(raw, func(t *testing.T) {
			var resource Resource
			if err := json.Unmarshal([]byte(`{"id":`+raw+`}`), &resource); err == nil {
				t.Error("invalid resource ID should fail decoding")
			}
			var svg SVG
			if err := json.Unmarshal([]byte(`{"embeddedResourceId":`+raw+`}`), &svg); err == nil {
				t.Error("invalid SVG reference should fail decoding")
			}
		})
	}
}
