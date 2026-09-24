package xlsx

import (
	"bytes"
	"context"
	"fmt"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"html"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// NewConverter builds the XLSX converter.
func NewConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".xlsx", ".xlsm", ".xltx", ".xltm"},
		[]string{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-excel.sheet.macroenabled.12"},
		func(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
			workbook, err := excelize.OpenReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("markdown: open spreadsheet: %w", err)
			}
			defer workbook.Close()

			blocks := make([]string, 0)
			for _, sheet := range workbook.GetSheetList() {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				rows, err := workbook.GetRows(sheet)
				if err != nil {
					return nil, fmt.Errorf("markdown: read sheet %q: %w", sheet, err)
				}
				merges, err := workbook.GetMergeCells(sheet)
				if err != nil {
					return nil, fmt.Errorf("markdown: read merged cells in %q: %w", sheet, err)
				}
				tableFormat := core.TableFormatFromContext(ctx)
				table := core.MarkdownTable(rows)
				if len(merges) > 0 {
					table = renderXLSXHTMLTable(rows, merges)
				} else if tableFormat == core.TableFormatHTML {
					table = core.HTMLTable(rows)
				}
				if table == "" {
					continue
				}
				blocks = append(blocks, "## "+sheet+"\n\n"+table)
			}
			return &core.Result{Markdown: core.JoinBlocks(blocks)}, nil
		},
	)
}

type xlsxSpan struct {
	rowSpan int
	colSpan int
}

func renderXLSXHTMLTable(rows [][]string, mergeCells []excelize.MergeCell) string {
	spans := make(map[[2]int]xlsxSpan)
	covered := make(map[[2]int]bool)
	maxColumns := 0
	for _, row := range rows {
		if len(row) > maxColumns {
			maxColumns = len(row)
		}
	}
	for index := range mergeCells {
		merge := &mergeCells[index]
		startColumn, startRow, startErr := excelize.CellNameToCoordinates(merge.GetStartAxis())
		endColumn, endRow, endErr := excelize.CellNameToCoordinates(merge.GetEndAxis())
		if startErr != nil || endErr != nil {
			continue
		}
		spans[[2]int{startRow - 1, startColumn - 1}] = xlsxSpan{rowSpan: endRow - startRow + 1, colSpan: endColumn - startColumn + 1}
		for row := startRow - 1; row < endRow; row++ {
			for column := startColumn - 1; column < endColumn; column++ {
				if row != startRow-1 || column != startColumn-1 {
					covered[[2]int{row, column}] = true
				}
			}
		}
		if endColumn > maxColumns {
			maxColumns = endColumn
		}
	}

	var out strings.Builder
	out.WriteString("<table>\n")
	for rowIndex, row := range rows {
		out.WriteString("<tr>")
		for column := 0; column < maxColumns; column++ {
			key := [2]int{rowIndex, column}
			if covered[key] {
				continue
			}
			tag := "td"
			if rowIndex == 0 {
				tag = "th"
			}
			out.WriteString("<" + tag)
			if span := spans[key]; span.colSpan > 0 {
				if span.colSpan > 1 {
					out.WriteString(` colspan="` + strconv.Itoa(span.colSpan) + `"`)
				}
				if span.rowSpan > 1 {
					out.WriteString(` rowspan="` + strconv.Itoa(span.rowSpan) + `"`)
				}
			}
			out.WriteString(">")
			if column < len(row) {
				out.WriteString(html.EscapeString(row[column]))
			}
			out.WriteString("</" + tag + ">")
		}
		out.WriteString("</tr>\n")
	}
	out.WriteString("</table>")
	return out.String()
}
