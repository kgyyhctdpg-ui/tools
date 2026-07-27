package common

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/88250/lute/html"
	"github.com/mmonterroca/docxgo/v2/domain"
	"github.com/scoming-dev/tools/docx/document"
	"github.com/xuri/excelize/v2"
)

// PaperSize 纸张大小类型
type PaperSize string

const (
	PaperSizeA1 PaperSize = "A1"
	PaperSizeA2 PaperSize = "A2"
	PaperSizeA3 PaperSize = "A3"
	PaperSizeA4 PaperSize = "A4"
)

type Orientation string

const (
	OrientationH Orientation = "landscape"
	OrientationV Orientation = "portrait"
)

// paperDimensions 纸张尺寸定义（纵向时的宽度和高度，以twips为单位）
// 1 inch = 1440 twips
// A1: 594mm × 841mm = 23.39" × 33.11"
// A2: 420mm × 594mm = 16.54" × 23.39"
// A3: 297mm × 420mm = 11.69" × 16.54"
// A4: 210mm × 297mm = 8.27" × 11.69"
var paperDimensions = map[PaperSize]struct {
	width  uint64
	height uint64
}{
	PaperSizeA1: {width: 33681, height: 47682}, // 23.39*1440 × 33.11*1440
	PaperSizeA2: {width: 23811, height: 33681}, // 16.54*1440 × 23.39*1440
	PaperSizeA3: {width: 16834, height: 23811}, // 11.69*1440 × 16.54*1440
	PaperSizeA4: {width: 11909, height: 16834}, // 8.27*1440 × 11.69*1440
}

func AddSectionBreak(doc *document.Document) {
	doc.AddSectionBreak(domain.OrientationPortrait, domain.PageSizeA4)
}

// AddSectionBreakWithOrientation 添加分节符并设置页面方向
func AddSectionBreakWithOrientation(doc *document.Document, orientation Orientation, paperSize ...PaperSize) {

	// 默认使用 A4
	size := PaperSizeA4
	if len(paperSize) > 0 {
		size = paperSize[0]
	}

	// 获取纸张尺寸（这些是纵向时的尺寸）
	dim, ok := paperDimensions[size]
	if !ok {
		dim = paperDimensions[PaperSizeA4] // 如果未找到，使用 A4
	}

	var widthVal, heightVal int
	docOrientation := domain.OrientationPortrait
	if orientation == OrientationH {
		widthVal = int(dim.height)
		heightVal = int(dim.width)
		docOrientation = domain.OrientationLandscape
	} else {
		widthVal = int(dim.width)
		heightVal = int(dim.height)
	}

	doc.AddSectionBreak(docOrientation, domain.PageSize{Width: widthVal, Height: heightVal})
}

// 获取节点
func GetNode(content string, selector string) (nodes *html.Node, err error) {
	err = nil
	htmlTree, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	var findNodes func(*html.Node) error
	findNodes = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == selector {
			nodes = n
			return nil
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = findNodes(c); err == nil {
				return nil
			}
		}
		return errors.New("search nodes error")
	}
	err = findNodes(htmlTree)
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		return nil, errors.New("not found nodes")
	}
	return nodes, err
}

// 获取节点属性
func GetNodeValue(content string, selector string, attrKey string) string {
	node, err := GetNode(content, selector)
	if err != nil {
		return ""
	}
	if attrKey == "" {
		if node.FirstChild == nil {
			return ""
		}
		return strings.TrimSpace(node.FirstChild.Data)
	}
	for _, attr := range node.Attr {
		if attr.Key == attrKey {
			return attr.Val
		}
	}
	return ""
}

// EditAttribute 修改 markdown 中 HTML 标签的属性值
func EditAttribute(content string, selector string, filter map[string]string, key, val string) (string, error) {
	// 匹配标签，例如 <s-tag type='cwcs' url='' name='xxx'>
	re := regexp.MustCompile(fmt.Sprintf(`(?i)<(%s)(\s[^>]*)?(/?)>`, regexp.QuoteMeta(selector)))
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取标签名和属性部分
		matches := re.FindStringSubmatch(match)
		if len(matches) < 4 {
			return match
		}

		tagName := matches[1]
		attrStr := matches[2]
		closing := matches[3]

		// 解析属性 - 分别匹配单引号和双引号
		attrRe1 := regexp.MustCompile(`([a-zA-Z0-9_\-:]+)\s*=\s*'([^']*)'`)
		attrRe2 := regexp.MustCompile(`([a-zA-Z0-9_\-:]+)\s*=\s*"([^"]*)"`)

		attrs := make(map[string]string)

		// 匹配单引号属性
		for _, m := range attrRe1.FindAllStringSubmatch(attrStr, -1) {
			if len(m) == 3 {
				attrs[m[1]] = m[2]
			}
		}

		// 匹配双引号属性
		for _, m := range attrRe2.FindAllStringSubmatch(attrStr, -1) {
			if len(m) == 3 {
				attrs[m[1]] = m[2]
			}
		}

		// 检查过滤条件
		for k, v := range filter {
			if curVal, ok := attrs[k]; !ok || curVal != v {
				return match
			}
		}

		// 修改或添加目标属性
		if _, exists := attrs[key]; exists {
			// 替换现有属性 - 同时支持单引号和双引号
			keyRe1 := regexp.MustCompile(fmt.Sprintf(`(?i)(\s+)%s\s*=\s*'[^']*'`, regexp.QuoteMeta(key)))
			keyRe2 := regexp.MustCompile(fmt.Sprintf(`(?i)(\s+)%s\s*=\s*"[^"]*"`, regexp.QuoteMeta(key)))

			if keyRe1.MatchString(attrStr) {
				attrStr = keyRe1.ReplaceAllString(attrStr, fmt.Sprintf(`${1}%s='%s'`, key, val))
			} else if keyRe2.MatchString(attrStr) {
				attrStr = keyRe2.ReplaceAllString(attrStr, fmt.Sprintf(`${1}%s="%s"`, key, val))
			}
		} else {
			// 追加新属性
			attrStr += fmt.Sprintf(` %s="%s"`, key, val)
		}

		return fmt.Sprintf("<%s%s%s>", tagName, attrStr, closing)
	})

	return result, nil
}

// 判断文件并下载
func PrepareFilepath(filepathOrURL string) (string, bool, error) {
	// 解析URL
	u, err := url.Parse(filepathOrURL)
	if err != nil {
		return "", false, fmt.Errorf("无效的URL: %w", err)
	}

	// 如果不是http/https协议，当作本地文件处理
	if !strings.HasPrefix(u.Scheme, "http") {
		// 检查本地文件是否存在
		if _, err := os.Stat(filepathOrURL); err != nil {
			return "", false, fmt.Errorf("本地文件不存在: %w", err)
		}
		return filepathOrURL, false, nil
	}

	// 计算URL的MD5值作为文件名的一部分，防止文件名冲突
	hash := md5.Sum([]byte(filepathOrURL))
	hashStr := hex.EncodeToString(hash[:])

	// 从URL中提取文件名，如果没有则使用默认名称
	filename := filepath.Base(u.Path)
	if filename == "" || filename == "/" || filename == "." {
		filename = "downloaded.xlsx"
	}

	// 组合最终的文件路径: hash_filename
	finalFilename := fmt.Sprintf("%s_%s", hashStr, filename)
	finalPath := filepath.Join(os.TempDir(), finalFilename)

	// 检查文件是否存在
	if _, err := os.Stat(finalPath); err == nil {
		log.Printf("使用缓存文件: %s", finalPath)
		return finalPath, false, nil
	}

	// 下载文件到临时文件
	tempFile, err := DownloadFile(filepathOrURL)
	if err != nil {
		return "", false, fmt.Errorf("下载文件失败: %w", err)
	}

	// 注意：在某些系统上（如Windows），如果目标文件已存在，Rename可能会失败
	if err := os.Rename(tempFile, finalPath); err != nil {
		// 如果重命名失败，检查目标文件是否已存在（可能是并发下载导致的）
		if _, statErr := os.Stat(finalPath); statErr == nil {
			os.Remove(tempFile) // 清理多余的临时文件
			log.Printf("并发下载，使用已存在的文件: %s", finalPath)
			return finalPath, true, nil
		}
		os.Remove(tempFile) // 清理临时文件
		return "", false, fmt.Errorf("重命名临时文件失败: %w", err)
	}

	return finalPath, true, nil
}

// 下载文件
func DownloadFile(urlStr string) (string, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "excel-download-*.xlsx")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}

	// 写入文件
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	return tempFile.Name(), nil
}

func FormatContent(content string) string {
	content = FormatTag(content)
	// content = FormatLatexInMathBlocks(content)
	content = RemoveSpacesBeforeMathBlockAndLineBreak(content)
	content = RemoveEmptyLines(content)
	content = RemoveFirstLineDash(content)
	return content
}

func FormatTag(content string) string {
	// 将内联 s-tag 转成块级：在前后补充空行让 markdown 解析为块级 HTML
	content = strings.ReplaceAll(content, "<s-tag", "\n\n<s-tag")
	content = strings.ReplaceAll(content, "</s-tag>", "</s-tag>\n\n")
	content = strings.ReplaceAll(content, "$\\", "$")
	return content
}

func RemoveEmptyLines(content string) string {
	// 编译正则表达式
	re := regexp.MustCompile(`\n{2,}`)

	// 使用正则表达式替换
	result := re.ReplaceAllString(content, "\n\n")
	return result
}

func RemoveFirstLineDash(content string) string {
	re := regexp.MustCompile(`^(\s*\n)*[-]{3,}\s*\n`)

	// 使用正则表达式替换
	result := re.ReplaceAllString(content, "")

	return result
}

// getMergedCells 获取工作表中的所有合并单元格，并根据偏移量和范围调整坐标
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

		// 转换为0-based索引
		startRow--
		endRow2--
		startCol--
		endCol2--

		// 检查合并单元格是否在指定的范围内
		// 如果设置了 endRow，检查是否超出范围
		if endRow > 0 && startRow > endRow {
			continue // 合并单元格起始行超出结束行，跳过
		}
		// 如果设置了 endCol，检查是否超出范围
		if endCol > 0 && startCol > endCol {
			continue // 合并单元格起始列超出结束列，跳过
		}

		// 调整坐标：减去偏移量，只保留在有效范围内的合并单元格
		adjustedStartRow := startRow - offsetRow
		adjustedEndRow := endRow2 - offsetRow
		adjustedStartCol := startCol - offsetCol
		adjustedEndCol := endCol2 - offsetCol

		// 如果设置了结束位置，裁剪合并单元格的范围
		if endRow > 0 && endRow2 > endRow {
			adjustedEndRow = endRow - offsetRow
		}
		if endCol > 0 && endCol2 > endCol {
			adjustedEndCol = endCol - offsetCol
		}

		// 只保留至少部分在有效范围内的合并单元格
		if adjustedEndRow >= 0 && adjustedEndCol >= 0 {
			// 确保起始位置不小于0
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

// buildMergeMap 构建合并单元格的查找映射
func BuildMergeMap(mergedCells []MergedCell) map[string]MergedCell {
	mergeMap := make(map[string]MergedCell)

	for _, merge := range mergedCells {
		// 只为合并区域的左上角单元格创建映射
		key := fmt.Sprintf("%d-%d", merge.StartRow, merge.StartCol)
		mergeMap[key] = merge
	}

	return mergeMap
}

// getSheetConfig 获取指定工作表的配置，并将1-based索引转换为0-based索引
func GetSheetConfig(sheetName string, configs []SheetConfig) SheetConfig {
	var config SheetConfig
	found := false

	// 首先查找精确匹配的配置
	for _, cfg := range configs {
		if cfg.Name == sheetName {
			config = cfg
			found = true
			break
		}
	}

	// 然后查找通配符配置（Name为空）
	if !found {
		for _, cfg := range configs {
			if cfg.Name == "" {
				config = cfg
				found = true
				break
			}
		}
	}

	// 如果没有找到配置，返回默认配置
	if !found {
		config = SheetConfig{
			Name:     sheetName,
			StartRow: 1, // 默认从第1行开始
			StartCol: 1, // 默认从第1列开始
			EndRow:   0, // 默认读取到最后一行
			EndCol:   0, // 默认读取到最后一列
		}
	}

	// 将1-based索引转换为0-based索引
	// 如果用户设置为0或负数，StartRow/StartCol 视为1
	if config.StartRow <= 0 {
		config.StartRow = 1
	}
	if config.StartCol <= 0 {
		config.StartCol = 1
	}

	// 转换为0-based
	config.StartRow--
	config.StartCol--

	// 处理 EndRow 和 EndCol
	// 0 表示读取到最后，保持为0
	// 大于0的值需要转换为0-based
	if config.EndRow > 0 {
		config.EndRow-- // 转换为0-based
		// 验证 EndRow 不能小于 StartRow
		if config.EndRow < config.StartRow {
			log.Printf("警告: 工作表 %s 的结束行（%d）小于开始行（%d），将调整为开始行", sheetName, config.EndRow+1, config.StartRow+1)
			config.EndRow = config.StartRow
		}
	}

	if config.EndCol > 0 {
		config.EndCol-- // 转换为0-based
		// 验证 EndCol 不能小于 StartCol
		if config.EndCol < config.StartCol {
			log.Printf("警告: 工作表 %s 的结束列（%d）小于开始列（%d），将调整为开始列", sheetName, config.EndCol+1, config.StartCol+1)
			config.EndCol = config.StartCol
		}
	}

	return config
}

// isLastSectionLandscape 检查文档中最后一个定义了纸张方向的分节符是否为横向
func IsLastSectionLandscape(doc *document.Document) bool {
	return doc.IsLastSectionLandscape()
}
