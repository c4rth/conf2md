package confluence_plugin

import (
	"golang.org/x/net/html"
	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

func (cp *confluencePlugin) renderTextFormatting(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	format := dom.NodeName(node)
	switch format {
	case "u":
		renderTextFormatted(ctx, w, node, "^^")
	case "sub":
		renderTextFormatted(ctx, w, node, "~")
	case "sup":
		renderTextFormatted(ctx, w, node, "^")
	case "strong", "b":
		renderTextFormatted(ctx, w, node, "**")
	case "em", "i":
		renderTextFormatted(ctx, w, node, "_")
	case "pre":
		renderPreCode(ctx, w, node)
	case "s":
		renderTextFormatted(ctx, w, node, "~~")
	}
	return converter.RenderSuccess
}

func renderTextFormatted(ctx converter.Context, w converter.Writer, node *html.Node, format string) {
	w.WriteString(format)
	ctx.RenderChildNodes(ctx, w, node)
	defer w.WriteString(format)
}

func renderPreCode(ctx converter.Context, w converter.Writer, node *html.Node) {
	if node.FirstChild != nil {
		format := dom.NodeName(node.FirstChild)
		if format == "code" {
			ctx.RenderChildNodes(ctx, w, node.FirstChild)
			return
		}
	}
	renderTextFormatted(ctx, w, node, "`")
}
