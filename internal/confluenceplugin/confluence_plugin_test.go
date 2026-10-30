package confluenceplugin

import (
	"bytes"
	"testing"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func TestRenderBlockNilNode(t *testing.T) {
	plugin := &confluencePlugin{}
	if got := plugin.renderBlock(nil, nil, nil); got != converter.RenderSuccess {
		t.Errorf("renderBlock(nil) = %v, want %v", got, converter.RenderSuccess)
	}
}

func TestHandleRenderTaskNodes(t *testing.T) {
	plugin := &confluencePlugin{}
	for _, name := range []string{"ac:task", "ac:task-body"} {
		t.Run(name, func(t *testing.T) {
			node := &html.Node{Type: html.ElementNode, Data: name}
			if got := plugin.handleRender(nil, nil, node); got != converter.RenderTryNext {
				t.Errorf("handleRender(%q) = %v, want %v", name, got, converter.RenderTryNext)
			}
		})
	}
}

func TestRenderAsImageSanitizesAttachmentFilename(t *testing.T) {
	node := elementNode("ac:image", nil,
		elementNode("ri:attachment", []html.Attribute{{Key: "ri:filename", Val: "My Diagram.png"}}),
	)
	var output bytes.Buffer

	(&confluencePlugin{}).renderAsImage(nil, &output, node)

	if got, want := output.String(), "![](attachments/my-diagram.png)"; got != want {
		t.Errorf("renderAsImage() = %q, want %q", got, want)
	}
}

func TestRenderMacroGliffySanitizesAttachmentFilename(t *testing.T) {
	node := elementNode("ac:structured-macro", nil,
		parameterNode("name", "My Diagram"),
		parameterNode("displayName", "Flow"),
	)
	var output bytes.Buffer

	renderMacroGliffy(&output, node)

	if got, want := output.String(), "![Flow](attachments/my-diagram.png)"; got != want {
		t.Errorf("renderMacroGliffy() = %q, want %q", got, want)
	}
}

func TestRenderMacroViewFileSanitizesAttachmentFilename(t *testing.T) {
	node := elementNode("ac:structured-macro", nil,
		parameterNode("name", "Document"),
		elementNode("ri:attachment", []html.Attribute{{Key: "ri:filename", Val: "My File.pdf"}}),
	)
	var output bytes.Buffer

	renderMacroViewFile(&output, node)

	if got, want := output.String(), "![Document](attachments/my-file.pdf)"; got != want {
		t.Errorf("renderMacroViewFile() = %q, want %q", got, want)
	}
}

func parameterNode(name, value string) *html.Node {
	return elementNode("ac:parameter", []html.Attribute{{Key: "ac:name", Val: name}},
		&html.Node{Type: html.TextNode, Data: value},
	)
}

func elementNode(name string, attributes []html.Attribute, children ...*html.Node) *html.Node {
	node := &html.Node{Type: html.ElementNode, Data: name, Attr: attributes}
	for _, child := range children {
		node.AppendChild(child)
	}
	return node
}

func TestRenderMacroStatusUsesTransparentBackgroundColor(t *testing.T) {
	tests := []struct {
		colour string
		want   string
	}{
		{"Green", `<mark style="background-color:rgba(76, 175, 80, 0.3); padding: 0.2em;">Done</mark>`},
		{"", `<mark style="background-color:rgba(158, 158, 158, 0.3); padding: 0.2em;">Done</mark>`},
		{"Unknown", `<mark style="background-color:Unknown; padding: 0.2em;">Done</mark>`},
	}
	for _, tt := range tests {
		t.Run(tt.colour, func(t *testing.T) {
			node := elementNode("ac:structured-macro", nil,
				parameterNode("colour", tt.colour),
				parameterNode("title", "Done"),
			)
			var output bytes.Buffer

			renderMacroStatus(&output, node)

			if got := output.String(); got != tt.want {
				t.Errorf("renderMacroStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTransparentColor(t *testing.T) {
	tests := []struct {
		color string
		alpha float64
		want  string
	}{
		{"Grey", 0.5, "rgba(158, 158, 158, 0.5)"},
		{"Red", 0.5, "rgba(244, 67, 54, 0.5)"},
		{"Yellow", 0.5, "rgba(255, 235, 59, 0.5)"},
		{"Green", 0.5, "rgba(76, 175, 80, 0.5)"},
		{"blue", 0.5, "rgba(33, 150, 243, 0.5)"},
		{"PURPLE", 0.5, "rgba(156, 39, 176, 0.5)"},
		{"#123456", 0.25, "rgba(18, 52, 86, 0.25)"},
		{"#f00", 1, "rgba(255, 0, 0, 1)"},
		{"#12345", 0.5, "#12345"},
		{"#zzzzzz", 0.5, "#zzzzzz"},
		{"orange", 0.5, "orange"},
	}
	for _, tt := range tests {
		if got := transparentColor(tt.color, tt.alpha); got != tt.want {
			t.Errorf("transparentColor(%q, %v) = %q, want %q", tt.color, tt.alpha, got, tt.want)
		}
	}
}
