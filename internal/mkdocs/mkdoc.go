package mkdocs

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type MkDoc struct {
	ID       string
	Title    string
	Children []*MkDoc
}

func NewMkDocs(id, title string) *MkDoc {
	return &MkDoc{
		ID:       id,
		Title:    title,
		Children: []*MkDoc{},
	}
}

func (m *MkDoc) AddChild(child *MkDoc) {
	m.Children = append(m.Children, child)
}

func cleanString(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

func mkDocToString(node *MkDoc, depth int, root bool) string {
	nodeId := node.ID
	var sb strings.Builder
	title := cleanString(node.Title)

	sb.WriteString(strings.Repeat(" ", depth*2))
	sb.WriteString("- '")
	sb.WriteString(title)
	sb.WriteString("':")

	if len(node.Children) > 0 {
		if root {
			sb.WriteString(" ")
			sb.WriteString(nodeId)
			sb.WriteString("\n")
		} else {
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat(" ", (depth+1)*2))
			sb.WriteString("- ")
			sb.WriteString(nodeId)
			sb.WriteString("\n")
			depth += 1
		}
		for _, child := range node.Children {
			sb.WriteString(mkDocToString(child, depth, false))
		}
	} else {
		sb.WriteString(" ")
		sb.WriteString(nodeId)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m *MkDoc) GenerateMkDocs() string {
	if len(m.Children) == 0 {
		return ""
	}

	log.Debug().Interface("root", m).Msgf("Generating MkDocs navigation for root page: %s", m.Children[0].Title)

	root := m.Children[0] // Equivalent to Java's getChildren().getFirst()
	navigation := mkDocToString(root, 1, true)

	template := `site_name: '%s'
repo_url: TODO
edit_uri: ?path=<PATH>

theme:
  name: material

nav:
%s
plugins:
  - techdocs-core
  - section-index
  - git-revision-date-localized:
      fallback_to_build_date: true
`

	return fmt.Sprintf(template, cleanString(root.Title), navigation)
}