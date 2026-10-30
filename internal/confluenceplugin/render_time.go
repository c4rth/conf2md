package confluenceplugin

import (
	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderTime(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	datetime := dom.GetAttributeOr(node, "datetime", "")
	WriteString(w, " <span style=\"padding: 3px; background-color: rgb(235, 236, 240)\">"+datetime+" </span> ")
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}
