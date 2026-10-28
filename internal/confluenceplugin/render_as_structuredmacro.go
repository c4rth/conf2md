package confluenceplugin

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"

	"conf2md/internal/storage"
)

func (cp *confluencePlugin) renderAsStructuredMacro(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	acName, found := dom.GetAttribute(node, "ac:name")
	log.Debug().Str("name", acName).Msg("found structured macro")
	if found {
		switch acName {
		case "info", "warning", "note", "tip", "success", "error":
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
		case "tab-pane", "horizontal-nav-item":
			return renderMacroTabNav(ctx, w, node)
		case "panel":
			return renderMacroPanel(ctx, w, node)
		default:
			if strings.Contains(acName, "nav-grou") {
				return renderMacroTabsGroup(acName)
			}
			log.Warn().Str("name", acName).Msg("unknown macro")
		}
	}
	return converter.RenderSuccess
}

func renderMacroAnchor(w converter.Writer, node *html.Node) converter.RenderStatus {
	id := getAcParameterText(node, "id")
	WriteString(w, "<a id=\""+id+"\"></a>")
	return converter.RenderSuccess
}

func writeIndentedChildNode(ctx converter.Context, w converter.Writer, childNode *html.Node) {
	if childNode != nil {
		var body bytes.Buffer
		ctx.RenderNodes(ctx, &body, childNode)
		content := strings.Trim(body.String(), "\n")
		if content != "" {
			WriteString(w, "    "+strings.ReplaceAll(content, "\n", "\n    "))
		} else {
			log.Error().Msg("no content found for indented child node")
		}
	}
}

func writeAdmonition(ctx converter.Context, w converter.Writer, acName, title string, collapsed bool, richTextNode *html.Node) {
	if collapsed {
		WriteString(w, "???")
	} else {
		WriteString(w, "!!!")
	}
	WriteString(w, " "+acName)
	if strings.TrimSpace(title) != "" {
		WriteString(w, " \""+strings.TrimSpace(title)+"\"\n")
	} else {
		WriteString(w, " \""+strings.ToUpper(acName[0:1])+acName[1:]+"\"\n")
	}
	writeIndentedChildNode(ctx, w, richTextNode)
	WriteString(w, "\n\n")
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
	}

	if collapsed {
		if title == "" {
			title = "Code"
		}
		WriteString(w, "??? code \""+title+"\"\n")
		WriteString(w, "    ```"+language+"\n")
		if content != "" {
			WriteString(w, "    "+strings.ReplaceAll(content, "\n", "\n    "))
		}
		WriteString(w, "\n    ```")
	} else {
		if title != "" {
			WriteString(w, "**"+title+"**\n\n")
		}
		WriteString(w, "```"+language+"\n")
		WriteString(w, content)
		WriteString(w, "\n```\n")
	}
	return converter.RenderSuccess
}

func renderMacroGliffy(w converter.Writer, node *html.Node) converter.RenderStatus {
	alt := getAcParameterText(node, "displayName")
	src := storage.SanitizeFilename(getAcParameterText(node, "name"))
	WriteString(w, "!["+alt+"]"+"(attachments/"+src+".png)")
	return converter.RenderSuccess
}

func renderMacroUnderConstruction(ctx converter.Context, w converter.Writer, node *html.Node, acName string) converter.RenderStatus {
	WriteString(w, "\n")
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
			filename := storage.SanitizeFilename(dom.GetAttributeOr(attachmentNode, "ri:filename", ""))
			WriteString(w, "!["+name+"](attachments/"+filename+")")
		}
	}
	return converter.RenderSuccess
}

func renderMacroSpInfobulle(w converter.Writer, node *html.Node) converter.RenderStatus {
	background := getAcParameterText(node, "Background")
	term := getAcParameterText(node, "Term")
	description := getAcParameterText(node, "Description")
	WriteString(w, "<span title=\""+description+"\" style=\"background-color:"+getBackgroundColor(background)+"\">"+term+"</span>")
	return converter.RenderSuccess
}

func renderMacroTabsGroup(acName string) converter.RenderStatus {
	return converter.RenderTryNext
}

func renderMacroExpand(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	title := getAcParameterText(node, "title")
	richTextNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:rich-text-body"
	})
	if richTextNode != nil {
		writeAdmonition(ctx, w, "question", title, true, richTextNode)
	}
	return converter.RenderSuccess
}

func renderMacroExcerpt(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	hidden := getAcParameterText(node, "hidden")
	if hidden == "true" {
		WriteString(w, "<!-- Excerpt hidden -->")
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
		color = "grey"
	}
	color = transparentColor(color, 0.3)
	WriteString(w, "<mark style=\"background-color:"+color+"; padding: 0.2em;\">"+title+"</mark>")
	return converter.RenderSuccess
}

// statusColors maps the colours offered by the Confluence status macro to the
// Material Design 500 shades.
var statusColors = map[string][3]uint8{
	"grey":   {0x9e, 0x9e, 0x9e},
	"gray":   {0x9e, 0x9e, 0x9e},
	"red":    {0xf4, 0x43, 0x36},
	"yellow": {0xff, 0xeb, 0x3b},
	"green":  {0x4c, 0xaf, 0x50},
	"blue":   {0x21, 0x96, 0xf3},
	"purple": {0x9c, 0x27, 0xb0},
}

// transparentColor returns a named or hex colour as rgba() with the given alpha (0 is fully
// transparent, 1 is opaque). Unrecognised colours are returned unchanged.
func transparentColor(color string, alpha float64) string {
	rgb, ok := parseColor(color)
	if !ok {
		return color
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", rgb[0], rgb[1], rgb[2], strconv.FormatFloat(alpha, 'f', -1, 64))
}

func parseColor(color string) ([3]uint8, bool) {
	color = strings.ToLower(strings.TrimSpace(color))
	if rgb, ok := statusColors[color]; ok {
		return rgb, true
	}
	hex, ok := strings.CutPrefix(color, "#")
	if !ok {
		return [3]uint8{}, false
	}
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return [3]uint8{}, false
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return [3]uint8{}, false
	}
	return [3]uint8{uint8(value >> 16), uint8(value >> 8), uint8(value)}, true
}

func renderMacroTabNav(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	name := getAcParameterText(node, "name")
	if name == "" {
		name = "Tab"
	}
	WriteString(w, "\n=== \"<span style=\"font-size: 1rem\">"+name+"</span>\"")
	richTextNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:rich-text-body"
	})
	if richTextNode != nil {
		writeIndentedChildNode(ctx, w, richTextNode)
		WriteString(w, "\n")
	}
	return converter.RenderSuccess
}

func renderMacroToc(acName string) converter.RenderStatus {
	log.Warn().Str("name", acName).Msg("rendering toc macro - skipped")
	return converter.RenderSuccess
}

func renderMacroPanel(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
	richTextNode := dom.FindFirstNode(node, func(n *html.Node) bool {
		return dom.NodeName(n) == "ac:rich-text-body"
	})
	if richTextNode != nil {
		writeAdmonition(ctx, w, "quote", " ", false, richTextNode)
	}
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
