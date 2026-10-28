package confluenceplugin

import (
	"strings"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

// TextFormatTag maps inline formatting tags to the Markdown delimiter wrapped around their content.
var TextFormatTag = map[string]string{
	"u":      "^^",
	"sub":    "~",
	"sup":    "^",
	"strong": "**",
	"b":      "**",
	"em":     "_",
	"i":      "_",
	"s":      "~~",
	"del":    "~~",
	"pre":    "`",
	"span":   "==",
}

func (cp *confluencePlugin) isTextFormatTag(name string) bool {
	_, ok := TextFormatTag[name]
	return ok
}

func (cp *confluencePlugin) renderTextFormatting(ctx converter.Context, w converter.Writer, node *html.Node, name string) converter.RenderStatus {
	switch name {
	case "pre":
		return cp.renderPreCode(ctx, w, node)
	case "span":
		return cp.renderSpan(ctx, w, node)
	default:
		return cp.renderTextWithDelimiter(ctx, w, node, TextFormatTag[name])
	}
}

func (cp *confluencePlugin) renderTextWithDelimiter(ctx converter.Context, w converter.Writer, node *html.Node, delimiter string) converter.RenderStatus {
	WriteString(w, delimiter)
	ctx.RenderChildNodes(ctx, w, node)
	WriteString(w, delimiter)
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderPreCode(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	if node.FirstChild != nil {
		format := dom.NodeName(node.FirstChild)
		if format == "code" {
			ctx.RenderChildNodes(ctx, w, node.FirstChild)
			return converter.RenderSuccess
		}
	}
	return cp.renderTextWithDelimiter(ctx, w, node, "`")
}

func (cp *confluencePlugin) renderSpan(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	style := dom.GetAttributeOr(node, "style", "")
	if strings.Contains(style, "background-color:") {
		return cp.renderTextWithDelimiter(ctx, w, node, "==")
	}
	if strings.Contains(style, "color:") {
		return cp.renderTextWithDelimiter(ctx, w, node, "==")
	}

	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}
