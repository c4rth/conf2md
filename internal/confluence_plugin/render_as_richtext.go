package confluence_plugin

import (
	"golang.org/x/net/html"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

func (cp *confluencePlugin) renderAsRichText(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}