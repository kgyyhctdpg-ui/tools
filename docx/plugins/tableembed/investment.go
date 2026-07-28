package tableembed

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/common"

	"github.com/scoming-dev/tools/docx/document"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
	"github.com/spf13/cast"
)

const investmentHeaderShading = "#D9D9D9"

// InvestmentEstimatePlugin renders the legacy investment-estimate JSON table.
type InvestmentEstimatePlugin struct {
	tagType string
}

// InvestmentEstimate returns the legacy investment-estimate preset plugin.
func InvestmentEstimate() docx.HTMLBlockPlugin {
	return NewInvestmentEstimatePlugin("tzgs")
}

// NewInvestmentEstimatePlugin creates an investment-estimate plugin for a custom s-tag type.
func NewInvestmentEstimatePlugin(tagType string) *InvestmentEstimatePlugin {
	return &InvestmentEstimatePlugin{tagType: tagType}
}

// Type returns the s-tag type handled by this plugin.
func (p *InvestmentEstimatePlugin) Type() string {
	return p.tagType
}

// RenderHTML renders the JSON payload inside the matched s-tag.
func (p *InvestmentEstimatePlugin) RenderHTML(renderer *docx.DocxRenderer, content string) error {
	builder := &investmentEstimateRenderer{DocxRenderer: renderer}
	return builder.render(tagContent(content))
}

func tagContent(content string) string {
	result := common.GetNodeValue(content, "s-tag", "")
	result = strings.TrimPrefix(result, "\n")
	result = strings.TrimSuffix(result, "\n")
	result = strings.TrimPrefix(result, "\r\n")
	result = strings.TrimSuffix(result, "\r\n")
	return result
}

type investmentEstimateRenderer struct {
	*docx.DocxRenderer
	gcPrice     float64
	sbPrice     float64
	otherPrice  float64
	totalPrice  float64
	firstPrice  float64
	secondPrice float64
}

func (r *investmentEstimateRenderer) render(content string) error {
	data := make(map[string]any)
	if content == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return fmt.Errorf("解析投资估算表 JSON 失败: %w", err)
	}

	// Keep the legacy section behavior so existing templates render the same page orientation.
	paperSize := common.PaperSizeA4
	common.AddSectionBreakWithOrientation(r.Document(), common.OrientationV)
	r.createInvestmentTable(data)
	common.AddSectionBreakWithOrientation(r.Document(), common.OrientationH, paperSize)
	return nil
}

func (r *investmentEstimateRenderer) createInvestmentTable(data map[string]any) {
	table := r.Document().AddTable()
	table.Properties().SetWidthPercent(100)
	table.Properties().SetAlignment(wml.ST_JcTableCenter)

	borders := table.Properties().Borders()
	config := r.Config()
	borders.SetAll(wml.ST_BorderSingle, color.FromHex(config.Table.BorderColor), config.Table.BorderWidth)

	r.createTableHeader(table)

	r.totalPrice = cast.ToFloat64(data["建设总投资总计"])
	if engineeringCosts, ok := data["工程费用"].([]any); ok {
		r.createFirstCostsSection(table, engineeringCosts)
	}
	if otherCosts, ok := data["其他费用"].([]any); ok {
		r.createSecondCostsSection(table, otherCosts)
	}
	r.createTotalRow(table)
}

func (r *investmentEstimateRenderer) addTableCell(
	cell document.Cell,
	text string,
	verticalMerge wml.ST_Merge,
	horizontalMerge int64,
	shadingColor string,
	bold bool,
) {
	cell.Properties().SetVerticalAlignment(wml.ST_VerticalJcCenter)
	cell.Properties().SetVerticalMerge(verticalMerge)
	config := r.Config()
	cell.Properties().Borders().SetAll(wml.ST_BorderSingle, color.FromHex(config.Table.BorderColor), config.Table.BorderWidth)
	if horizontalMerge > 0 {
		tcPr := cell.Properties().X()
		tcPr.GridSpan = &wml.CT_DecimalNumber{ValAttr: horizontalMerge}
	}
	para := cell.AddParagraph()
	para.Properties().SetAlignment(wml.ST_JcCenter)
	if shadingColor != "" {
		cell.Properties().SetShading(wml.ST_ShdClear, color.FromHex(shadingColor), color.FromHex(shadingColor))
	}
	run := para.AddRun()
	if bold {
		run.Properties().SetBold(true)
	}
	run.Properties().SetFontFamily(config.Fonts.Content)
	run.Properties().SetSize(float64(config.Table.HeaderFontSize))
	run.Properties().SetCharacterSpacing(config.Table.CellCharacterSpacing)
	run.AddText(text)
}

func (r *investmentEstimateRenderer) createTableHeader(table document.Table) {
	row1 := table.AddRow()
	cell1 := row1.AddCell()
	r.addTableCell(cell1, "序号", wml.ST_MergeRestart, 0, investmentHeaderShading, true)

	cell2 := row1.AddCell()
	r.addTableCell(cell2, "项目名称", wml.ST_MergeRestart, 0, investmentHeaderShading, true)

	cell3 := row1.AddCell()
	r.addTableCell(cell3, "估算价值(万元)", wml.ST_MergeRestart, 3, investmentHeaderShading, true)

	cell4 := row1.AddCell()
	r.addTableCell(cell4, "合计", wml.ST_MergeRestart, 0, investmentHeaderShading, true)

	cell5 := row1.AddCell()
	r.addTableCell(cell5, "技术经济指标", wml.ST_MergeRestart, 3, investmentHeaderShading, true)

	cell6 := row1.AddCell()
	r.addTableCell(cell6, "备注", wml.ST_MergeRestart, 0, investmentHeaderShading, true)

	cell7 := row1.AddCell()
	r.addTableCell(cell7, "比例", wml.ST_MergeRestart, 0, investmentHeaderShading, true)

	row2 := table.AddRow()
	cell8 := row2.AddCell()
	r.addTableCell(cell8, "", wml.ST_MergeContinue, 0, investmentHeaderShading, true)

	cell9 := row2.AddCell()
	r.addTableCell(cell9, "", wml.ST_MergeContinue, 0, investmentHeaderShading, true)

	cell10 := row2.AddCell()
	r.addTableCell(cell10, "建安工程", wml.ST_MergeUnset, 0, investmentHeaderShading, true)

	cell11 := row2.AddCell()
	r.addTableCell(cell11, "设备及工器具购置", wml.ST_MergeUnset, 0, investmentHeaderShading, true)

	cell12 := row2.AddCell()
	r.addTableCell(cell12, "其他费用", wml.ST_MergeUnset, 0, investmentHeaderShading, true)

	cell13 := row2.AddCell()
	r.addTableCell(cell13, "", wml.ST_MergeContinue, 0, investmentHeaderShading, true)

	cell14 := row2.AddCell()
	r.addTableCell(cell14, "单位", wml.ST_MergeUnset, 0, investmentHeaderShading, true)

	cell15 := row2.AddCell()
	r.addTableCell(cell15, "负荷或工程量", wml.ST_MergeUnset, 0, investmentHeaderShading, true)

	cell16 := row2.AddCell()
	r.addTableCell(cell16, "单位指标(元/单位)", wml.ST_MergeUnset, 0, investmentHeaderShading, true)

	cell17 := row2.AddCell()
	r.addTableCell(cell17, "", wml.ST_MergeContinue, 0, investmentHeaderShading, true)

	cell18 := row2.AddCell()
	r.addTableCell(cell18, "", wml.ST_MergeContinue, 0, investmentHeaderShading, true)
}

func (r *investmentEstimateRenderer) createFirstCostsSection(table document.Table, costs []any) {
	price1 := 0.0
	price2 := 0.0
	price3 := 0.0
	price4 := 0.0
	for _, cost := range costs {
		if costMap, ok := cost.(map[string]any); ok {
			switch cast.ToString(costMap["类型"]) {
			case "建安工程":
				if cast.ToString(costMap["子项"]) == "0" {
					price1 += cast.ToFloat64(costMap["金额"])
					r.gcPrice = price1
				}
			case "设备及工器具购置":
				if cast.ToString(costMap["子项"]) == "0" {
					price2 += cast.ToFloat64(costMap["金额"])
					r.sbPrice = price2
				}
			case "其他费用":
				if cast.ToString(costMap["子项"]) == "0" {
					price3 += cast.ToFloat64(costMap["金额"])
					r.otherPrice = price3
				}
			}
		}
	}
	price4 = price1 + price2 + price3
	r.firstPrice = price4

	row := table.AddRow()
	cell1 := row.AddCell()
	r.addTableCell(cell1, "一", wml.ST_MergeUnset, 0, "", true)

	cell2 := row.AddCell()
	r.addTableCell(cell2, "工程费用", wml.ST_MergeUnset, 0, "", true)

	cell3 := row.AddCell()
	r.addTableCell(cell3, fmt.Sprintf("%.2f", price1/10000), wml.ST_MergeUnset, 0, "", true)

	cell4 := row.AddCell()
	r.addTableCell(cell4, fmt.Sprintf("%.2f", price2/10000), wml.ST_MergeUnset, 0, "", true)

	cell5 := row.AddCell()
	r.addTableCell(cell5, fmt.Sprintf("%.2f", price3/10000), wml.ST_MergeUnset, 0, "", true)

	cell6 := row.AddCell()
	r.addTableCell(cell6, fmt.Sprintf("%.2f", price4/10000), wml.ST_MergeUnset, 0, "", true)

	cell7 := row.AddCell()
	r.addTableCell(cell7, "", wml.ST_MergeUnset, 0, "", true)
	cell8 := row.AddCell()
	r.addTableCell(cell8, "", wml.ST_MergeUnset, 0, "", true)
	cell9 := row.AddCell()
	r.addTableCell(cell9, "", wml.ST_MergeUnset, 0, "", true)

	cell10 := row.AddCell()
	r.addTableCell(cell10, "", wml.ST_MergeUnset, 0, "", true)

	cell11 := row.AddCell()
	price0 := (price4) / r.totalPrice
	r.addTableCell(cell11, fmt.Sprintf("%.2f", price0*100)+"%", wml.ST_MergeUnset, 0, "", true)

	for _, cost := range costs {
		if costMap, ok := cost.(map[string]any); ok {
			r.addCostDetailRow(table, costMap)
		}
	}
}

func (r *investmentEstimateRenderer) createSecondCostsSection(table document.Table, costs []any) {
	price1 := 0.0
	price2 := 0.0
	price3 := 0.0
	price4 := 0.0
	for _, cost := range costs {
		if costMap, ok := cost.(map[string]any); ok {
			switch cast.ToString(costMap["类型"]) {
			case "建安工程":
				if cast.ToString(costMap["子项"]) == "0" {
					price1 += cast.ToFloat64(costMap["金额"])
					r.gcPrice = price1
				}
			case "设备及工器具购置":
				if cast.ToString(costMap["子项"]) == "0" {
					price2 += cast.ToFloat64(costMap["金额"])
					r.sbPrice = price2
				}
			case "其他费用":
				if cast.ToString(costMap["子项"]) == "0" {
					price3 += cast.ToFloat64(costMap["金额"])
					r.otherPrice = price3
				}
			}
		}
	}
	price4 = price1 + price2 + price3
	r.secondPrice = price4

	row := table.AddRow()
	cell1 := row.AddCell()
	r.addTableCell(cell1, "二", wml.ST_MergeUnset, 0, "", true)
	cell2 := row.AddCell()
	r.addTableCell(cell2, "其他费用", wml.ST_MergeUnset, 0, "", true)
	cell3 := row.AddCell()
	r.addTableCell(cell3, fmt.Sprintf("%.2f", price1/10000), wml.ST_MergeUnset, 0, "", true)
	cell4 := row.AddCell()
	r.addTableCell(cell4, fmt.Sprintf("%.2f", price2/10000), wml.ST_MergeUnset, 0, "", true)
	cell5 := row.AddCell()
	r.addTableCell(cell5, fmt.Sprintf("%.2f", price3/10000), wml.ST_MergeUnset, 0, "", true)
	cell6 := row.AddCell()
	r.addTableCell(cell6, fmt.Sprintf("%.2f", price4/10000), wml.ST_MergeUnset, 0, "", true)
	cell7 := row.AddCell()
	r.addTableCell(cell7, "", wml.ST_MergeUnset, 0, "", true)
	cell8 := row.AddCell()
	r.addTableCell(cell8, "", wml.ST_MergeUnset, 0, "", true)
	cell9 := row.AddCell()
	r.addTableCell(cell9, "", wml.ST_MergeUnset, 0, "", true)
	cell10 := row.AddCell()
	r.addTableCell(cell10, "", wml.ST_MergeUnset, 0, "", true)
	cell11 := row.AddCell()
	price0 := (price4) / r.totalPrice
	r.addTableCell(cell11, fmt.Sprintf("%.2f", price0*100)+"%", wml.ST_MergeUnset, 0, "", true)
	for _, cost := range costs {
		if costMap, ok := cost.(map[string]any); ok {
			r.addCostDetailRow(table, costMap)
		}
	}
}

func (r *investmentEstimateRenderer) addCostDetailRow(table document.Table, cost map[string]any) {
	price := 0.0
	row := table.AddRow()
	cell1 := row.AddCell()
	r.addTableCell(cell1, cast.ToString(cost["序号"]), wml.ST_MergeUnset, 0, "", false)

	cell2 := row.AddCell()
	r.addTableCell(cell2, cast.ToString(cost["名称"]), wml.ST_MergeUnset, 0, "", false)

	cell3 := row.AddCell()
	if cast.ToString(cost["类型"]) == "建安工程" {
		price = cast.ToFloat64(cost["金额"])
		r.addTableCell(cell3, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)
	} else {
		r.addTableCell(cell3, "", wml.ST_MergeUnset, 0, "", false)
	}

	cell4 := row.AddCell()
	if cast.ToString(cost["类型"]) == "设备及工器具购置" {
		price = cast.ToFloat64(cost["金额"])
		r.addTableCell(cell4, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)
	} else {
		r.addTableCell(cell4, "", wml.ST_MergeUnset, 0, "", false)
	}

	cell5 := row.AddCell()
	if cast.ToString(cost["类型"]) == "其他费用" {
		price = cast.ToFloat64(cost["金额"])
		r.addTableCell(cell5, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)
	} else {
		r.addTableCell(cell5, "", wml.ST_MergeUnset, 0, "", false)
	}

	cell6 := row.AddCell()
	price = cast.ToFloat64(cost["金额"])
	r.addTableCell(cell6, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)

	if cast.ToString(cost["类型"]) == "其他费用" {
		cell7 := row.AddCell()
		r.addTableCell(cell7, cast.ToString(cost["依据"]), wml.ST_MergeUnset, 4, "", false)
	} else {
		cell7 := row.AddCell()
		if cast.ToString(cost["子项"]) == "1" {
			r.addTableCell(cell7, "", wml.ST_MergeUnset, 0, "", false)
		} else {
			r.addTableCell(cell7, cast.ToString(cost["单位"]), wml.ST_MergeUnset, 0, "", false)
		}

		cell8 := row.AddCell()
		if cast.ToString(cost["子项"]) == "1" {
			r.addTableCell(cell8, "", wml.ST_MergeUnset, 0, "", false)
		} else {
			r.addTableCell(cell8, cast.ToString(cost["负荷或工程量"]), wml.ST_MergeUnset, 0, "", false)
		}

		cell9 := row.AddCell()
		if cast.ToString(cost["子项"]) == "1" {
			r.addTableCell(cell9, "", wml.ST_MergeUnset, 0, "", false)
		} else {
			r.addTableCell(cell9, cast.ToString(cost["单位指标"]), wml.ST_MergeUnset, 0, "", false)
		}

		cell10 := row.AddCell()
		r.addTableCell(cell10, "", wml.ST_MergeUnset, 0, "", false)
	}

	cell11 := row.AddCell()
	price0 := (price) / r.totalPrice
	r.addTableCell(cell11, fmt.Sprintf("%.2f", price0*100)+"%", wml.ST_MergeUnset, 0, "", false)
}

func (r *investmentEstimateRenderer) createTotalRow(table document.Table) {
	row := table.AddRow()
	cell1 := row.AddCell()
	r.addTableCell(cell1, "三", wml.ST_MergeUnset, 0, "", true)

	cell2 := row.AddCell()
	r.addTableCell(cell2, "预备费", wml.ST_MergeUnset, 0, "", true)

	cell3 := row.AddCell()
	r.addTableCell(cell3, "0.00", wml.ST_MergeUnset, 0, "", true)

	cell4 := row.AddCell()
	r.addTableCell(cell4, "0.00", wml.ST_MergeUnset, 0, "", true)

	cell5 := row.AddCell()
	price := (r.firstPrice + r.secondPrice) * 0.08
	r.otherPrice += price
	r.addTableCell(cell5, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", true)

	cell6 := row.AddCell()
	r.addTableCell(cell6, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", true)
	cell7 := row.AddCell()
	r.addTableCell(cell7, "按第一部分工程费用和第二部分其他费用之和的8%计算", wml.ST_MergeUnset, 4, "", true)

	cell11 := row.AddCell()
	price1 := (price) / r.totalPrice
	r.addTableCell(cell11, fmt.Sprintf("%.2f", price1*100)+"%", wml.ST_MergeUnset, 0, "", true)

	row = table.AddRow()
	cell1 = row.AddCell()
	r.addTableCell(cell1, "四", wml.ST_MergeUnset, 0, "", true)

	cell2 = row.AddCell()
	r.addTableCell(cell2, "项目总投资", wml.ST_MergeUnset, 0, "", true)

	cell3 = row.AddCell()
	r.addTableCell(cell3, fmt.Sprintf("%.2f", r.gcPrice/10000), wml.ST_MergeUnset, 0, "", true)

	cell4 = row.AddCell()
	r.addTableCell(cell4, fmt.Sprintf("%.2f", r.sbPrice/10000), wml.ST_MergeUnset, 0, "", true)

	cell5 = row.AddCell()
	r.addTableCell(cell5, fmt.Sprintf("%.2f", r.otherPrice/10000), wml.ST_MergeUnset, 0, "", true)

	cell6 = row.AddCell()
	price = r.gcPrice + r.sbPrice + r.otherPrice
	r.addTableCell(cell6, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", true)
	cell7 = row.AddCell()
	r.addTableCell(cell7, "", wml.ST_MergeUnset, 4, "", true)

	cell11 = row.AddCell()
	r.addTableCell(cell11, "100.00%", wml.ST_MergeUnset, 0, "", true)
}
