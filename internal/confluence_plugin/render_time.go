package confluence_plugin

import (
	"golang.org/x/net/html"
	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

func (cp *confluencePlugin) renderTime(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	datetime := dom.GetAttributeOr(node, "datetime", "")
	w.WriteString("<span style=\"font-size: 0.9rem; padding: 3px; background-color: rgb(235, 236, 240)\">" + datetime + " </span> ")
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}
