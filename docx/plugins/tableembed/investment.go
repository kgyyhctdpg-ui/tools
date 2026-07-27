package tableembed

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/common"

	"github.com/scoming-dev/tools/docx/document"
	"github.com/scoming-dev/tools/docx/measurement"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
	"github.com/spf13/cast"
)

type InvestmentEstimatePlugin struct {
	tagType string
}

func InvestmentEstimate() docx.HTMLBlockPlugin {
	return NewInvestmentEstimatePlugin("tzgs")
}

func NewInvestmentEstimatePlugin(tagType string) *InvestmentEstimatePlugin {
	return &InvestmentEstimatePlugin{tagType: tagType}
}

func (p *InvestmentEstimatePlugin) Type() string {
	return p.tagType
}

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

	// 投资估算表固定为10列，使用A4纸张横向
	paperSize := common.PaperSizeA4

	// 步骤1: 插入分节符(纵向)，结束前面的内容，确保前文保持纵向，并开始新页
	common.AddSectionBreakWithOrientation(r.Document(), common.OrientationV)

	// 创建表格标题
	// r.createTableTitle("建设项目投资估算表")
	// 步骤2: 创建表格内容（将在新页上）
	r.createInvestmentTable(data)

	// 步骤3: 插入分节符(横向)，结束表格节，确保表格页是横向，并开始新页
	// 这样后续内容将从新的一页开始，且恢复默认方向（通常是纵向）
	common.AddSectionBreakWithOrientation(r.Document(), common.OrientationH, paperSize)
	return nil
}

// 创建表格标题
func (r *investmentEstimateRenderer) createTableTitle(title string) {
	titlePara := r.Document().AddParagraph()
	titleParaProps := titlePara.Properties()
	titleParaProps.SetAlignment(wml.ST_JcCenter)

	// 设置标题段落间距
	titleSpacing := titleParaProps.Spacing()
	config := r.Config()
	titleSpacing.SetBefore(config.TOC.TitleSpacingBefore)
	titleSpacing.SetAfter(config.TOC.TitleSpacingAfter)

	titleRun := titlePara.AddRun()
	titleProps := titleRun.Properties()
	r.SetFontFamily(&titleProps, config.Fonts.Title)
	r.SetFontSize(&titleProps, config.Text.SubTitleSize)
	titleProps.SetBold(true)
	titleRun.AddText(title)
}

// 创建投资估算表
func (r *investmentEstimateRenderer) createInvestmentTable(data map[string]any) {
	// 创建表格
	table := r.Document().AddTable()
	table.Properties().SetWidthPercent(100)
	table.Properties().SetAlignment(wml.ST_JcTableCenter)

	// 设置表格边框
	borders := table.Properties().Borders()
	config := r.Config()
	borders.SetAll(wml.ST_BorderSingle, color.FromHex(config.Table.BorderColor), config.Table.BorderWidth)

	// 创建表头
	r.createTableHeader(table)

	r.totalPrice = cast.ToFloat64(data["建设总投资总计"])
	// 创建工程费用部分
	if engineeringCosts, ok := data["工程费用"].([]any); ok {
		r.createFirstCostsSection(table, engineeringCosts)
	}

	// 创建其他费用部分
	if otherCosts, ok := data["其他费用"].([]any); ok {
		r.createSecondCostsSection(table, otherCosts)
	}

	// 创建总计行
	r.createTotalRow(table, data)
}

func (r *investmentEstimateRenderer) AddTableCell(
	table document.Table,
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
	run.Properties().SetSize(measurement.Point * measurement.Distance(config.Table.HeaderFontSize))
	run.Properties().SetCharacterSpacing(config.Table.CellCharacterSpacing)
	run.AddText(text)
}

// 创建表头
func (r *investmentEstimateRenderer) createTableHeader(table document.Table) {
	// 第一行：主标题
	row1 := table.AddRow()

	// 序号列
	cell1 := row1.AddCell()
	grayColor := color.FromHex("#D9D9D9") // 浅灰色 #D9D9D9
	cell1.Properties().SetShading(wml.ST_ShdClear, grayColor, grayColor)
	r.AddTableCell(table, cell1, "序号", wml.ST_MergeRestart, 0, "#D9D9D9", true)

	// 项目名称列
	cell2 := row1.AddCell()
	r.AddTableCell(table, cell2, "项目名称", wml.ST_MergeRestart, 0, "#D9D9D9", true)

	// 估算价值列（合并3列）
	cell3 := row1.AddCell()
	r.AddTableCell(table, cell3, "估算价值(万元)", wml.ST_MergeRestart, 3, "#D9D9D9", true)

	// 合计列
	cell4 := row1.AddCell()
	r.AddTableCell(table, cell4, "合计", wml.ST_MergeRestart, 0, "#D9D9D9", true)

	// 技术经济指标列（合并3列）
	cell5 := row1.AddCell()
	r.AddTableCell(table, cell5, "技术经济指标", wml.ST_MergeRestart, 3, "#D9D9D9", true)

	// 备注列
	cell6 := row1.AddCell()
	r.AddTableCell(table, cell6, "备注", wml.ST_MergeRestart, 0, "#D9D9D9", true)

	// 比例列
	cell7 := row1.AddCell()
	r.AddTableCell(table, cell7, "比例", wml.ST_MergeRestart, 0, "#D9D9D9", true)

	// 第二行：子标题
	row2 := table.AddRow()

	// 序号列（继续垂直合并）
	cell8 := row2.AddCell()
	r.AddTableCell(table, cell8, "", wml.ST_MergeContinue, 0, "#D9D9D9", true)

	// 项目名称列（继续垂直合并）
	cell9 := row2.AddCell()
	r.AddTableCell(table, cell9, "", wml.ST_MergeContinue, 0, "#D9D9D9", true)

	// 建安工程
	cell10 := row2.AddCell()
	r.AddTableCell(table, cell10, "建安工程", wml.ST_MergeUnset, 0, "#D9D9D9", true)

	// 设备及工器具购置
	cell11 := row2.AddCell()
	r.AddTableCell(table, cell11, "设备及工器具购置", wml.ST_MergeUnset, 0, "#D9D9D9", true)

	// 其他费用
	cell12 := row2.AddCell()
	r.AddTableCell(table, cell12, "其他费用", wml.ST_MergeUnset, 0, "#D9D9D9", true)

	// 合计列（继续垂直合并）
	cell13 := row2.AddCell()
	r.AddTableCell(table, cell13, "", wml.ST_MergeContinue, 0, "#D9D9D9", true)

	// 单位
	cell14 := row2.AddCell()
	r.AddTableCell(table, cell14, "单位", wml.ST_MergeUnset, 0, "#D9D9D9", true)

	// 负荷或工程量
	cell15 := row2.AddCell()
	r.AddTableCell(table, cell15, "负荷或工程量", wml.ST_MergeUnset, 0, "#D9D9D9", true)

	// 单位指标
	cell16 := row2.AddCell()
	r.AddTableCell(table, cell16, "单位指标(元/单位)", wml.ST_MergeUnset, 0, "#D9D9D9", true)

	// 备注列（继续垂直合并）
	cell17 := row2.AddCell()
	r.AddTableCell(table, cell17, "", wml.ST_MergeContinue, 0, "#D9D9D9", true)

	// 比例列（继续垂直合并）
	cell18 := row2.AddCell()
	r.AddTableCell(table, cell18, "", wml.ST_MergeContinue, 0, "#D9D9D9", true)
}

// 创建工程费用部分
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

	// 工程费用汇总行
	row := table.AddRow()
	// 序号
	cell1 := row.AddCell()
	r.AddTableCell(table, cell1, "一", wml.ST_MergeUnset, 0, "", true)

	// 项目名称
	cell2 := row.AddCell()
	r.AddTableCell(table, cell2, "工程费用", wml.ST_MergeUnset, 0, "", true)

	// 建安工程
	cell3 := row.AddCell()
	r.AddTableCell(table, cell3, fmt.Sprintf("%.2f", price1/10000), wml.ST_MergeUnset, 0, "", true)

	// 设备及工器具购置
	cell4 := row.AddCell()
	r.AddTableCell(table, cell4, fmt.Sprintf("%.2f", price2/10000), wml.ST_MergeUnset, 0, "", true)

	// 其他费用
	cell5 := row.AddCell()
	r.AddTableCell(table, cell5, fmt.Sprintf("%.2f", price3/10000), wml.ST_MergeUnset, 0, "", true)

	// 合计
	cell6 := row.AddCell()
	r.AddTableCell(table, cell6, fmt.Sprintf("%.2f", price4/10000), wml.ST_MergeUnset, 0, "", true)

	// 技术经济指标（空）
	cell7 := row.AddCell()
	r.AddTableCell(table, cell7, "", wml.ST_MergeUnset, 0, "", true)
	cell8 := row.AddCell()
	r.AddTableCell(table, cell8, "", wml.ST_MergeUnset, 0, "", true)
	cell9 := row.AddCell()
	r.AddTableCell(table, cell9, "", wml.ST_MergeUnset, 0, "", true)

	// 备注（空）
	cell10 := row.AddCell()
	r.AddTableCell(table, cell10, "", wml.ST_MergeUnset, 0, "", true)

	// 比例
	cell11 := row.AddCell()
	price0 := (price4) / r.totalPrice
	r.AddTableCell(table, cell11, fmt.Sprintf("%.2f", price0*100)+"%", wml.ST_MergeUnset, 0, "", true)

	// 添加工程费用明细
	for i, cost := range costs {
		if costMap, ok := cost.(map[string]any); ok {
			r.addCostDetailRow(table, i+1, costMap)
		}
	}
}

// 创建其他费用部分
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

	// 其他费用汇总行
	row := table.AddRow()

	// 序号
	cell1 := row.AddCell()
	r.AddTableCell(table, cell1, "二", wml.ST_MergeUnset, 0, "", true)
	// 项目名称
	cell2 := row.AddCell()
	r.AddTableCell(table, cell2, "其他费用", wml.ST_MergeUnset, 0, "", true)
	// 建安工程
	cell3 := row.AddCell()
	r.AddTableCell(table, cell3, fmt.Sprintf("%.2f", price1/10000), wml.ST_MergeUnset, 0, "", true)
	// 设备及工器具购置
	cell4 := row.AddCell()
	r.AddTableCell(table, cell4, fmt.Sprintf("%.2f", price2/10000), wml.ST_MergeUnset, 0, "", true)
	// 其他费用
	cell5 := row.AddCell()
	r.AddTableCell(table, cell5, fmt.Sprintf("%.2f", price3/10000), wml.ST_MergeUnset, 0, "", true)
	// 合计
	cell6 := row.AddCell()
	r.AddTableCell(table, cell6, fmt.Sprintf("%.2f", price4/10000), wml.ST_MergeUnset, 0, "", true)
	// 技术经济指标（空）
	cell7 := row.AddCell()
	r.AddTableCell(table, cell7, "", wml.ST_MergeUnset, 0, "", true)
	cell8 := row.AddCell()
	r.AddTableCell(table, cell8, "", wml.ST_MergeUnset, 0, "", true)
	cell9 := row.AddCell()
	r.AddTableCell(table, cell9, "", wml.ST_MergeUnset, 0, "", true)
	// 备注（空）
	cell10 := row.AddCell()
	r.AddTableCell(table, cell10, "", wml.ST_MergeUnset, 0, "", true)
	// 比例
	cell11 := row.AddCell()
	price0 := (price4) / r.totalPrice
	r.AddTableCell(table, cell11, fmt.Sprintf("%.2f", price0*100)+"%", wml.ST_MergeUnset, 0, "", true)
	// 添加其他费用明细
	for i, cost := range costs {
		if costMap, ok := cost.(map[string]any); ok {
			r.addCostDetailRow(table, i+1, costMap)
		}
	}
}

// 添加费用明细行
func (r *investmentEstimateRenderer) addCostDetailRow(table document.Table, index int, cost map[string]any) {
	price := 0.0
	row := table.AddRow()
	// 序号
	cell1 := row.AddCell()
	r.AddTableCell(table, cell1, cast.ToString(cost["序号"]), wml.ST_MergeUnset, 0, "", false)
	// 名称
	cell2 := row.AddCell()
	r.AddTableCell(table, cell2, cast.ToString(cost["名称"]), wml.ST_MergeUnset, 0, "", false)
	// 设备及工器具购置
	cell3 := row.AddCell()
	if cast.ToString(cost["类型"]) == "建安工程" {
		price = cast.ToFloat64(cost["金额"])
		r.AddTableCell(table, cell3, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)
	} else {
		r.AddTableCell(table, cell3, "", wml.ST_MergeUnset, 0, "", false)
	}
	// 金额
	cell4 := row.AddCell()
	if cast.ToString(cost["类型"]) == "设备及工器具购置" {
		price = cast.ToFloat64(cost["金额"])
		r.AddTableCell(table, cell4, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)
	} else {
		r.AddTableCell(table, cell4, "", wml.ST_MergeUnset, 0, "", false)
	}
	// 其他费用
	cell5 := row.AddCell()
	if cast.ToString(cost["类型"]) == "其他费用" {
		price = cast.ToFloat64(cost["金额"])
		r.AddTableCell(table, cell5, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)
	} else {
		r.AddTableCell(table, cell5, "", wml.ST_MergeUnset, 0, "", false)
	}
	// 合计
	cell6 := row.AddCell()
	price = cast.ToFloat64(cost["金额"])
	r.AddTableCell(table, cell6, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", false)

	if cast.ToString(cost["类型"]) == "其他费用" {
		cell7 := row.AddCell()
		r.AddTableCell(table, cell7, cast.ToString(cost["依据"]), wml.ST_MergeUnset, 4, "", false)
	} else {
		// 单位
		cell7 := row.AddCell()
		if cast.ToString(cost["子项"]) == "1" {
			r.AddTableCell(table, cell7, "", wml.ST_MergeUnset, 0, "", false)
		} else {
			r.AddTableCell(table, cell7, cast.ToString(cost["单位"]), wml.ST_MergeUnset, 0, "", false)
		}
		// 负荷或工程量
		cell8 := row.AddCell()
		if cast.ToString(cost["子项"]) == "1" {
			r.AddTableCell(table, cell8, "", wml.ST_MergeUnset, 0, "", false)
		} else {
			r.AddTableCell(table, cell8, cast.ToString(cost["负荷或工程量"]), wml.ST_MergeUnset, 0, "", false)
		}

		// 单位指标
		cell9 := row.AddCell()
		if cast.ToString(cost["子项"]) == "1" {
			r.AddTableCell(table, cell9, "", wml.ST_MergeUnset, 0, "", false)
		} else {
			r.AddTableCell(table, cell9, cast.ToString(cost["单位指标"]), wml.ST_MergeUnset, 0, "", false)
		}
		// 备注
		cell10 := row.AddCell()
		r.AddTableCell(table, cell10, "", wml.ST_MergeUnset, 0, "", false)
	}

	// 比例
	cell11 := row.AddCell()
	price0 := (price) / r.totalPrice
	r.AddTableCell(table, cell11, fmt.Sprintf("%.2f", price0*100)+"%", wml.ST_MergeUnset, 0, "", false)

}

// 创建总计行
func (r *investmentEstimateRenderer) createTotalRow(table document.Table, data map[string]any) {
	row := table.AddRow()
	// 序号
	cell1 := row.AddCell()
	r.AddTableCell(table, cell1, "三", wml.ST_MergeUnset, 0, "", true)
	// 项目名称（空）
	cell2 := row.AddCell()
	r.AddTableCell(table, cell2, "预备费", wml.ST_MergeUnset, 0, "", true)
	// 建安工程
	cell3 := row.AddCell()
	r.AddTableCell(table, cell3, "0.00", wml.ST_MergeUnset, 0, "", true)
	// 设备及工器具购置
	cell4 := row.AddCell()
	r.AddTableCell(table, cell4, "0.00", wml.ST_MergeUnset, 0, "", true)
	// 其他费用
	cell5 := row.AddCell()
	price := (r.firstPrice + r.secondPrice) * 0.08
	r.otherPrice += price
	r.AddTableCell(table, cell5, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", true)
	// 合计
	cell6 := row.AddCell()
	r.AddTableCell(table, cell6, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", true)
	cell7 := row.AddCell()
	r.AddTableCell(table, cell7, "按第一部分工程费用和第二部分其他费用之和的8%计算", wml.ST_MergeUnset, 4, "", true)
	// 比例
	cell11 := row.AddCell()
	price1 := (price) / r.totalPrice
	r.AddTableCell(table, cell11, fmt.Sprintf("%.2f", price1*100)+"%", wml.ST_MergeUnset, 0, "", true)

	// 第四部分
	row = table.AddRow()
	// 序号
	cell1 = row.AddCell()
	r.AddTableCell(table, cell1, "四", wml.ST_MergeUnset, 0, "", true)
	// 项目名称（空）
	cell2 = row.AddCell()
	r.AddTableCell(table, cell2, "项目总投资", wml.ST_MergeUnset, 0, "", true)
	// 建安工程
	cell3 = row.AddCell()
	r.AddTableCell(table, cell3, fmt.Sprintf("%.2f", r.gcPrice/10000), wml.ST_MergeUnset, 0, "", true)
	// 设备及工器具购置
	cell4 = row.AddCell()
	r.AddTableCell(table, cell4, fmt.Sprintf("%.2f", r.sbPrice/10000), wml.ST_MergeUnset, 0, "", true)
	// 其他费用
	cell5 = row.AddCell()
	r.AddTableCell(table, cell5, fmt.Sprintf("%.2f", r.otherPrice/10000), wml.ST_MergeUnset, 0, "", true)
	// 合计
	cell6 = row.AddCell()
	price = r.gcPrice + r.sbPrice + r.otherPrice
	r.AddTableCell(table, cell6, fmt.Sprintf("%.2f", price/10000), wml.ST_MergeUnset, 0, "", true)
	cell7 = row.AddCell()
	r.AddTableCell(table, cell7, "", wml.ST_MergeUnset, 4, "", true)
	// 比例
	cell11 = row.AddCell()
	r.AddTableCell(table, cell11, "100.00%", wml.ST_MergeUnset, 0, "", true)
}
