package docx_test

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/common"
	"github.com/scoming-dev/tools/docx/plugins/cover"

	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
)

func TestDocxRenderer(t *testing.T) {
	md := `
# 7. 投资估算编制依据与原则
## 7.1 投资估算编制依据与原则
### 7.1.1 投资估算编制依据与原则

本项目投资估算严格遵循国家及四川省现行有关法律、法规及工程建设定额标准，结合遂宁市蓬溪县当地市场价格信息及项目建设实际条件进行编制。主要依据如下：

1.  **政策法规与定额标准**：
    *   《四川省建设工程工程量清单计价定额》（2020年版）；
    *   《四川省市政工程计价定额》及相关配套文件；
    *   《建设项目经济评价方法与参数》（第三版）；
    *   《市政工程投资估算编制办法》；
    *   国家发展改革委、建设部关于工程建设其他费用定额的相关规定。

2.  **价格依据**：
    *   人工、材料、机械台班价格参照遂宁市工程造价管理站发布的最新一期《工程造价信息》及周边地区同期市场价格；
    *   设备价格采用询价或参照近期同类工程采购合同价；
    *   土地征用及拆迁补偿标准依据遂宁市及蓬溪县人民政府有关文件执行。

3.  **工程量与技术资料**：
    *   项目可行性研究报告及初步设计图纸；
    *   项目拟建场址的自然地理条件、工程地质资料；
    *   类似工程的技术经济指标及造价资料。

4.  **取费标准**：
    *   建设单位管理费按照财建[2016]504号文计取；
    *   工程监理费参照发改价格(2007)670号文计取；
    *   工程勘察设计费参照计价格[2002]10号文计取；
    *   预备费按工程费用与工程建设其他费用之和的8%估算。

### 7.1.2 项目建设总投资估算

本项目建设总投资估算为 **9,780.00万元**。投资构成主要包括第一部分工程费用、第二部分工程建设其他费用以及第三部分预备费。

#### 7.1.2.1 第一部分 工程费用
工程费用合计 **8,232.00万元**，占项目总投资的84.17%。涵盖了景区服务区改造、道路交通体系构建、景观打造、停车及充电设施、公用工程及智慧旅游系统等核心建设内容。

1.  **景区服务区改造工程**：估算投资 **2,340.00万元**。
    *   包含拆除工程（117.00万元）、结构加固工程（351.00万元）以及装饰装修工程（936.00万元）。
    *   安装工程配套投资936.00万元，具体细分为给排水（234.00万元）、强电（280.80万元）、弱电（187.20万元）、消防（140.40万元）及暖通工程（93.60万元），全面提升服务区功能品质。

2.  **景区道路工程**：估算投资 **2,990.00万元**。
    *   作为投资占比最大的单项工程（30.57%），主要用于13公里景区道路的提质升级。
    *   其中路面工程投资1,345.50万元，路基工程747.50万元，排水工程448.50万元，交通安全设施299.00万元，边坡防护工程149.50万元。

3.  **景观及步行道工程**：估算投资 **1,060.30万元**。
    *   重点打造16580平方米芍药种植艺术田园，投资636.18万元（含地形整理、土壤改良、灌溉系统及景观小品）。
    *   8公里步行道工程投资424.12万元，旨在完善景区慢行游览体系。

4.  **停车场及配套设施工程**：估算投资 **962.50万元**。
    *   生态停车场建设（土建部分）577.50万元。
    *   **充电桩设施**：设备购置及安装费 **336.88万元**。根据市场调研与成本分析，本项目配置140个充电桩（含直流快充与交流慢充），综合单价约2.4万元/个，符合当前主流设备及安装调试成本区间。
    *   场内管网及标识工程48.13万元。

5.  **公用设施工程**：估算投资 **525.00万元**。
    *   新建微型污水处理站1座，投资315.00万元，其中设备购置及安装189.00万元，土建工程126.00万元。
    *   电力管网工程（5公里）投资210.00万元，保障景区电力供应稳定性。

6.  **智慧旅游及其他工程**：估算投资 **354.20万元**。
    *   包括智慧旅游系统平台搭建（177.10万元）、安防监控系统（88.55万元）、信息发布及广告位（53.13万元）以及票务及导览系统（35.42万元），全面提升景区数字化管理水平。

#### 7.1.2.2 第二部分 工程建设其他费用
工程建设其他费用合计 **822.07万元**，占项目总投资的8.41%。主要费用项目如下：
*   **工程设计费**：179.22万元；
*   **工程监理费**：129.75万元；
*   **建设单位管理费**：108.89万元；
*   **工程造价咨询服务费**：82.32万元；
*   **水土保持方案编制费**：67.86万元；
*   **工程勘察费**：46.10万元；
*   **工程检测费**及**场地准备及临时设施费**：各41.16万元；
*   其他包括前期工作咨询、环评、招代、图审、保险及安评等费用共计125.61万元。

#### 7.1.2.3 第三部分 预备费
本项目预备费为 **725.93万元**，占项目总投资的7.42%。主要用于解决在项目实施过程中可能发生的难以预料的工程变更、自然灾害等风险因素导致的费用增加，确保项目资金安全。

### 7.1.3 资金筹措方案

根据项目总投资估算（9,780.00万元）及项目建设进度安排，本项目拟采用多渠道筹资方式，确保建设资金及时足额到位。

1.  **资金来源结构**
    *   **财政资金与专项债券**：计划申请地方政府专项债券或上级补助资金 **6,846.00万元**，占总投资的70%。鉴于本项目属于旅游基础设施建设，符合国家关于支持文化旅游产业发展及基础设施补短板的政策导向，具备申请专项债的条件。
    *   **企业自筹资金**：项目单位自筹资金 **1,956.00万元**，占总投资的20%。主要由项目业主单位通过自有资金注入，作为项目资本金。
    *   **银行贷款**：计划向商业银行或政策性银行申请中长期项目贷款 **978.00万元**，占总投资的10%，用于补充建设期资金缺口。

2.  **资金使用计划**
    *   资金投入将严格按照工程建设进度进行拨付。预计建设期第一年投入总投资的40%，主要用于征拆、土建基础及设备订货；第二年投入60%，用于主体工程建设、装修、设备安装调试及竣工验收。

3.  **融资风险管控**
    *   项目单位将建立严格的资金管理制度，设立专用账户，专款专用。同时，积极与金融机构对接，落实贷款条件，确保融资渠道畅通，降低融资成本，保障资金链安全。

$$I_{total} = C_{eng} + C_{other} + C_{res}$$

其中：$I_{total}$ 为项目总投资（9780.00万元）；$C_{eng}$ 为工程费用（8232.00万元）；$C_{other}$ 为工程建设其他费用（822.07万元）；$C_{res}$ 为预备费（725.93万元）。

![](https://dp-static.obs.cn-southwest-2.myhuaweicloud.com/file/601914931079561219.svg)
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

	renderer := docx.NewDocxRenderer(
		tree,
		renderOptions,
		docx.HeadingStyleLevelOneCenterPageBreak,
		docx.WithCover(cover.SpecialDebt()),
	)
	renderer.Render()

	outputPath := rendererTestOutputPath(t)
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
	if !strings.Contains(numberingXML, `w:numFmt w:val="chineseCounting"`) {
		t.Fatalf("expected numbering.xml to contain top-level Chinese numbering, got: %s", numberingXML)
	}
	if !strings.Contains(numberingXML, `w:start w:val="3"`) {
		t.Fatalf("expected numbering.xml to preserve ordered list start 3, got: %s", numberingXML)
	}
	if !strings.Contains(numberingXML, `w:lvlText w:val="（%1）"`) {
		t.Fatalf("expected numbering.xml to use top-level Chinese parentheses, got: %s", numberingXML)
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

func TestDocxRendererUnorderedListUsesNumericNumbering(t *testing.T) {
	files := renderDocxParts(t, `1. 一级
    * 子项一
    * 子项二
`)
	documentXML := files["word/document.xml"]
	numberingXML := files["word/numbering.xml"]

	if strings.Contains(numberingXML, `w:numFmt w:val="bullet"`) {
		t.Fatalf("ordinary unordered lists should use numeric numbering, got: %s", numberingXML)
	}
	if !strings.Contains(numberingXML, `w:numFmt w:val="decimal"`) ||
		!strings.Contains(numberingXML, `w:lvlText w:val="%1."`) {
		t.Fatalf("expected nested unordered list to render as decimal numbering, got: %s", numberingXML)
	}
	if strings.Contains(documentXML, "<w:t>•</w:t>") {
		t.Fatalf("ordinary unordered list markers should not be text bullets: %s", documentXML)
	}
}

func TestDocxRendererNestedOrderedListUsesConfiguredNumberingStyles(t *testing.T) {
	files := renderDocxParts(t, `1. 一级
    1. 二级
        1. 三级
            1. 四级
                1. 五级
                    1. 六级
                        1. 七级
`)
	documentXML := files["word/document.xml"]
	numberingXML := files["word/numbering.xml"]

	required := []string{
		`w:numFmt w:val="chineseCounting"`,
		`w:numFmt w:val="decimal"`,
		`w:numFmt w:val="decimalEnclosedCircle"`,
		`w:numFmt w:val="upperLetter"`,
		`w:numFmt w:val="lowerLetter"`,
		`w:lvlText w:val="%1."`,
		`w:lvlText w:val="%1）"`,
		`w:lvlText w:val="%1"`,
	}
	for _, want := range required {
		if !strings.Contains(numberingXML, want) {
			t.Fatalf("expected numbering.xml to contain %q, got: %s", want, numberingXML)
		}
	}
	if strings.Count(numberingXML, `w:lvlText w:val="（%1）"`) < 2 {
		t.Fatalf("expected top and third nested levels to use full-width parentheses, got: %s", numberingXML)
	}
	if strings.Count(numberingXML, `w:lvlText w:val="%1."`) < 3 {
		t.Fatalf("expected decimal, upper-letter, and lower-letter levels to use dot suffixes, got: %s", numberingXML)
	}
	if strings.Count(documentXML, "w:numPr") < 7 {
		t.Fatalf("expected all nested items to use docx numbering properties, got: %s", documentXML)
	}
	if strings.Contains(documentXML, "<w:t>1.</w:t>") || strings.Contains(documentXML, "<w:t>（一）</w:t>") {
		t.Fatalf("ordered markers should be docx numbering, not text runs: %s", documentXML)
	}
}

func TestDocxRendererTOCUsesWPSFieldStructure(t *testing.T) {
	files, _ := renderDocxArchive(t, "# 第一章\n\n## 第一节\n\n正文", docx.WithCover(cover.Report()))
	documentXML := files["word/document.xml"]

	required := []string{
		`<w:fldChar w:fldCharType="begin" w:dirty="true"/>`,
		`<w:instrText xml:space="preserve">TOC \o "1-3" \h \z \u</w:instrText>`,
		`<w:fldChar w:fldCharType="separate"/>`,
		`<w:fldChar w:fldCharType="end"/>`,
		`w:pStyle w:val="TOC1"`,
		`w:pStyle w:val="TOC2"`,
		`<w:tab w:val="right" w:leader="dot" w:pos="8600"/>`,
		`<w:tab/>`,
		`PAGEREF _Toc0 \h`,
		`PAGEREF _Toc1 \h`,
		`w:hyperlink w:anchor="_Toc0"`,
		`w:pStyle w:val="Heading1"`,
		`w:pStyle w:val="Heading2"`,
	}
	for _, want := range required {
		if !strings.Contains(documentXML, want) {
			t.Fatalf("expected document.xml to contain %q, got: %s", want, documentXML)
		}
	}
	if strings.Contains(documentXML, "目录待更新") {
		t.Fatalf("TOC with headings should render seeded entries, not only a placeholder: %s", documentXML)
	}
	if strings.Count(documentXML, "<w:t>第一章</w:t>") < 2 ||
		strings.Count(documentXML, "<w:t>第一节</w:t>") < 2 {
		t.Fatalf("expected TOC seeded entries and body headings to both be present, got: %s", documentXML)
	}
	fieldEnd := strings.LastIndex(documentXML, `<w:fldChar w:fldCharType="end"/>`)
	if fieldEnd < 0 {
		t.Fatalf("expected TOC field end, got: %s", documentXML)
	}
	pageBreakAfterTOC := strings.Index(documentXML[fieldEnd:], `<w:br w:type="page"`)
	bodyHeading := strings.Index(documentXML, `w:pStyle w:val="Heading1"`)
	if pageBreakAfterTOC < 0 || bodyHeading < 0 || fieldEnd+pageBreakAfterTOC > bodyHeading {
		t.Fatalf("expected a page break after TOC before body headings, got: %s", documentXML)
	}

	for _, legacy := range []string{"toc-div", "toc-a", "toc-h"} {
		if strings.Contains(documentXML, legacy) {
			t.Fatalf("TOC should use a WPS field, not legacy HTML %q, got: %s", legacy, documentXML)
		}
	}
}

func TestDocxRendererHeadingStyleOptionLimitsHeadingRendering(t *testing.T) {
	files, _ := renderDocxArchive(t, "# 第一章", docx.WithHeadingStyle(docx.HeadingStyleLevelOneCenterPageBreak))
	documentXML := files["word/document.xml"]

	for _, want := range []string{
		`w:pStyle w:val="Heading1"`,
		`<w:jc w:val="center"`,
		`<w:br w:type="page"`,
		`<w:t>第一章</w:t>`,
	} {
		if !strings.Contains(documentXML, want) {
			t.Fatalf("expected document.xml to contain %q, got: %s", want, documentXML)
		}
	}
}

func TestDocxRendererInlineMathUsesNativeOMML(t *testing.T) {
	files, names := renderDocxArchive(t, `before $a^2+\frac{b}{c}$ after`)
	documentXML := files["word/document.xml"]

	if !strings.Contains(documentXML, `xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"`) {
		t.Fatalf("expected document.xml to declare the Office Math namespace, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<m:oMath>") {
		t.Fatalf("expected document.xml to contain native inline OMML, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<m:f>") {
		t.Fatalf("expected fraction to render as native OMML, got: %s", documentXML)
	}
	if strings.Contains(documentXML, "<m:oMathPara>") {
		t.Fatalf("inline-only math should not render block OMML, got: %s", documentXML)
	}
	paragraphs := documentParagraphs(documentXML)
	if len(paragraphs) < 1 {
		t.Fatalf("expected inline math paragraph, got: %s", documentXML)
	}
	if !strings.Contains(paragraphs[0], "<m:oMath>") || strings.Contains(paragraphs[0], "<m:oMathPara>") {
		t.Fatalf("inline math should render as m:oMath in the surrounding text paragraph, got: %s", paragraphs[0])
	}
	if !strings.Contains(documentXML, `<w:t xml:space="preserve">before </w:t>`) ||
		!strings.Contains(documentXML, `<w:t xml:space="preserve"> after</w:t>`) {
		t.Fatalf("inline math should preserve surrounding text, got: %s", documentXML)
	}
	assertMathDoesNotUseImages(t, documentXML, names)
}

func TestDocxRendererInlineAndBlockMathUsesNativeOMML(t *testing.T) {
	files, names := renderDocxArchive(t, `before $a^2+\frac{b}{c}$ after

$$
\sqrt{x}
$$
`)
	documentXML := files["word/document.xml"]

	if !strings.Contains(documentXML, `xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"`) {
		t.Fatalf("expected document.xml to declare the Office Math namespace, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<m:oMath>") {
		t.Fatalf("expected document.xml to contain native inline OMML, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<m:oMathPara>") {
		t.Fatalf("expected document.xml to contain native block OMML, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<m:f>") {
		t.Fatalf("expected fraction to render as native OMML, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<m:rad>") {
		t.Fatalf("expected square root to render as native OMML, got: %s", documentXML)
	}
	paragraphs := documentParagraphs(documentXML)
	if len(paragraphs) < 2 {
		t.Fatalf("expected inline and block math paragraphs, got: %s", documentXML)
	}
	if !strings.Contains(paragraphs[0], "<m:oMath>") || strings.Contains(paragraphs[0], "<m:oMathPara>") {
		t.Fatalf("inline math should render as m:oMath in the surrounding text paragraph, got: %s", paragraphs[0])
	}
	if !strings.Contains(paragraphs[1], "<m:oMathPara>") || !strings.Contains(paragraphs[1], `w:jc w:val="center"`) {
		t.Fatalf("block math should render as a centered m:oMathPara paragraph, got: %s", paragraphs[1])
	}
	if !strings.Contains(documentXML, `<w:t xml:space="preserve">before </w:t>`) ||
		!strings.Contains(documentXML, `<w:t xml:space="preserve"> after</w:t>`) {
		t.Fatalf("inline math should preserve surrounding text, got: %s", documentXML)
	}
	assertMathDoesNotUseImages(t, documentXML, names)
}

func TestDocxRendererMathUsesContentSize(t *testing.T) {
	config := docx.DefaultConfig()
	config.Text.ContentSize = 18

	files, _ := renderDocxArchive(t, `inline $x$

$$
y
$$
`, docx.WithConfig(config))
	documentXML := files["word/document.xml"]

	inlineMath := `<m:r><w:rPr><w:sz w:val="36"/><w:szCs w:val="36"/></w:rPr><m:t>x</m:t></m:r>`
	if !strings.Contains(documentXML, inlineMath) {
		t.Fatalf("inline math should use ContentSize 18pt, got: %s", documentXML)
	}
	blockMath := `<m:r><w:rPr><w:sz w:val="36"/><w:szCs w:val="36"/></w:rPr><m:t>y</m:t></m:r>`
	if !strings.Contains(documentXML, blockMath) {
		t.Fatalf("block math should use ContentSize 18pt, got: %s", documentXML)
	}
}

func TestDocxRendererEmbedsSVGWithoutRasterConversion(t *testing.T) {
	svgPath := filepath.Join(t.TempDir(), "logo.svg")
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="60"><rect width="120" height="60" fill="#336699"/></svg>`
	if err := os.WriteFile(svgPath, []byte(svgContent), 0o644); err != nil {
		t.Fatalf("write svg fixture: %v", err)
	}

	files, names := renderDocxArchive(t, "![logo]("+svgPath+")")
	documentXML := files["word/document.xml"]
	contentTypesXML := files["[Content_Types].xml"]
	documentRelsXML := files["word/_rels/document.xml.rels"]

	if strings.Contains(documentXML, "__DOCX_SVG_IMAGE_") {
		t.Fatalf("svg placeholder should be replaced before saving, got: %s", documentXML)
	}
	if !strings.Contains(documentXML, "<w:drawing>") || !strings.Contains(documentXML, `r:embed="rIdDocxSVGImage`) {
		t.Fatalf("expected SVG to render as an inline drawing, got: %s", documentXML)
	}
	if !strings.Contains(contentTypesXML, `Extension="svg"`) || !strings.Contains(contentTypesXML, `ContentType="image/svg+xml"`) {
		t.Fatalf("expected [Content_Types].xml to register svg content type, got: %s", contentTypesXML)
	}
	if !strings.Contains(documentRelsXML, `Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"`) ||
		!strings.Contains(documentRelsXML, `Target="media/image1.svg"`) {
		t.Fatalf("expected document relationships to reference embedded svg media, got: %s", documentRelsXML)
	}

	svgMediaName := ""
	for _, name := range names {
		if strings.HasPrefix(name, "word/media/") && strings.HasSuffix(name, ".svg") {
			svgMediaName = name
		}
		if strings.HasPrefix(name, "word/media/") && strings.HasSuffix(name, ".png") {
			t.Fatalf("svg should not be rasterized to png, found media file: %s", name)
		}
	}
	if svgMediaName == "" {
		t.Fatalf("expected svg media file in docx archive, files: %v", names)
	}
	if !strings.Contains(files[svgMediaName], `<svg`) {
		t.Fatalf("expected embedded svg media to keep original SVG XML, got: %s", files[svgMediaName])
	}
}

func renderDocxParts(t *testing.T, markdown string) map[string]string {
	t.Helper()

	files, _ := renderDocxArchive(t, markdown)
	required := []string{"word/document.xml", "word/numbering.xml"}
	for _, name := range required {
		if _, ok := files[name]; !ok {
			t.Fatalf("rendered docx does not contain %s", name)
		}
	}
	return files
}

func renderDocxArchive(t *testing.T, markdown string, rendererOptions ...docx.RendererOption) (map[string]string, []string) {
	t.Helper()

	markdown = formatContent(markdown)
	parseOptions := parse.NewOptions()
	parseOptions.HTMLTag2TextMark = true
	parseOptions.Spin = true
	tree := parse.Parse("", []byte(markdown), parseOptions)
	renderOptions := render.NewOptions()
	renderOptions.SoftBreak2HardBreak = false
	renderOptions.RenderListStyle = true

	renderer := docx.NewDocxRenderer(tree, renderOptions, 0, rendererOptions...)
	renderer.Render()

	outputPath := rendererTestOutputPath(t)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	if err := renderer.Save(outputPath); err != nil {
		t.Fatalf("save rendered docx: %v", err)
	}
	t.Logf("rendered docx: %s", outputPath)

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("open rendered docx as zip: %v", err)
	}
	defer reader.Close()

	files := map[string]string{}
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
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
	return files, names
}

func rendererTestOutputPath(t *testing.T) string {
	t.Helper()

	name := regexp.MustCompile(`[^A-Za-z0-9_-]+`).ReplaceAllString(t.Name(), "_")
	if name == "" {
		name = "renderer"
	}
	filename := name + "_" + time.Now().Format("20060102150405.000000000") + ".docx"
	return filepath.Join("tmp", filename)
}

func documentParagraphs(documentXML string) []string {
	return regexp.MustCompile(`(?s)<w:p\b.*?</w:p>`).FindAllString(documentXML, -1)
}

func assertMathDoesNotUseImages(t *testing.T, documentXML string, names []string) {
	t.Helper()

	if strings.Contains(documentXML, "<w:drawing") {
		t.Fatalf("math should not render as an image drawing, got: %s", documentXML)
	}
	if strings.Contains(documentXML, "__DOCX_RAW_XML_") {
		t.Fatalf("raw XML placeholders should be replaced before saving, got: %s", documentXML)
	}
	for _, name := range names {
		if strings.HasPrefix(name, "word/media/") {
			t.Fatalf("math should not create media files, found: %s", name)
		}
	}
}

func formatContent(content string) string {
	content = common.FormatTag(content)
	content = common.FormatLatexInMathBlocks(content)
	content = common.RemoveSpacesBeforeMathBlockAndLineBreak(content)
	content = common.RemoveEmptyLines(content)
	content = common.RemoveFirstLineDash(content)
	return content
}
