package confluenceplugin

import (
	"conf2md/internal/storage"
	"strings"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAsLink(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	// attachment
	linkType := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ri:attachment"
	})
	if linkType != nil {
		name := dom.GetAttributeOr(linkType, "ri:filename", "")
		WriteString(w, "["+name+"](attachments/"+storage.SanitizeFilename(name)+")")
		return converter.RenderSuccess
	}
	// page
	linkType = dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ri:page"
	})
	if linkType != nil {
		anchor := dom.GetAttributeOr(linkType, "ac:anchor", "")
		if anchor != "" {
			anchor = "#" + strings.ReplaceAll(anchor, " ", "-")
		}
		title := dom.GetAttributeOr(linkType, "ri:content-title", "")
		bodyLink := dom.FindFirstNode(node, func(n *html.Node) bool {
			return dom.NodeName(n) == "ac:plain-text-link-body"
		})
		link := title
		if bodyLink != nil {
			link = bodyLink.Data
		}
		WriteString(w, "["+link+"]("+title+anchor+")")
		return converter.RenderSuccess
	}
	// user
	linkType = dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ri:user"
	})
	if linkType != nil {
		log.Warn().Msg("user link NOT supported")
		return converter.RenderSuccess
	}
	// anchor
	anchor := dom.GetAttributeOr(node, "ac:anchor", "")
	if anchor != "" {
		bodyLink := dom.FindFirstNode(node, func(n *html.Node) bool {
			return dom.NodeName(n) == "ac:plain-text-link-body"
		})
		link := ""
		if bodyLink != nil {
			link = bodyLink.Data
		}
		WriteString(w, "["+link+"]("+"#"+strings.ReplaceAll(anchor, " ", "-")+")")
		return converter.RenderSuccess
	}

	return converter.RenderSuccess
}
