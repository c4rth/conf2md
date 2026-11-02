package confluenceplugin

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/marker"
	"golang.org/x/net/html"
)

func listStart(node *html.Node, itemCount int) (int, int) {
	step := 1
	if _, reversed := dom.GetAttribute(node, "reversed"); reversed {
		step = -1
	}

	defaultStart := 1
	if step < 0 {
		defaultStart = itemCount
	}
	start, err := strconv.Atoi(dom.GetAttributeOr(node, "start", strconv.Itoa(defaultStart)))
	if err != nil {
		start = defaultStart
	}
	return start, step
}

func renderMultilineListItem(w converter.Writer, content []byte, indentCount int) {
	lines := bytes.Split(content, []byte("\n"))
	indent := bytes.Repeat([]byte(" "), indentCount)
	indentedCodeBlockNewline := append(bytes.Clone(marker.BytesMarkerCodeBlockNewline), indent...)

	for i, line := range lines {
		line = bytes.ReplaceAll(line, marker.BytesMarkerCodeBlockNewline, indentedCodeBlockNewline)
		if i > 0 {
			_, _ = w.Write(indent)
		}
		_, _ = w.Write(line)
		if i < len(lines)-1 {
			_, _ = w.WriteString("\n")
		}
	}
}

func (cp *confluencePlugin) renderList(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	items := make([]*html.Node, 0)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "li" {
			items = append(items, child)
		}
	}
	if len(items) == 0 {
		return converter.RenderSuccess
	}

	start, step := listStart(node, len(items))
	numbers := make([]int, len(items))
	width := 1
	nextNumber := start
	for i, item := range items {
		if value, found := dom.GetAttribute(item, "value"); found {
			if parsed, err := strconv.Atoi(value); err == nil {
				nextNumber = parsed
			}
		}
		numbers[i] = nextNumber
		nextNumber += step
		if currentWidth := utf8.RuneCountInString(strconv.Itoa(numbers[i])); currentWidth > width {
			width = currentWidth
		}
	}

	WriteString(w, "\n\n")
	renderedItems := 0
	for i, item := range items {
		var content bytes.Buffer
		ctx.RenderChildNodes(ctx, &content, item)
		rendered := bytes.TrimSpace(content.Bytes())
		if len(rendered) == 0 {
			continue
		}

		prefix := "- "
		if node.Data == "ol" {
			prefix = fmt.Sprintf("%0*d. ", width, numbers[i])
		}
		WriteString(w, prefix)
		rendered = trimListNewlines(rendered)
		rendered = trimListHardBreaks(rendered)
		rendered = ctx.UnEscapeContent(rendered)
		renderMultilineListItem(w, rendered, 4)

		renderedItems++
		if renderedItems < len(items) {
			WriteString(w, "\n")
		}
		
	}
	WriteString(w, endingListMarker + "\n")
	return converter.RenderSuccess
}

func trimListNewlines(content []byte) []byte {
	var result []byte
	newlineCount := 0
	spaces := make([]byte, 0)
	for i := 0; i < len(content); {
		r, size := utf8.DecodeRune(content[i:])
		switch r {
		case '\n':
			newlineCount++
			if newlineCount <= 2 {
				result = append(result, spaces...)
				result = append(result, '\n')
			}
			spaces = spaces[:0]
		case ' ':
			spaces = append(spaces, content[i:i+size]...)
		default:
			newlineCount = 0
			result = append(result, spaces...)
			result = append(result, content[i:i+size]...)
			spaces = spaces[:0]
		}
		i += size
	}
	return append(result, spaces...)
}

func trimListHardBreaks(content []byte) []byte {
	content = bytes.ReplaceAll(content, []byte("  \n\n"), []byte("\n\n"))
	content = bytes.ReplaceAll(content, []byte("  \n  \n"), []byte("\n\n"))
	return bytes.ReplaceAll(content, []byte("  \n \n"), []byte("\n\n"))
}
