// Package excel provides simple helpers for reading and writing xlsx files.
package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Header describes an exported column.
type Header struct {
	Title string
	Key   string
	Width float64
}

type exportConfig struct {
	freezeHeader bool
	autoFilter   bool
}

// ExportOption configures xlsx export.
type ExportOption func(*exportConfig)

// WithFreezeHeader freezes the first row.
func WithFreezeHeader() ExportOption {
	return func(cfg *exportConfig) {
		cfg.freezeHeader = true
	}
}

// WithAutoFilter adds an auto filter to the header row.
func WithAutoFilter() ExportOption {
	return func(cfg *exportConfig) {
		cfg.autoFilter = true
	}
}

// ExportMaps exports rows into an xlsx file using headers as title/key mapping.
func ExportMaps(path, sheet string, headers []Header, rows []map[string]any, opts ...ExportOption) error {
	if len(headers) == 0 {
		return fmt.Errorf("excel: headers are required")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet = prepareSheet(f, sheet)
	cfg := applyExportOptions(opts...)

	if err := writeHeaders(f, sheet, headers); err != nil {
		return err
	}
	for rowIndex, row := range rows {
		for colIndex, header := range headers {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, row[header.Key]); err != nil {
				return err
			}
		}
	}
	if err := applySheetOptions(f, sheet, len(headers), len(rows)+1, cfg); err != nil {
		return err
	}
	return saveAs(f, path)
}

// ExportRows exports a simple two-dimensional table. headers can be nil.
func ExportRows(path, sheet string, headers []string, rows [][]any, opts ...ExportOption) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet = prepareSheet(f, sheet)
	cfg := applyExportOptions(opts...)

	rowOffset := 1
	if len(headers) > 0 {
		for colIndex, title := range headers {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, 1)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, title); err != nil {
				return err
			}
		}
		if err := styleHeader(f, sheet, len(headers)); err != nil {
			return err
		}
		rowOffset = 2
	}

	maxCols := len(headers)
	for rowIndex, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
		for colIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+rowOffset)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return err
			}
		}
	}
	if err := applySheetOptions(f, sheet, maxCols, len(rows)+rowOffset-1, cfg); err != nil {
		return err
	}
	return saveAs(f, path)
}

// ReadRows reads all rows from a sheet. Empty trailing cells follow excelize's
// behavior and may be omitted.
func ReadRows(path, sheet string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if sheet == "" {
		sheet = f.GetSheetName(0)
	}
	return f.GetRows(sheet)
}

// ReadMaps reads rows using a header row. headerRow is 1-based; values <= 0 use
// the first row.
func ReadMaps(path, sheet string, headerRow int) ([]map[string]string, error) {
	rows, err := ReadRows(path, sheet)
	if err != nil {
		return nil, err
	}
	if headerRow <= 0 {
		headerRow = 1
	}
	if len(rows) < headerRow {
		return nil, nil
	}
	headers := rows[headerRow-1]
	result := make([]map[string]string, 0, len(rows)-headerRow)
	for _, row := range rows[headerRow:] {
		if isBlankRow(row) {
			continue
		}
		item := make(map[string]string, len(headers))
		for i, header := range headers {
			header = strings.TrimSpace(header)
			if header == "" {
				continue
			}
			if i < len(row) {
				item[header] = row[i]
			} else {
				item[header] = ""
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func prepareSheet(f *excelize.File, sheet string) string {
	defaultSheet := f.GetSheetName(0)
	if sheet == "" {
		return defaultSheet
	}
	if sheet != defaultSheet {
		_, _ = f.NewSheet(sheet)
		_ = f.DeleteSheet(defaultSheet)
	}
	return sheet
}

func applyExportOptions(opts ...ExportOption) exportConfig {
	var cfg exportConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func writeHeaders(f *excelize.File, sheet string, headers []Header) error {
	for colIndex, header := range headers {
		cell, err := excelize.CoordinatesToCellName(colIndex+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, header.Title); err != nil {
			return err
		}
		if header.Width > 0 {
			col, err := excelize.ColumnNumberToName(colIndex + 1)
			if err != nil {
				return err
			}
			if err := f.SetColWidth(sheet, col, col, header.Width); err != nil {
				return err
			}
		}
	}
	return styleHeader(f, sheet, len(headers))
}

func styleHeader(f *excelize.File, sheet string, columnCount int) error {
	if columnCount == 0 {
		return nil
	}
	style, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	lastCell, err := excelize.CoordinatesToCellName(columnCount, 1)
	if err != nil {
		return err
	}
	return f.SetCellStyle(sheet, "A1", lastCell, style)
}

func applySheetOptions(f *excelize.File, sheet string, columnCount, rowCount int, cfg exportConfig) error {
	if cfg.freezeHeader {
		if err := f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			Split:       false,
			XSplit:      0,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		}); err != nil {
			return err
		}
	}
	if cfg.autoFilter && columnCount > 0 && rowCount > 0 {
		lastCell, err := excelize.CoordinatesToCellName(columnCount, rowCount)
		if err != nil {
			return err
		}
		if err := f.AutoFilter(sheet, "A1:"+lastCell, nil); err != nil {
			return err
		}
	}
	return nil
}

func saveAs(f *excelize.File, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return f.SaveAs(path)
}

func isBlankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
