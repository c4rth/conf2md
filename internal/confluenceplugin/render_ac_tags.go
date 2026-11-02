package confluenceplugin

import (
	"strings"

	"conf2md/internal/storage"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/rs/zerolog/log"

	"golang.org/x/net/html"
	
)

func (cp *confluencePlugin) renderAcEmoticon(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	emojiId, found := dom.GetAttribute(node, "ac:emoji-id")
	if found {
		emoji := GetEmoji(emojiId)
		if emoji != "" {
			WriteString(w, emoji+" ")
		}
		return converter.RenderSuccess
	}
	return converter.RenderTryNext
}

func (cp *confluencePlugin) renderAcImage(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
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


func (cp *confluencePlugin) renderAcLink(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
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

func (cp *confluencePlugin) renderAcPlainText(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderAcRichText(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderAcTaskList(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	for task := range node.ChildNodes() {
		taskStatus := dom.FindFirstNode(task, func(n *html.Node) bool {
			return dom.NodeName(n) == "ac:task-status"
		})
		if taskStatus == nil {
			continue
		}
		status := dom.CollectText(taskStatus) == "complete"
		taskBody := dom.FindFirstNode(task, func(n *html.Node) bool {
			return dom.NodeName(n) == "ac:task-body"
		})
		WriteString(w, " - [")
		if status {
			WriteString(w, "x")
		} else {
			WriteString(w, " ")
		}
		WriteString(w, "]   ")
		ctx.RenderChildNodes(ctx, w, taskBody)
		WriteString(w, "\n")
	}
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderInlineCommentMarker(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, n)
	return converter.RenderSuccess
}