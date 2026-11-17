package converter

import (
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"golang.org/x/net/html"

	"conf2md/internal/confluenceplugin"
	table "conf2md/internal/tableplugin"
)

var plainTextBody = regexp.MustCompile(
	`(?s)<ac:plain-text-body>\s*<!\[CDATA\[(.*?)\]\]>\s*</ac:plain-text-body>`,
)

// emptyParagraph matches <p></p>, <p> </p>, <p/> and the same forms with attributes.
var emptyParagraph = regexp.MustCompile(`(?i)<p(?:\s[^>]*)?(?:/>|>\s*</p\s*>)`)

var confluenceConverter = converter.NewConverter(
	converter.WithPlugins(
		confluenceplugin.NewConfluencePlugin(),
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(
			commonmark.WithEmDelimiter("_"),
			commonmark.WithStrongDelimiter("**"),
			commonmark.WithBulletListMarker("-"),
		),
		table.NewTablePlugin(
			table.WithNewlineBehavior(table.NewlineBehaviorPreserve),
		),
	),
)

func convertPlainTextBodies(input string) string {
	matches := plainTextBody.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input
	}

	var result strings.Builder
	lastEnd := 0
	for _, match := range matches {
		result.WriteString(input[lastEnd:match[0]])
		result.WriteString("<pre><code>")
		result.WriteString(stdhtml.EscapeString(input[match[2]:match[3]]))
		result.WriteString("</code></pre>")
		lastEnd = match[1]
	}
	result.WriteString(input[lastEnd:])
	return result.String()
}

func sanitizeConfluenceHTML(input string) string {
	// Code blocks are converted first so that their escaped content is never altered.
	return emptyParagraph.ReplaceAllString(convertPlainTextBodies(input), "")
}

func convertConfluenceToMarkdown(content string) ([]byte, error) {
	cleanedContent := sanitizeConfluenceHTML(content)
	doc, err := html.Parse(strings.NewReader(cleanedContent))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	markdown, err := confluenceConverter.ConvertNode(doc)
	if err != nil {
		return nil, fmt.Errorf("converting HTML to Markdown: %w", err)
	}
	return markdown, nil
}
