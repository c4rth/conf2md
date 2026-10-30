package mkdocs

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v3"
)

type mkDocsConfig struct {
	SiteName string `yaml:"site_name"`
	RepoURL  string `yaml:"repo_url"`
	EditURI  string `yaml:"edit_uri"`
	Theme    struct {
		Name string `yaml:"name"`
		Features []string `yaml:"features"`
	} `yaml:"theme"`
	Nav     []any `yaml:"nav"`
	Plugins []any `yaml:"plugins"`
}

type MkDoc struct {
	Path     string
	Title    string
	Children []*MkDoc
}

func NewMkDocs(title, path string) *MkDoc {
	return &MkDoc{
		Title:    title,
		Path:     strings.TrimPrefix(path, "docs/"),
		Children: []*MkDoc{},
	}
}

func (m *MkDoc) AddChild(child *MkDoc) {
	m.Children = append(m.Children, child)
}

func navigationEntry(node *MkDoc) map[string]any {
	var value any = node.Path
	if len(node.Children) > 0 {
		children := make([]any, 1, len(node.Children)+1)
		children[0] = node.Path
		for _, child := range node.Children {
			children = append(children, navigationEntry(child))
		}
		value = children
	}
	return map[string]any{node.Title: value}
}

func (m *MkDoc) GenerateMkDocs() (string, error) {
	if len(m.Children) == 0 {
		return "", nil
	}

	log.Debug().Interface("root", m).Msgf("generating MkDocs navigation for root page: %s", m.Children[0].Title)

	root := m.Children[0]
	navigation := []any{map[string]any{root.Title: root.Path}}
	for _, child := range root.Children {
		navigation = append(navigation, navigationEntry(child))
	}

	config := mkDocsConfig{
		SiteName: root.Title,
		RepoURL:  "https://<REPO_URL>",
		EditURI:  "?path=<PATH>",
		Nav:      navigation,
		Plugins: []any{
			"techdocs-core",
			map[string]any{
				"git-revision-date-localized": map[string]any{
					"fallback_to_build_date": true,
				},
			},
		},
	}
	config.Theme.Name = "material"
	config.Theme.Features = []string{"navigation.indexes", "navigation.expand"}

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
