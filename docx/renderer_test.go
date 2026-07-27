package docx_test

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/common"
	"github.com/scoming-dev/tools/docx/plugins/cover"

	"github.com/88250/lute/html"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
)

func TestDocxRenderer(t *testing.T) {
	md := `
1. 敏感性分析  
项目的敏感性分析是通过考察项目涉及的各种不确定因素对项目基本方案经济评价指标（如财务内部收益率 FIRR）的影响，找出敏感因素，估计项目效益对它们的敏感程度，粗略预测项目可能承担的风险。  
敏感度系数计算公式如下：  
$$SAF=(\\Delta A\/A)\div(\\Delta F/F)$$
式中：  
SAF—— 评价指标 A 对于不确定因素 F 的敏感度系数；  
ΔA/A —— 不确定因素 F 发生 ΔF 变体时，评价指标 A 的相应变化率（%）；  
ΔF/F —— 不确定因素 F 的变化率（%）。  
可结合本项目的具体情况，确定固定资产投资、经营成本及经营收入三个要素，采取多种变化幅度为来测定财务内部收益率受影响的变化程度，计算敏感度系数。

2. 结合本项目的具体情况，确定敏感性分析要素，采取多种变化幅度为来测定分析对象受影响的变化程度，计算敏感度系数。本项目以项目政府专项债融资本息覆盖倍数作为敏感性分析对象，考虑项目实施过程中一些不确定因素的变化，分别对营业收入、经营成本考虑变化率 -8.00%、-6.00%、-4.00%、-2.00%、0.00%、2.00%、4.00%、6.00%、8.00% 的单因素变化影响平均偿债备付率影响的敏感性分析。

| **变化因素** | **-8.00%** | **-6.00%** | **-4.00%** | **-2.00%** | **0.00%** | **2.00%** | **4.00%** | **6.00%** | **8.00%** |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 营业收入 | 1.14 | 1.18 | 1.22 | 1.25 | 1.29 | 1.33 | 1.37 | 1.41 | 1.45 |
| 经营成本 | 1.33 | 1.32 | 1.31 | 1.30 | 1.29 | 1.29 | 1.28 | 1.27 | 1.26 |
| 利率 | 1.33 | 1.32 | 1.31 | 1.30 | 1.29 | 1.29 | 1.28 | 1.27 | 1.26 |
| 发行政府专项债融资本金 | 1.36 | 1.35 | 1.33 | 1.31 | 1.29 | 1.28 | 1.26 | 1.25 | 1.23 |

3. 通过测算，经营收入、建设投资变化的变化对项目效益的影响最大，最为敏感，经营成本变化相对影响较小。  
通过敏感性分析，固定资产投资、经营成本及经营收入等因素都有抗风险能力，尤其是建设投资的抗风险能力最强，经营收入的变化最敏感。因此合理确定运营定价收入相关定价以及控制工程运行成本都将直接影响项目的收入。

	`

	md = formatContent(md)
	parseOptions := parse.NewOptions()
	parseOptions.HTMLTag2TextMark = true
	parseOptions.Spin = true
	tree := parse.Parse("", []byte(md), parseOptions)
	renderOptions := render.NewOptions()
	renderOptions.SoftBreak2HardBreak = false
	renderOptions.RenderListStyle = true

	html := render.NewHtmlRenderer(tree, renderOptions)
	htmlContent := html.Render()
	if len(htmlContent) == 0 {
		t.Fatal("expected non-empty HTML render output")
	}

	renderer := docx.NewDocxRenderer(tree, renderOptions, 0, docx.WithCover(cover.Report()))
	renderer.Render()

	outputPath := filepath.Join("tmp", "renderer.docx")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	if err := renderer.Save(outputPath); err != nil {
		t.Fatalf("save rendered docx: %v", err)
	}
	t.Logf("rendered docx: %s", outputPath)

	if info, err := os.Stat(outputPath); err != nil {
		t.Fatalf("stat rendered docx: %v", err)
	} else if info.Size() == 0 {
		t.Fatal("expected rendered docx to be non-empty")
	}
}

func TestDocxRendererOrderedListUsesDocxNumbering(t *testing.T) {
	files := renderDocxParts(t, "3) first\n4) second\n")
	documentXML := files["word/document.xml"]
	numberingXML := files["word/numbering.xml"]

	if !strings.Contains(documentXML, "w:numPr") {
		t.Fatalf("expected document.xml to contain docx numbering properties, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "w:firstLine") {
		t.Fatalf("expected document.xml to contain list first-line indent, got: %s", documentXML)
	}
	if !strings.Contains(numberingXML, `w:numFmt w:val="decimal"`) {
		t.Fatalf("expected numbering.xml to contain a decimal numbering definition, got: %s", numberingXML)
	}
	if !strings.Contains(numberingXML, `w:start w:val="3"`) {
		t.Fatalf("expected numbering.xml to preserve ordered list start 3, got: %s", numberingXML)
	}
	if !strings.Contains(numberingXML, `w:lvlText w:val="%1)"`) {
		t.Fatalf("expected numbering.xml to preserve ordered list delimiter ), got: %s", numberingXML)
	}
	if !strings.Contains(numberingXML, `w:suff w:val="space"`) {
		t.Fatalf("expected numbering.xml to use a single-space list marker suffix, got: %s", numberingXML)
	}
	if strings.Contains(documentXML, "<w:t>3)</w:t>") || strings.Contains(documentXML, "<w:t>4)</w:t>") {
		t.Fatalf("ordered markers should be docx numbering, not text runs: %s", documentXML)
	}
	if strings.Contains(numberingXML, "<w:ind") {
		t.Fatalf("numbering definition should not set whole-paragraph indent, got: %s", numberingXML)
	}
}

func renderDocxParts(t *testing.T, markdown string) map[string]string {
	t.Helper()

	tree := parse.Parse("", []byte(markdown), parse.NewOptions())
	renderer := docx.NewDocxRenderer(tree, render.NewOptions(), 0)
	renderer.Render()

	outputPath := filepath.Join(t.TempDir(), "list.docx")
	if err := renderer.Save(outputPath); err != nil {
		t.Fatalf("save rendered docx: %v", err)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("open rendered docx as zip: %v", err)
	}
	defer reader.Close()

	files := map[string]string{}
	for _, file := range reader.File {
		if file.Name != "word/document.xml" && file.Name != "word/numbering.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		content, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			t.Fatalf("read %s: %v", file.Name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close %s: %v", file.Name, closeErr)
		}
		files[file.Name] = string(content)
	}

	if _, ok := files["word/document.xml"]; !ok {
		t.Fatal("rendered docx does not contain word/document.xml")
	}
	if _, ok := files["word/numbering.xml"]; !ok {
		t.Fatal("rendered docx does not contain word/numbering.xml")
	}
	return files
}

func formatContent(content string) string {
	content = formatTag(content)
	content = common.FormatLatexInMathBlocks(content)
	content = common.RemoveSpacesBeforeMathBlockAndLineBreak(content)
	content = removeEmptyLines(content)
	content = removeFirstLineDash(content)
	return content
}

func formatTag(content string) string {
	content = strings.ReplaceAll(content, "<s-tag", "\n\n<s-tag")
	content = strings.ReplaceAll(content, "</s-tag>", "</s-tag>\n\n")
	content = strings.ReplaceAll(content, "$\\", "$")
	return content
}

// RemoveSpacesBeforeMathBlockAndLineBreak 去掉块级公式分隔符和换行符前面的空格
func RemoveSpacesBeforeMathBlockAndLineBreak(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "$$" {
			line = "$$"
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func removeEmptyLines(content string) string {
	// 编译正则表达式
	re := regexp.MustCompile(`\n{2,}`)

	// 使用正则表达式替换
	result := re.ReplaceAllString(content, "\n\n")
	return result
}

func removeFirstLineDash(content string) string {
	re := regexp.MustCompile(`^(\s*\n)*[-]{3,}\s*\n`)

	// 使用正则表达式替换
	result := re.ReplaceAllString(content, "")

	return result
}

// 获取节点
func GetNodes(content string, selector string) (nodes []*html.Node, err error) {
	err = nil
	htmlTree, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	var findNodes func(*html.Node) error
	findNodes = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == selector {
			nodes = append(nodes, n)
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
	return nodes, err
}

// modifyHTMLTag 使用正则表达式修改 HTML 标签属性
func EditHtmlAttribute(htmlContent, selector string, filter map[string]string, key, val string) string {
	// 匹配标签，例如 <s-tag type='cwcs' url='' name='xxx'>
	re := regexp.MustCompile(fmt.Sprintf(`(?i)<(%s)(\s[^>]*)?(/?)>`, regexp.QuoteMeta(selector)))

	result := re.ReplaceAllStringFunc(htmlContent, func(match string) string {
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

	return result
}
