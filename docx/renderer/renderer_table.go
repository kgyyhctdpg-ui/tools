package renderer

import (
	"github.com/88250/lute/ast"
	"github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

func (r *DocxRenderer) renderTableCell(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if r.currentRow != nil {
			cell := (*r.currentRow).AddCell()
			r.currentCell = &cell

			para := cell.AddParagraph()
			cellProps := cell.Properties()
			cellProps.SetVerticalAlignment(wml.ST_VerticalJcCenter)
			cellProps.Borders().SetAll(wml.ST_BorderSingle, color.FromHex(r.config.Table.BorderColor), r.config.Table.BorderWidth)
			para.Properties().SetAlignment(wml.ST_JcCenter)

			run := para.AddRun()
			runProps := run.Properties()
			runProps.SetFontFamily(r.config.Fonts.Content)
			runProps.SetSize(float64(r.config.Table.HeaderFontSize))

			if r.isInTableHead(node) {
				runProps.SetBold(true)

				cellProps := cell.Properties()
				grayColor := color.RGB(217, 217, 217)
				cellProps.SetShading(wml.ST_ShdClear, grayColor, grayColor)
			}
			run.AddText(node.Text())
		}
	} else {
		if r.currentCell != nil {
			r.currentCell = nil
		}
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) isInTableHead(node *ast.Node) bool {
	for parent := node.Parent; nil != parent; parent = parent.Parent {
		if parent.Type == ast.NodeTableHead {
			return true
		}
		if parent.Type == ast.NodeTable {
			break
		}
	}
	return false
}

func (r *DocxRenderer) renderTableRow(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if r.currentTable != nil {
			row := (*r.currentTable).AddRow()
			r.currentRow = &row
			r.currentCell = nil
		}
	} else {
		r.currentRow = nil
		r.currentCell = nil
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderTableHead(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderTable(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		table := r.doc.AddTable()
		table.Properties().SetWidthPercent(100)
		table.Properties().SetAlignment(wml.ST_JcTableCenter)

		borders := table.Properties().Borders()
		borders.SetAll(wml.ST_BorderSingle, color.FromHex(r.config.Table.BorderColor), r.config.Table.BorderWidth)

		r.currentTable = &table
		r.currentRow = nil
		r.currentCell = nil
	} else {
		r.currentTable = nil
		r.currentRow = nil
		r.currentCell = nil
		r.addTableAfterSpacing()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) addTableAfterSpacing() {
	if r.config.Table.AfterLineSpacing <= 0 {
		return
	}
	para := r.doc.AddParagraph()
	para.Properties().Spacing().SetLineSpacing(r.config.Table.AfterLineSpacing, wml.ST_LineSpacingRuleAuto)
}
