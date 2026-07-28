package common

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

// GetMergedCells returns worksheet merged-cell ranges normalized to the selected range.
func GetMergedCells(f *excelize.File, sheetName string, offsetRow, offsetCol, endRow, endCol int) ([]MergedCell, error) {
	mergeList, err := f.GetMergeCells(sheetName)
	if err != nil {
		return nil, err
	}

	var result []MergedCell
	for _, merge := range mergeList {
		ref := merge.GetCellValue()
		startAxis := merge.GetStartAxis()
		endAxis := merge.GetEndAxis()

		startCol, startRow, _ := excelize.CellNameToCoordinates(startAxis)
		endCol2, endRow2, _ := excelize.CellNameToCoordinates(endAxis)

		// excelize returns 1-based coordinates; plugin table data uses zero-based indexes.
		startRow--
		endRow2--
		startCol--
		endCol2--

		if endRow > 0 && startRow > endRow {
			continue
		}
		if endCol > 0 && startCol > endCol {
			continue
		}

		adjustedStartRow := startRow - offsetRow
		adjustedEndRow := endRow2 - offsetRow
		adjustedStartCol := startCol - offsetCol
		adjustedEndCol := endCol2 - offsetCol

		if endRow > 0 && endRow2 > endRow {
			adjustedEndRow = endRow - offsetRow
		}
		if endCol > 0 && endCol2 > endCol {
			adjustedEndCol = endCol - offsetCol
		}

		if adjustedEndRow >= 0 && adjustedEndCol >= 0 {
			if adjustedStartRow < 0 {
				adjustedStartRow = 0
			}
			if adjustedStartCol < 0 {
				adjustedStartCol = 0
			}

			result = append(result, MergedCell{
				StartRow: adjustedStartRow,
				EndRow:   adjustedEndRow,
				StartCol: adjustedStartCol,
				EndCol:   adjustedEndCol,
				Ref:      ref,
			})
		}
	}

	return result, nil
}

// BuildMergeMap indexes merged-cell ranges by their top-left cell.
func BuildMergeMap(mergedCells []MergedCell) map[string]MergedCell {
	mergeMap := make(map[string]MergedCell)

	for _, merge := range mergedCells {
		key := fmt.Sprintf("%d-%d", merge.StartRow, merge.StartCol)
		mergeMap[key] = merge
	}

	return mergeMap
}

// GetSheetConfig resolves a sheet-specific config and converts public 1-based indexes to zero-based indexes.
func GetSheetConfig(sheetName string, configs []SheetConfig) SheetConfig {
	var config SheetConfig
	found := false

	for _, cfg := range configs {
		if cfg.Name == sheetName {
			config = cfg
			found = true
			break
		}
	}

	if !found {
		for _, cfg := range configs {
			if cfg.Name == "" {
				config = cfg
				found = true
				break
			}
		}
	}

	if !found {
		config = SheetConfig{
			Name:     sheetName,
			StartRow: 1,
			StartCol: 1,
			EndRow:   0,
			EndCol:   0,
		}
	}

	if config.StartRow <= 0 {
		config.StartRow = 1
	}
	if config.StartCol <= 0 {
		config.StartCol = 1
	}

	config.StartRow--
	config.StartCol--

	if config.EndRow > 0 {
		config.EndRow--
		if config.EndRow < config.StartRow {
			log.Printf("警告: 工作表 %s 的结束行（%d）小于开始行（%d），将调整为开始行", sheetName, config.EndRow+1, config.StartRow+1)
			config.EndRow = config.StartRow
		}
	}

	if config.EndCol > 0 {
		config.EndCol--
		if config.EndCol < config.StartCol {
			log.Printf("警告: 工作表 %s 的结束列（%d）小于开始列（%d），将调整为开始列", sheetName, config.EndCol+1, config.StartCol+1)
			config.EndCol = config.StartCol
		}
	}

	return config
}
