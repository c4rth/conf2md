package confluence_plugin

import (
	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAsEmoticon(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	emojiId, found := dom.GetAttribute(node, "ac:emoji-id")
	if found {
		emoji := GetEmoji(emojiId)
		if emoji != "" {
			w.WriteString(emoji + " ")
		}
	}
	return converter.RenderTryNext
}