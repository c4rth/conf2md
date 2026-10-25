package confluence_plugin

import (
	"bytes"
	"strings"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/marker"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAsStructuredMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	acName, found := dom.GetAttribute(node, "ac:name")
	log.Debug().Str("name", acName).Msg("Found structured macro")
	if found {
		switch acName {
		case "tab-pane", "horizontal-nav-item":
			log.Warn().Str("name", acName).Msg("TODO")
			return renderMacroTabNav(ctx, w, node)
		case "info", "warning", "note", "tip":
			return renderMacroAdmonition(ctx, w, node, acName)
		case "anchor":
			return renderMacroAnchor(w, node)
		case "excerpt":
			return renderMacroExcerpt(ctx, w, node)
		case "status":
			return renderMacroStatus(w, node)
		case "code":
			return renderMacroCode(w, node) 
		case "expand":
			return renderMacroExpand(ctx, w, node)
		case "gliffy":
			return renderMacroGliffy(w, node)
		case "sp-infobulle":
			return renderMacroSpInfobulle(w, node)
		case "toc":
			return renderMacroToc(acName)
		case "underconstruction":
			return renderMacroUnderConstruction(ctx, w, node, acName)
		case "view-file":
			return renderMacroViewFile(w, node)
		case "tabs-group":
			return renderMacroTabsGroup(acName)
		default:
			if strings.Contains(acName, "nav-grou") {
				return renderMacroTabsGroup(acName)
			}
			log.Warn().Str("name", acName).Msg("Unknown macro")
		}
	}
	return converter.RenderSuccess
}

func renderMacroAnchor(w converter.Writer, node *html.Node) converter.RenderStatus {
	id := getAcParameterText(node, "id")
	w.WriteString("<a id=\"" + id + "\"></a>")
	return converter.RenderSuccess
}

func writeIndentedChildNode(ctx converter.Context, w converter.Writer, node *html.Node) {
	if node != nil {
		var body bytes.Buffer
		ctx.RenderChildNodes(ctx, &body, node)
		content := strings.Trim(body.String(), "\n")
		if content != "" {
			content = strings.ReplaceAll(content, string(marker.BytesMarkerCodeBlockNewline), string(marker.BytesMarkerCodeBlockNewline)+"    ")
			w.WriteString("\n    " + strings.ReplaceAll(content, "\n", "\n    "))
		}
	}
}

func writeAdmonition(ctx converter.Context, w converter.Writer, acName, title string, collapsed bool, richTextNode *html.Node) {
	if collapsed {
		w.WriteString("???")
	} else {
		w.WriteString("!!!")
	}
	w.WriteString(" " + acName + " \"" + title + "\"\n")
	writeIndentedChildNode(ctx, w, richTextNode)
	w.WriteString("\n\n")
}

func renderMacroAdmonition(ctx converter.Context, w converter.Writer, node *html.Node, acName string) converter.RenderStatus {
	title := getAcParameterText(node, "title")
	collapsed := getAcParameterText(node, "collapse") == "true"
	richTextNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:rich-text-body"
	})
	writeAdmonition(ctx, w, acName, title, collapsed, richTextNode)
	return converter.RenderSuccess
}

func renderMacroCode(w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getAcParameterText(node, "title")
	language := getAcParameterText(node, "language")
	collapsed := getAcParameterText(node, "collapse") == "true"
	codeNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "pre"
	})
	var content string
	if codeNode != nil {
		content = dom.CollectText(codeNode)
		content = strings.ReplaceAll(content, "\n", string(marker.BytesMarkerCodeBlockNewline))
	}

	if collapsed {
		if title == "" {
			title = "Code"
		}
		w.WriteString("??? code \"" + title + "\"\n")
		w.WriteString("    ```" + language + "\n")
		if content != "" {
			w.WriteString("    " + strings.ReplaceAll(content, string(marker.BytesMarkerCodeBlockNewline), string(marker.BytesMarkerCodeBlockNewline)+"    "))
		}
		w.WriteString("\n    ```")
	} else {
		if title != "" {
			w.WriteString("**" + title + "**\n\n")
		}
		w.WriteString("```" + language + "\n")
		w.WriteString(content)
		w.WriteString("\n```\n")
	}
	return converter.RenderSuccess
}

func renderMacroGliffy(w converter.Writer, node *html.Node) converter.RenderStatus {
	alt := getAcParameterText(node, "displayName")
	src := getAcParameterText(node, "name")
	w.WriteString("![" + alt + "]" + "(attachments/" + src + ".png)")
	return converter.RenderSuccess
}

func renderMacroUnderConstruction(ctx converter.Context, w converter.Writer, node *html.Node, acName string) converter.RenderStatus {
	w.WriteString("\n")
	richTextNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:rich-text-body"
	})
	writeAdmonition(ctx, w, acName, "This page is under construction", false, richTextNode)
	return converter.RenderSuccess
}

func getBackgroundColor(color string) string {
	switch strings.ToLower(color) {
	case "paleturquoise":
		return "rgb(175, 238, 238)"
	case "oldmace":
		return "rgb(253, 245, 230)"
	case "lightyellow":
		return "rgb(255, 255, 224)"
	case "azure":
		return "rgb(240, 255, 255)"
	case "lightgrey":
		return "rgb(211, 211, 211)"
	case "honeydew":
		return "rgb(240, 255, 240)"
	default:
		return "rgb(245, 245, 245)"
	}
}

func renderMacroViewFile(w converter.Writer, node *html.Node) converter.RenderStatus {
	name := getAcParameterText(node, "name")
	if name != "" {
		attachmentNode := dom.FindFirstNode(node, func(n *html.Node) bool {
			return dom.NodeName(n) == "ri:attachment"
		})
		if attachmentNode != nil {
			filename := getAcParameterText(attachmentNode, "ri:filename")
			w.WriteString("![" + name + "](attachments/" + filename + ")")
		}
	}
	return converter.RenderSuccess
}

func renderMacroSpInfobulle(w converter.Writer, node *html.Node) converter.RenderStatus {
	background := getAcParameterText(node, "Background")
	term := getAcParameterText(node, "Term")
	description := getAcParameterText(node, "Description")
	w.WriteString("<span title=\"" + description + "\" style=\"background-color:" + getBackgroundColor(background) + "\">" + term + "</span>")
	return converter.RenderSuccess
}

func renderMacroTabsGroup(acName string) converter.RenderStatus {	
	log.Warn().Str("name", acName).Msg("Rendering tabs group macro - skipped")
	return converter.RenderSuccess
}

func renderMacroExpand(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getAcParameterText(node, "title")
	firstNode := node.FirstChild
	if firstNode != nil {
		writeAdmonition(ctx, w, "question", title, true, firstNode)
	}
	return converter.RenderSuccess
}

func renderMacroExcerpt(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	hidden := getAcParameterText(node, "hidden")
	if hidden == "true" {
		w.WriteString("<!-- Excerpt hidden -->")
	} else {
		firstNode := node.FirstChild
		if firstNode != nil {
			ctx.RenderChildNodes(ctx, w, firstNode)
		}
	}
	return converter.RenderSuccess
}

func renderMacroStatus(w converter.Writer, node *html.Node) converter.RenderStatus {
	color := getAcParameterText(node, "colour")
	title := getAcParameterText(node, "title")
	if color == "" {
		w.WriteString(title)
	} else {
		w.WriteString("<span style=\"background-color:" + color + "; color: white\">" + title + "</span>")
	}
	return converter.RenderSuccess
}

func renderMacroTabNav(ctx converter.Context, w converter.Writer, node *html.Node,) converter.RenderStatus {
	name := getAcParameterText(node, "name")
	if name == "" {
		name = "Tab"
	}
	w.WriteString("\n=== \"<span style=\"font-size: 1rem\">"+name+"</span>\"")
	writeIndentedChildNode(ctx, w, node.FirstChild)
	return converter.RenderSuccess
}

func renderMacroToc(acName string) converter.RenderStatus {
	log.Warn().Str("name", acName).Msg("Rendering toc macro - skipped")
	return converter.RenderSuccess
}

//

func getAcParameterText(node *html.Node, paramName string) string {
	first := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.GetAttributeOr(n, "ac:name", "") == paramName
	})
	if first == nil {
		return ""
	}
	return dom.CollectText(first)
}
