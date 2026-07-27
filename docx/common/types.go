package common

// MergedCell 表示一个合并单元格的范围
type MergedCell struct {
	StartRow int
	EndRow   int
	StartCol int
	EndCol   int
	Ref      string
}

// SheetConfig 工作表配置
type SheetConfig struct {
	Name     string // 工作表名称，为空表示匹配所有
	StartRow int    // 开始行（从1开始计算，如Excel），默认1表示第1行
	StartCol int    // 开始列（从1开始计算，如Excel），默认1表示第1列
	EndRow   int    // 结束行（从1开始计算，如Excel），0表示读取到最后一行
	EndCol   int    // 结束列（从1开始计算，如Excel），0表示读取到最后一列
	BoldRows []int  // 加粗的行号
}

// TableData 表示一个表格的数据
type TableData struct {
	SheetName   string       // 工作表名称
	Rows        [][]string   // 单元格数据（二维数组）
	MergedCells []MergedCell // 合并单元格信息
	MaxCols     int          // 最大列数
	BoldRows    []int        // 加粗的行号
}
