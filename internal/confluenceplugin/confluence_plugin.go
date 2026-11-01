package confluenceplugin

import (
	"bytes"
	"strings"

	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

const (
	leadingHashMarker    = "CONF2MDLEADINGHASH"
)

type confluencePlugin struct {
}

func NewConfluencePlugin() converter.Plugin {
	plugin := &confluencePlugin{}
	return plugin
}

func (cp *confluencePlugin) Name() string {
	return "confluence"
}

func (cp *confluencePlugin) Init(conv *converter.Converter) error {
	conv.Register.Renderer(cp.handleRender, converter.PriorityEarly)
	conv.Register.TextTransformer(cp.textTransform, converter.PriorityEarly)
	conv.Register.PostRenderer(cp.restoreLeadingHashes, converter.PriorityLate)
	return nil
}

func (cp *confluencePlugin) textTransform(ctx converter.Context, content string) string {
	if strings.HasPrefix(content, "#") {
		content = leadingHashMarker + content[1:]
	}
	return content
}

func (cp *confluencePlugin) handleRender(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	name := dom.NodeName(n)
	log.Debug().Str("name", name).Msg("handling node")
	switch name {
	case "ac:plain-text-body":
		return cp.renderAsPlainText(ctx, w, n)
	case "ac:rich-text-body":
		return cp.renderAsRichText(ctx, w, n)
	case "ac:emoticon":
		return cp.renderAsEmoticon(ctx, w, n)
	case "ac:link":
		return cp.renderAsLink(ctx, w, n)
	case "ac:image":
		return cp.renderAsImage(ctx, w, n)
	case "ac:inline-comment-marker":
		return cp.renderInlineCommentMarker(ctx, w, n)
	case "ac:task-list":
		return cp.renderAsTaskList(ctx, w, n)
	case "ac:parameter":
		return converter.RenderSuccess
	case "ac:layout", "ac:layout-cell", "ac:layout-section":
		return converter.RenderTryNext
	case "ac:structured-macro":
		return cp.renderAsStructuredMacro(ctx, w, n)
	case "time":
		return cp.renderTime(ctx, w, n)
	default:
		if cp.isTextFormatTag(name) {
			return cp.renderTextFormatting(ctx, w, n, name)
		}
		if strings.HasPrefix(name, "ac:adf") {
			return cp.renderAdf(ctx, w, n, name)
		}
		if strings.HasPrefix(name, "ac:") || strings.HasPrefix(name, "ri:") {
			log.Warn().Str("name", name).Msg("unknown tag type")
			return converter.RenderTryNext
		}
	}
	return converter.RenderTryNext
}

func (cp *confluencePlugin) restoreLeadingHashes(_ converter.Context, content []byte) []byte {
	return bytes.ReplaceAll(content, []byte(leadingHashMarker), []byte(`\#`))
}

func (cp *confluencePlugin) renderInlineCommentMarker(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	ctx.RenderChildNodes(ctx, w, n)
	return converter.RenderSuccess
}

func WriteString(w converter.Writer, s string) {
	_, _ = w.WriteString(s)
}
