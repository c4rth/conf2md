package table

import (
	"strings"
	"unicode/utf8"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"golang.org/x/net/html"
)

func (t *tablePlugin) renderTable(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	table := t.collectTableContent(ctx, n)
	if table == nil {
		// Sometimes we just cannot render the table.
		// Either because it is an empty table OR
		// because there are newlines inside the content (which would break the table).
		WriteString(w, "\n**************************\n* Unable to render table *\n**************************\n")
		return converter.RenderTryNext
	}

	// Sometimes we pad the cells with extra spaces (e.g. "| text    |").
	// For that we first need to know the maximum width of every column.
	counts := calculateMaxCounts(table.Rows)

	// Sometimes a row contains less cells that another row.
	// We then fill it up with empty cells (e.g. "| text |     |").
	table.Rows = fillUpRows(table.Rows, len(counts))

	// - - - - - - - - - - - - - - - - - - - - - - - - - - //

	WriteString(w, "\n\n")
	// - - - Header - - - //
	t.writeRow(w, counts, table.Rows[0])
	WriteString(w, "\n")
	t.writeHeaderUnderline(w, table.Alignments, counts)
	WriteString(w, "\n")

	// - - - Body - - - //
	for _, cells := range table.Rows[1:] {
		t.writeRow(w, counts, cells)
		WriteString(w, "\n")
	}

	// - - - Caption - - - //
	if table.Caption != nil {
		WriteString(w, "\n\n")
		Write(w, table.Caption)

	}
	// - - - - - - //
	WriteString(w, "\n\n")

	return converter.RenderSuccess
}

func getAlignmentFor(alignments []string, index int) string {
	if index > len(alignments)-1 {
		return ""
	}

	return alignments[index]
}
func (t *tablePlugin) writeHeaderUnderline(w converter.Writer, alignments []string, counts []int) {
	for i, maxLength := range counts {
		align := getAlignmentFor(alignments, i)

		isFirstCell := i == 0
		if isFirstCell {
			WriteString(w, "|")
		}
		if align == "left" || align == "center" {
			WriteString(w, ":")
		} else {
			WriteString(w, "-")
		}

		if t.cellPaddingBehavior == CellPaddingBehaviorAligned {
			WriteString(w, strings.Repeat("-", maxLength))
		} else {
			WriteString(w, "-")
		}

		if align == "right" || align == "center" {
			WriteString(w, ":")
		} else {
			WriteString(w, "-")
		}
		WriteString(w, "|")
	}
}

func (t *tablePlugin) writeRow(w converter.Writer, counts []int, cells [][]byte) {
	for i, cell := range cells {
		isFirstCell := i == 0
		if isFirstCell {
			WriteString(w, "|")
		}

		currentCount := utf8.RuneCount(cell)
		filler := counts[i] - currentCount

		if t.cellPaddingBehavior == CellPaddingBehaviorAligned || t.cellPaddingBehavior == CellPaddingBehaviorMinimal {
			WriteString(w, " ")
		}

		Write(w, cell)

		if t.cellPaddingBehavior == CellPaddingBehaviorAligned && filler > 0 {
			WriteString(w, strings.Repeat(" ", filler))
		}

		if t.cellPaddingBehavior == CellPaddingBehaviorAligned || t.cellPaddingBehavior == CellPaddingBehaviorMinimal {
			WriteString(w, " ")
		}

		WriteString(w, "|")
	}
}
