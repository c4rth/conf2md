package confluenceplugin

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
