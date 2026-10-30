package mkdocs

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestMkDocToString(t *testing.T) {
	root := NewMkDocs("Root's page", "docs/index.md")
	section := NewMkDocs("Section", "docs/section/index.md")
	section.AddChild(NewMkDocs("Leaf", "docs/section/leaf.md"))
	root.AddChild(section)
	root.AddChild(NewMkDocs("Sibling", "docs/sibling.md"))

	wrapper := NewMkDocs("root", "root")
	wrapper.AddChild(root)
	got, err := wrapper.GenerateMkDocs()
	if err != nil {
		t.Fatal(err)
	}

	var config mkDocsConfig
	if err := yaml.Unmarshal([]byte(got), &config); err != nil {
		t.Fatalf("generated invalid YAML: %v\n%s", err, got)
	}
	if config.SiteName != "Root's page" {
		t.Errorf("site_name = %q, want %q", config.SiteName, "Root's page")
	}
	wantNavigation := []any{
		map[string]any{"Root's page": "index.md"},
		map[string]any{"Section": []any{
			"section/index.md",
			map[string]any{"Leaf": "section/leaf.md"},
		}},
		map[string]any{"Sibling": "sibling.md"},
	}
	if !reflect.DeepEqual(config.Nav, wantNavigation) {
		t.Errorf("navigation = %#v, want %#v", config.Nav, wantNavigation)
	}
	if config.RepoURL != "https://<REPO_URL>" || config.EditURI != "?path=<PATH>" ||
		config.Theme.Name != "material" || len(config.Plugins) != 2 {
		t.Errorf("generated configuration lost static settings: %+v", config)
	}
}

func TestGenerateMkDocsEscapesTitlesAndPaths(t *testing.T) {
	root := NewMkDocs("Root: '# page'\nsecond line", "docs/root: # page.md")
	root.AddChild(NewMkDocs("Section: [details]\n# heading", "docs/section: [details]/# heading.md"))

	wrapper := NewMkDocs("root", "root")
	wrapper.AddChild(root)
	got, err := wrapper.GenerateMkDocs()
	if err != nil {
		t.Fatal(err)
	}

	var config mkDocsConfig
	if err := yaml.Unmarshal([]byte(got), &config); err != nil {
		t.Fatalf("generated invalid YAML: %v\n%s", err, got)
	}
	if config.SiteName != root.Title {
		t.Errorf("site_name = %q, want %q", config.SiteName, root.Title)
	}
	wantNavigation := []any{
		map[string]any{root.Title: root.Path},
		map[string]any{root.Children[0].Title: root.Children[0].Path},
	}
	if !reflect.DeepEqual(config.Nav, wantNavigation) {
		t.Errorf("navigation = %#v, want %#v", config.Nav, wantNavigation)
	}
}

func TestGenerateMkDocsEmpty(t *testing.T) {
	got, err := NewMkDocs("root", "root").GenerateMkDocs()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("empty navigation generated %q", got)
	}
}
