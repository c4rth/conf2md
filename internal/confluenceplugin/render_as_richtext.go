package confluenceplugin

import (
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAsRichText(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}
