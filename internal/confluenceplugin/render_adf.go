package confluenceplugin

import (
	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAdf(ctx converter.Context, w converter.Writer, node *html.Node, acName string) converter.RenderStatus {
	switch acName {
	case "ac:adf-content", "ac:adf-extension-frame":
		return cp.renderAdfContent(ctx, w, node)
	case "ac:adf-extension":
		return cp.renderAdfExtension(ctx, w, node)
	default:
		log.Warn().Str("name", acName).Msg("unknown ADF node")
		return converter.RenderTryNext
	}
}

func (cp *confluencePlugin) renderAdfContent(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, node)
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderAdfExtension(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	adfNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-node"
	})
	if adfNode != nil {
		nodeType, found := dom.GetAttribute(adfNode, "type")
		if found {
			log.Debug().Str("type", nodeType).Msg("adf-extension - rendering node")
			switch nodeType {
			case "panel":
				return cp.renderAdfPanel(ctx, w, adfNode)
			case "decision-list":
				return cp.renderAdfDecisionList(ctx, w, adfNode)
			case "multi-bodied-extension":
				return cp.renderAdfMultiBodiedExtension(ctx, w, adfNode)
			default:
				log.Warn().Str("type", nodeType).Msg("adf-extension - node type not handled")
			}
		} else {
			log.Warn().Str("name", dom.NodeName(adfNode)).Msg("adf-extension - ADF node type NOT found")
		}
	} else {
		log.Warn().Str("name", dom.NodeName(node)).Msg("adf-extension - ADF node not found")
	}
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderAdfPanel(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	panelType := getAdfAttributeKey(node, "panel-type")
	adfContent := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-content"
	})
	if adfContent != nil {
		if panelType == "" {
			panelType = "info"
		}
		writeAdmonition(ctx, w, panelType, " ", false, adfContent)
	}
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderAdfDecisionList(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	itemNodes := dom.FindAllNodes(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-node" && dom.GetAttributeOr(n, "type", "") == "decision-item"
	})
	for _, item := range itemNodes {
		adfContent := dom.FindFirstNode(item, func(n *html.Node) bool {
			return dom.NodeName(n) == "ac:adf-content"
		})
		status := getAdfAttributeKey(item, "state") == "DECIDED"
		WriteString(w, " - [")
		if status {
			WriteString(w, "x")
		} else {
			WriteString(w, " ")
		}
		WriteString(w, "]   ")
		ctx.RenderChildNodes(ctx, w, adfContent)
		WriteString(w, "\n")
	}
	return converter.RenderSuccess
}

func (cp *confluencePlugin) renderAdfMultiBodiedExtension(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	extensionKey := getAdfAttributeKey(node, "extension-key")
	if extensionKey != "native-tabs" {
		log.Warn().Str("type", extensionKey).Msg("adf-extension - node type not handled")
		return converter.RenderTryNext
	}
	parameters := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-attribute" && dom.GetAttributeOr(n, "key", "") == "parameters"
	})
	if parameters == nil {
		log.Warn().Str("type", extensionKey).Msg("adf-extension - parameters not found")
		return converter.RenderTryNext
	}
	tabsNode := dom.FindFirstNode(parameters, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-parameter" && dom.GetAttributeOr(n, "key", "") == "tabs"
	})
	if tabsNode == nil {
		log.Warn().Str("type", extensionKey).Msg("adf-extension - tabs not found")
		return converter.RenderTryNext
	}
	tabsTitle := dom.FindAllNodes(tabsNode, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-parameter-value"
	})
	contentNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-content"
	})
	if contentNode == nil {
		log.Warn().Str("type", extensionKey).Msg("adf-extension - content not found")
		return converter.RenderTryNext
	}
	tabsContent := dom.FindAllNodes(contentNode, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-extension-frame"
	})

	for i, tab := range tabsTitle {
		title := getAdfParameterKey(tab, "title")
		WriteString(w, "\n=== \""+title+"\"\n")
		if i < len(tabsContent) {
			writeIndentedChildNode(ctx, w, tabsContent[i])
			WriteString(w, "\n")
		}
	}
	return converter.RenderSuccess
}

//

func getAdfAttributeKey(node *html.Node, paramName string) string {
	first := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-attribute" && dom.GetAttributeOr(n, "key", "") == paramName
	})
	if first == nil {
		return ""
	}
	return dom.CollectText(first)
}

func getAdfParameterKey(node *html.Node, paramName string) string {
	first := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:adf-parameter" && dom.GetAttributeOr(n, "key", "") == paramName
	})
	if first == nil {
		return ""
	}
	return dom.CollectText(first)
}
