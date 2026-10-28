package gliffy

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestCachedTranslations(t *testing.T) {
	const uid = "com.gliffy.shape.basic.basic_v1.default.triangle"
	for range 8 {
		t.Run("concurrent lookup", func(t *testing.T) {
			t.Parallel()
			translations, err := cachedTranslations()
			if err != nil {
				t.Fatal(err)
			}
			if got, want := translations[uid], "triangle;direction=north;perimeter=trianglePerimeter"; got != want {
				t.Errorf("translation = %q, want %q", got, want)
			}
		})
	}
}

func TestCachedTranslationsNoAllocations(t *testing.T) {
	if _, err := cachedTranslations(); err != nil {
		t.Fatal(err)
	}
	allocations := testing.AllocsPerRun(100, func() {
		if _, err := cachedTranslations(); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 0 {
		t.Errorf("cached translations allocate %g times per call, want 0", allocations)
	}
}

func TestResourceData(t *testing.T) {
	tests := []struct {
		name    string
		id      *int
		want    string
		warning string
	}{
		{"existing", new(1), "<svg/>", ""},
		{"empty", new(2), "", "embedded SVG resource 2 is empty"},
		{"missing", new(3), "", "embedded SVG resource 3 was not found"},
		{"nil ID", nil, "", "embedded SVG resource ID is missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &converter{resources: indexResources([]Resource{
				{ID: 1, Data: "<svg/>"},
				{ID: 1, Data: "duplicate"},
				{ID: 2},
			})}
			if got := c.resourceData(tt.id); got != tt.want {
				t.Errorf("resourceData() = %q, want %q", got, tt.want)
			}
			if tt.warning == "" {
				if len(c.warnings) != 0 {
					t.Errorf("unexpected warnings: %v", c.warnings)
				}
			} else if len(c.warnings) != 1 || c.warnings[0] != tt.warning {
				t.Errorf("warnings = %v, want %q", c.warnings, tt.warning)
			}
		})
	}
	if indexResources(nil) != nil {
		t.Error("empty resources should not allocate an index")
	}
}

func TestConvertIndexesResourcesPerDiagram(t *testing.T) {
	for _, data := range []string{"<svg/>", "<svg></svg>"} {
		input := `{"embeddedResources":{"resources":[{"id":1,"data":"` + data + `"}]},
			"stage":{"objects":[{"id":"svg","graphic":{"type":"Svg","Svg":{"embeddedResourceId":1}}}]}}`
		result, err := Convert([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Warnings) != 0 || !bytes.Contains(result.XML, []byte("image=data:image/svg+xml,"+base64.StdEncoding.EncodeToString([]byte(data))+";")) {
			t.Errorf("SVG resource not converted: %+v", result)
		}
	}
}
