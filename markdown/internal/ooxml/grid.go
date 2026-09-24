// Package ooxml holds the table-grid helpers shared by the OOXML converters
// (DOCX and PPTX), which both need to emit HTML tables that preserve merged
// cells.
package ooxml

import (
	"strconv"
	"strings"
)

// TableCell is one grid cell of an HTML table. An empty ColSpan or RowSpan
// means one.
type TableCell struct {
	Content string
	ColSpan int
	RowSpan int
}

// TableRow is one row of an HTML table.
type TableRow struct {
	Cells []TableCell
}

// RenderHTMLTable renders rows as an HTML table, emitting the first row as a
// header row and carrying merged cells as colspan/rowspan attributes.
func RenderHTMLTable(rows []TableRow) string {
	if len(rows) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("<table>")
	for rowIndex, row := range rows {
		out.WriteString("\n<tr>")
		tag := "td"
		if rowIndex == 0 {
			tag = "th"
		}
		for _, cell := range row.Cells {
			writeHTMLCell(&out, tag, cell)
		}
		out.WriteString("</tr>")
	}
	out.WriteString("\n</table>")
	return out.String()
}

func writeHTMLCell(out *strings.Builder, tag string, cell TableCell) {
	out.WriteString("<")
	out.WriteString(tag)
	if cell.ColSpan > 1 {
		out.WriteString(` colspan="`)
		out.WriteString(strconv.Itoa(cell.ColSpan))
		out.WriteString(`"`)
	}
	if cell.RowSpan > 1 {
		out.WriteString(` rowspan="`)
		out.WriteString(strconv.Itoa(cell.RowSpan))
		out.WriteString(`"`)
	}
	out.WriteString(">")
	out.WriteString(cell.Content)
	out.WriteString("</")
	out.WriteString(tag)
	out.WriteString(">")
}
