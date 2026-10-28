package confluenceplugin

import (
	"conf2md/internal/storage"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAsImage(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	altName := dom.GetAttributeOr(node, "ac:alt", "")
	attachment := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ri:attachment"
	})
	src := ""
	if attachment != nil {
		src = storage.SanitizeFilename(dom.GetAttributeOr(attachment, "ri:filename", ""))
	}
	title := dom.GetAttributeOr(node, "ac:title", "")
	width := dom.GetAttributeOr(node, "ac:width", "")
	height := dom.GetAttributeOr(node, "ac:height", "")
	stylePart := ""
	if width != "" || height != "" {
		stylePart = "{: style=\""
		if width != "" {
			stylePart += "width: " + width + "px;"
		}
		if height != "" {
			stylePart += "height: " + height + "px;"
		}
		stylePart += "\"}"
	}
	if title != "" {
		title = " \"" + title + "\""
	}
	WriteString(w, "!["+altName+"]"+"(attachments/"+src+title+")"+stylePart)
	return converter.RenderSuccess
}
