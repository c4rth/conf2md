package confluence_plugin

import (
	"github.com/JohannesKaufmann/dom"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func (cp *confluencePlugin) renderAsTaskList(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
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
		w.WriteString(" - [")
		if status {
			w.WriteString("x")
		} else {
			w.WriteString(" ")
		}
		w.WriteString("]   ")
		ctx.RenderChildNodes(ctx, w, taskBody)
		w.WriteString("\n")
	}
	return converter.RenderSuccess
}