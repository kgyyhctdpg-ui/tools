package tableembed

import (
	"fmt"
	"log"
	"slices"

	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/common"
	"github.com/scoming-dev/tools/docx/measurement"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
	"github.com/xuri/excelize/v2"
)

type ExcelTablePlugin struct {
	tagType      string
	sheetConfigs []common.SheetConfig
}

func NewExcelTablePlugin(tagType string, sheetConfigs []common.SheetConfig) *ExcelTablePlugin {
	configs := make([]common.SheetConfig, len(sheetConfigs))
	copy(configs, sheetConfigs)
	return &ExcelTablePlugin{tagType: tagType, sheetConfigs: configs}
}

func (p *ExcelTablePlugin) Type() string {
	return p.tagType
}

func (p *ExcelTablePlugin) RenderHTML(renderer *docx.DocxRenderer, content string) error {
	excelPath := common.GetNodeValue(content, "s-tag", "url")
	sheetName := common.GetNodeValue(content, "s-tag", "name")
	if excelPath == "" {
		return fmt.Errorf("表格链接不存在")
	}

	tables, err := extractTableDataFromExcel(renderer, excelPath, p.sheetConfigs)
	if err != nil {
		return fmt.Errorf("提取表格数据失败: %w", err)
	}

	if err := generateDocFromTables(renderer, tables, sheetName); err != nil {
		return fmt.Errorf("生成 Word 文档失败: %w", err)
	}
	return nil
}

func FinancialCalculation() docx.HTMLBlockPlugin {
	return NewExcelTablePlugin("cwcs", FinancialCalculationSheetConfigs())
}

func Performance() docx.HTMLBlockPlugin {
	return NewExcelTablePlugin("performance", []common.SheetConfig{
		{Name: "绩效目标表", StartRow: 2, StartCol: 0, EndRow: 0, EndCol: 0, BoldRows: []int{}},
	})
}

func Assessment() docx.HTMLBlockPlugin {
	return NewExcelTablePlugin("assessment", []common.SheetConfig{
		{Name: "绩效评估表", StartRow: 0, StartCol: 0, EndRow: 27, EndCol: 0, BoldRows: []int{}},
	})
}

func Tender() docx.HTMLBlockPlugin {
	return NewExcelTablePlugin("tender", []common.SheetConfig{
		{Name: "招标基本情况表", StartRow: 2, StartCol: 0, EndRow: 10, EndCol: 0, BoldRows: []int{2, 3}},
	})
}

func DefaultPlugins() []docx.HTMLBlockPlugin {
	return []docx.HTMLBlockPlugin{
		InvestmentEstimate(),
		FinancialCalculation(),
		Performance(),
		Assessment(),
		Tender(),
	}
}

func FinancialCalculationSheetConfigs() []common.SheetConfig {
	return []common.SheetConfig{
		{Name: "项目基本情况", StartRow: 2, StartCol: 0, EndRow: 0, EndCol: 0, BoldRows: []int{1}},
		{Name: "还本付息表-发行一期债券", StartRow: 3, StartCol: 2, EndRow: 52, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "收入测算表", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "补贴收入", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "项目营业收入、增值税及附加估算表", StartRow: 3, StartCol: 2, EndRow: 23, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "折旧摊销", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "成本测算表", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "利润及利润分配表", StartRow: 3, StartCol: 2, EndRow: 19, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "弥补亏损", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "净收益", StartRow: 3, StartCol: 2, EndRow: 19, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "现金流量表", StartRow: 3, StartCol: 2, EndRow: 46, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "组合", StartRow: 31, StartCol: 2, EndRow: 0, EndCol: 11, BoldRows: []int{1, 2}},
		{Name: "专项债", StartRow: 31, StartCol: 2, EndRow: 0, EndCol: 11, BoldRows: []int{1, 2}},
		{Name: "市场化", StartRow: 31, StartCol: 2, EndRow: 0, EndCol: 11, BoldRows: []int{1, 2}},
		{Name: "敏感性分析", StartRow: 3, StartCol: 2, EndRow: 8, EndCol: 11, BoldRows: []int{1, 2}},
		{Name: "投资估算表", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{1, 2}},
		{Name: "绩效目标表", StartRow: 3, StartCol: 2, EndRow: 0, EndCol: 0, BoldRows: []int{}},
		{Name: "绩效评估表", StartRow: 3, StartCol: 2, EndRow: 28, EndCol: 0, BoldRows: []int{}},
		{Name: "招标基本情况表", StartRow: 3, StartCol: 2, EndRow: 11, EndCol: 0, BoldRows: []int{1, 2}},
	}
}

func extractTableDataFromExcel(renderer *docx.DocxRenderer, excelPath string, sheetConfigs []common.SheetConfig) ([]common.TableData, error) {
	actualFilePath, isTemp, err := common.PrepareFilepath(excelPath)
	if err != nil {
		return nil, fmt.Errorf("准备文件路径失败: %w", err)
	}
	if isTemp {
		renderer.AddTempFile(actualFilePath)
	}

	f, err := excelize.OpenFile(actualFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开 Excel 文件失败: %w", err)
	}
	defer f.Close()

	sheetList := f.GetSheetList()
	var tables []common.TableData

	for _, sheetName := range sheetList {
		config := common.GetSheetConfig(sheetName, sheetConfigs)
		logSheetConfig(sheetName, config)

		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("读取工作表 %s 失败: %v", sheetName, err)
			continue
		}
		if len(rows) == 0 {
			continue
		}

		if config.StartRow > 0 {
			if config.StartRow >= len(rows) {
				log.Printf("工作表 %s 的开始行（第%d行）超出范围，跳过该工作表", sheetName, config.StartRow+1)
				continue
			}
			rows = rows[config.StartRow:]
		}

		if config.EndRow > 0 {
			endRowInSlice := config.EndRow - config.StartRow + 1
			if endRowInSlice < len(rows) {
				rows = rows[:endRowInSlice]
			}
		}

		mergedCells, err := common.GetMergedCells(f, sheetName, config.StartRow, config.StartCol, config.EndRow, config.EndCol)
		if err != nil {
			log.Printf("获取合并单元格失败: %v", err)
			mergedCells = []common.MergedCell{}
		}

		maxCols := maxColumnCount(rows, config)
		if maxCols <= 0 {
			log.Printf("工作表 %s 没有有效列，跳过", sheetName)
			continue
		}

		tables = append(tables, common.TableData{
			SheetName:   sheetName,
			Rows:        extractConfiguredRows(rows, config, maxCols),
			MergedCells: mergedCells,
			MaxCols:     maxCols,
			BoldRows:    config.BoldRows,
		})
	}

	return tables, nil
}

func generateDocFromTables(renderer *docx.DocxRenderer, tables []common.TableData, sheetName string) error {
	doc := renderer.Document()
	for _, tableData := range tables {
		if tableData.SheetName != sheetName {
			continue
		}
		log.Printf("生成表格: %s (%d行 x %d列)", tableData.SheetName, len(tableData.Rows), tableData.MaxCols)

		paperSize := selectPaperSize(tableData.MaxCols)
		log.Printf("列数: %d, 选择纸张: %s, 方向: 横向", tableData.MaxCols, paperSize)

		if !common.IsLastSectionLandscape(doc) {
			common.AddSectionBreak(doc)
		}

		mergeMap := common.BuildMergeMap(tableData.MergedCells)

		titlePara := doc.AddParagraph()
		titleParaProps := titlePara.Properties()
		titleParaProps.SetAlignment(wml.ST_JcCenter)
		renderer.SetParagraphSpacing(&titlePara)
		titleRun := titlePara.AddRun()
		titleRunProps := titleRun.Properties()
		config := renderer.Config()
		renderer.SetSubTitleFont(&titleRunProps, config.Fonts.Title)
		titleRun.AddText(fmt.Sprintf("《%s》", tableData.SheetName))

		table := doc.AddTable()
		table.Properties().SetWidthPercent(100)
		table.Properties().SetAlignment(wml.ST_JcTableCenter)
		table.Properties().Borders().SetAll(wml.ST_BorderSingle, color.FromHex(config.Table.BorderColor), config.Table.BorderWidth)

		verticalMergeContinue := verticalMergeContinueMap(tableData.MergedCells)
		for rowIdx, rowData := range tableData.Rows {
			docRow := table.AddRow()
			for colIdx := 0; colIdx < tableData.MaxCols; colIdx++ {
				cellKey := fmt.Sprintf("%d-%d", rowIdx, colIdx)
				if verticalMergeContinue[cellKey] {
					cell := docRow.AddCell()
					cellProps := cell.Properties()
					cellProps.Borders().SetAll(wml.ST_BorderSingle, color.FromHex(config.Table.BorderColor), config.Table.BorderWidth)
					cellProps.SetVerticalMerge(wml.ST_MergeContinue)
					if merge, exists := mergeMap[cellKey]; exists && merge.EndCol > merge.StartCol {
						cellProps.SetColumnSpan(merge.EndCol - merge.StartCol + 1)
						colIdx = merge.EndCol
					} else {
						for _, merge := range tableData.MergedCells {
							if rowIdx >= merge.StartRow && rowIdx <= merge.EndRow &&
								colIdx >= merge.StartCol && colIdx <= merge.EndCol {
								if merge.EndCol > merge.StartCol && colIdx == merge.StartCol {
									cellProps.SetColumnSpan(merge.EndCol - merge.StartCol + 1)
									colIdx = merge.EndCol
								}
								break
							}
						}
					}
					cell.AddParagraph()
					continue
				}

				cellValue := ""
				if colIdx < len(rowData) {
					cellValue = rowData[colIdx]
				}

				cell := docRow.AddCell()
				cellProps := cell.Properties()
				cellProps.Borders().SetAll(wml.ST_BorderSingle, color.FromHex(config.Table.BorderColor), config.Table.BorderWidth)
				if merge, exists := mergeMap[cellKey]; exists {
					if merge.EndCol > merge.StartCol {
						cellProps.SetColumnSpan(merge.EndCol - merge.StartCol + 1)
						colIdx = merge.EndCol
					}
					if merge.EndRow > merge.StartRow {
						cellProps.SetVerticalMerge(wml.ST_MergeRestart)
					}
				}

				cellProps.SetVerticalAlignment(wml.ST_VerticalJcCenter)
				para := cell.AddParagraph()
				paraProps := para.Properties()
				paraProps.SetAlignment(wml.ST_JcCenter)
				paraProps.SetFirstLineIndent(0)

				run := para.AddRun()
				runProps := run.Properties()
				runProps.SetFontFamily(config.Fonts.Content)
				runProps.SetSize(measurement.Point * measurement.Distance(config.Table.CellFontSize))

				if slices.Contains(tableData.BoldRows, rowIdx+1) {
					runProps.SetBold(true)
					cellProps.SetShading(wml.ST_ShdClear, color.RGB(217, 217, 217), color.RGB(217, 217, 217))
				}

				run.AddText(cellValue)
			}
		}
		common.AddSectionBreakWithOrientation(doc, common.OrientationH, paperSize)
		if !common.IsLastSectionLandscape(doc) {
			common.AddSectionBreak(doc)
		}
	}
	return nil
}

func logSheetConfig(sheetName string, config common.SheetConfig) {
	if config.EndRow > 0 && config.EndCol > 0 {
		log.Printf("提取工作表: %s, 范围: 第%d行-%d行, 第%d列-%d列", sheetName, config.StartRow+1, config.EndRow+1, config.StartCol+1, config.EndCol+1)
		return
	}
	if config.EndRow > 0 {
		log.Printf("提取工作表: %s, 范围: 第%d行-%d行, 第%d列-末列", sheetName, config.StartRow+1, config.EndRow+1, config.StartCol+1)
		return
	}
	if config.EndCol > 0 {
		log.Printf("提取工作表: %s, 范围: 第%d行-末行, 第%d列-%d列", sheetName, config.StartRow+1, config.StartCol+1, config.EndCol+1)
		return
	}
	log.Printf("提取工作表: %s, 开始位置: 第%d行, 第%d列", sheetName, config.StartRow+1, config.StartCol+1)
}

func maxColumnCount(rows [][]string, config common.SheetConfig) int {
	maxCols := 0
	for _, row := range rows {
		effectiveCols := len(row) - config.StartCol
		if config.EndCol > 0 {
			maxColsFromEnd := config.EndCol - config.StartCol + 1
			if maxColsFromEnd < effectiveCols {
				effectiveCols = maxColsFromEnd
			}
		}
		if effectiveCols > maxCols {
			maxCols = effectiveCols
		}
	}
	return maxCols
}

func extractConfiguredRows(rows [][]string, config common.SheetConfig, maxCols int) [][]string {
	processedRows := make([][]string, len(rows))
	for rowIdx, rowData := range rows {
		processedRows[rowIdx] = make([]string, maxCols)
		for colIdx := 0; colIdx < maxCols; colIdx++ {
			actualColIdx := colIdx + config.StartCol
			if actualColIdx < len(rowData) {
				processedRows[rowIdx][colIdx] = rowData[actualColIdx]
			}
		}
	}
	return processedRows
}

func verticalMergeContinueMap(mergedCells []common.MergedCell) map[string]bool {
	ret := make(map[string]bool)
	for _, merge := range mergedCells {
		if merge.EndRow <= merge.StartRow {
			continue
		}
		for row := merge.StartRow + 1; row <= merge.EndRow; row++ {
			for col := merge.StartCol; col <= merge.EndCol; col++ {
				ret[fmt.Sprintf("%d-%d", row, col)] = true
			}
		}
	}
	return ret
}

func selectPaperSize(cols int) common.PaperSize {
	switch {
	case cols <= 20:
		return common.PaperSizeA4
	case cols <= 30:
		return common.PaperSizeA3
	case cols <= 40:
		return common.PaperSizeA2
	default:
		return common.PaperSizeA1
	}
}
