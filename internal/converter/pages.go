package converter

import (
	"fmt"
	"path/filepath"
	"strconv"

	"conf2md/internal/goconfluence"
	"conf2md/internal/mkdocs"
	"conf2md/internal/storage"
)

func pageOutputLocation(content *goconfluence.Content, children *goconfluence.Search, level int, outputDir, pageName string) (string, string) {
	if pageName == "" {
		pageName = storage.SanitizeFilename(content.Title)
	}
	if level != 0 && children != nil && len(children.Results) == 0 {
		return outputDir, pageName
	}
	if level != 0 {
		outputDir = filepath.Join(outputDir, pageName)
	}
	return outputDir, "index"
}

func uniqueChildPageNames(children []goconfluence.Results) []string {
	names := make([]string, len(children))
	counts := make(map[string]int, len(children))
	for _, child := range children {
		counts[storage.SanitizeFilename(child.Title)]++
	}

	used := make(map[string]struct{}, len(children))
	for i, child := range children {
		sanitizeFilename := storage.SanitizeFilename(child.Title)
		name := sanitizeFilename
		if counts[sanitizeFilename] > 1 {
			suffix := storage.SanitizeFilename(child.ID)
			if suffix == "untitled" {
				suffix = strconv.Itoa(i + 1)
			}
			name += "-" + suffix
		}

		candidate := name
		for suffix := 2; ; suffix++ {
			if _, exists := used[candidate]; !exists {
				break
			}
			candidate = name + "-" + strconv.Itoa(suffix)
		}
		names[i] = candidate
		used[candidate] = struct{}{}
	}
	return names
}

func (c *Crawler) convertAndWritePage(content *goconfluence.Content, outputDir, filename string, mkDoc *mkdocs.MkDoc) (*mkdocs.MkDoc, error) {
	md, err := convertConfluenceToMarkdown(content.Body.Storage.Value)
	if err != nil {
		return nil, fmt.Errorf("converting Confluence page '%s' to Markdown: %w", content.Title, err)
	}

	outFile := filepath.Join(c.args.OutputDir, outputDir, filename+".md")
	c.logger.Debug().Msgf("adding page to MkDocs: %s with title: %s", outFile, content.Title)
	relativePath, err := filepath.Rel(c.args.OutputDir, outFile)
	if err != nil {
		return nil, fmt.Errorf("resolve MkDocs page path: %w", err)
	}
	childMkDoc := mkdocs.NewMkDocs(
		content.Title,
		filepath.ToSlash(relativePath),
	)
	mkDoc.AddChild(childMkDoc)
	if err := storage.WriteFile(outFile, md); err != nil {
		return nil, fmt.Errorf("writing Markdown file: %w", err)
	}
	c.logger.Info().Msgf("markdown file written to: %s", outFile)
	return childMkDoc, nil
}
