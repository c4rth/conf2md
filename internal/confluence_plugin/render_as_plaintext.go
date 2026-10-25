package confluence_plugin

import (
	"golang.org/x/net/html"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/rs/zerolog/log"
)

func (cp *confluencePlugin) renderAsPlainText(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	log.Debug().Msg("********************************")
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}