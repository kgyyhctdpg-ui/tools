package markdown

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const docxDocumentPrefix = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
 xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
 xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
 xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"
 xmlns:v="urn:schemas-microsoft-com:vml"><w:body>`

const docxDocumentSuffix = `</w:body></w:document>`

func TestDOCXConvertsInlineAndBlockOMML(t *testing.T) {
	document := docxDocumentPrefix + `
<w:p><w:r><w:t>Inline </w:t></w:r><m:oMath><m:f><m:num><m:r><m:t>a</m:t></m:r></m:num><m:den><m:r><m:t>b</m:t></m:r></m:den></m:f></m:oMath><w:r><w:t> formula</w:t></w:r></w:p>
<w:p><w:r><m:oMathPara><m:oMath><m:rad><m:radPr><m:degHide m:val="on"/></m:radPr><m:e><m:r><m:t>x</m:t></m:r></m:e></m:rad></m:oMath></m:oMathPara></w:r></w:p>` + docxDocumentSuffix

	result := convertTestDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}, nil)
	if !strings.Contains(result.Markdown, `Inline $\frac{a}{b}$ formula`) {
		t.Fatalf("inline formula was not converted to LaTeX:\n%s", result.Markdown)
	}
	if !strings.Contains(result.Markdown, "$$\n\\sqrt{x}\n$$") {
		t.Fatalf("block formula was not converted to display LaTeX:\n%s", result.Markdown)
	}
}

func TestDOCXConvertsEmbeddedImageAndUsesAltText(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 1, 2, 3}
	document := docxDocumentPrefix + `<w:p><w:r><w:drawing><wp:inline><wp:docPr id="1" name="Picture 1" descr="formula preview"/><a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:embed="rId5"/></pic:blipFill></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>` + docxDocumentSuffix
	relationships := `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId5" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.png"/></Relationships>`

	result := convertTestDOCX(t, map[string][]byte{
		"word/document.xml":            []byte(document),
		"word/_rels/document.xml.rels": []byte(relationships),
		"word/media/image1.png":        png,
	}, nil)
	want := "![formula preview](data:image/png;base64," + base64.StdEncoding.EncodeToString(png) + ")"
	if !strings.Contains(result.Markdown, want) {
		t.Fatalf("embedded image was not converted to a data URI:\n%s", result.Markdown)
	}
}

func TestDOCXUsesCustomImageHandler(t *testing.T) {
	document := docxDocumentPrefix + `<w:p><w:r><w:pict><v:shape title="MathType equation"><v:imagedata r:id="rId9"/></v:shape></w:pict></w:r></w:p>` + docxDocumentSuffix
	relationships := `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId9" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/equation.wmf"/></Relationships>`
	var received Image
	handler := func(_ context.Context, image Image) (string, error) {
		received = image
		return "https://cdn.example.com/equation.wmf", nil
	}
	result := convertTestDOCX(t, map[string][]byte{
		"word/document.xml":            []byte(document),
		"word/_rels/document.xml.rels": []byte(relationships),
		"word/media/equation.wmf":      {1, 2, 3, 4},
	}, handler)
	if received.Name != "equation.wmf" || received.MIMEType != "image/wmf" || received.AltText != "MathType equation" {
		t.Fatalf("unexpected image passed to handler: %#v", received)
	}
	if !strings.Contains(result.Markdown, "![MathType equation](https://cdn.example.com/equation.wmf)") {
		t.Fatalf("custom image URL was not used:\n%s", result.Markdown)
	}
}

func TestDOCXTablePreservesFormattingNumbersFormulaAndImage(t *testing.T) {
	document := docxDocumentPrefix + `<w:tbl>
<w:tr><w:tc><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>Code</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Value</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>00123</w:t></w:r></w:p></w:tc><w:tc><w:p><m:oMath><m:sSup><m:e><m:r><m:t>x</m:t></m:r></m:e><m:sup><m:r><m:t>2</m:t></m:r></m:sup></m:sSup></m:oMath><w:r><w:drawing><wp:inline><wp:docPr id="2" descr="chart"/><a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:embed="rId2"/></pic:blipFill></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p></w:tc></w:tr>
</w:tbl>` + docxDocumentSuffix
	relationships := `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/chart.png"/></Relationships>`
	result := convertTestDOCX(t, map[string][]byte{
		"word/document.xml":            []byte(document),
		"word/_rels/document.xml.rels": []byte(relationships),
		"word/media/chart.png":         {0x89, 'P', 'N', 'G'},
	}, nil)
	for _, expected := range []string{"<strong>Code</strong>", "00123", "$x^{2}$", `<img src="data:image/png;base64,`} {
		if !strings.Contains(result.Markdown, expected) {
			t.Fatalf("table output does not contain %q:\n%s", expected, result.Markdown)
		}
	}
	if strings.Contains(result.Markdown, "**Code**") {
		t.Fatalf("table bold should use strong tags, got:\n%s", result.Markdown)
	}
}

func TestDOCXMergedTableUsesHTMLSpans(t *testing.T) {
	document := docxDocumentPrefix + `<w:tbl>
<w:tr><w:tc><w:tcPr><w:gridSpan w:val="2"/><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Merged header</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:tcPr><w:gridSpan w:val="2"/><w:vMerge/></w:tcPr><w:p/></w:tc></w:tr>
</w:tbl>` + docxDocumentSuffix
	result := convertTestDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}, nil)
	if !strings.Contains(result.Markdown, `<th colspan="2" rowspan="2">Merged header</th>`) {
		t.Fatalf("merged table did not preserve row and column spans:\n%s", result.Markdown)
	}
}

func TestDOCXDistinguishesMergedColumnsAndRows(t *testing.T) {
	document := docxDocumentPrefix + `<w:tbl>
<w:tr>
<w:tc><w:tcPr><w:hMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Columns</w:t></w:r></w:p></w:tc>
<w:tc><w:tcPr><w:hMerge/></w:tcPr><w:p/></w:tc>
<w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Rows</w:t></w:r></w:p></w:tc>
</w:tr>
<w:tr>
<w:tc><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>
<w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc>
<w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p/></w:tc>
</w:tr>
</w:tbl>` + docxDocumentSuffix
	result := convertTestDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}, nil)
	if !strings.Contains(result.Markdown, `<th colspan="2">Columns</th>`) {
		t.Fatalf("merged columns did not use colspan:\n%s", result.Markdown)
	}
	if !strings.Contains(result.Markdown, `<th rowspan="2">Rows</th>`) {
		t.Fatalf("merged rows did not use rowspan:\n%s", result.Markdown)
	}
	if strings.Contains(result.Markdown, `<th rowspan="2">Columns</th>`) || strings.Contains(result.Markdown, `<th colspan="2">Rows</th>`) {
		t.Fatalf("row and column merge attributes were mixed up:\n%s", result.Markdown)
	}
}

func TestDOCXReadsRowsAndCellsWrappedInContentControls(t *testing.T) {
	document := docxDocumentPrefix + `<w:tbl><w:sdt><w:sdtContent><w:tr>
<w:sdt><w:sdtContent><w:tc><w:sdt><w:sdtContent><w:p><w:r><w:t>wrapped header</w:t></w:r></w:p></w:sdtContent></w:sdt></w:tc></w:sdtContent></w:sdt>
<w:tc><w:p><w:r><w:t>100</w:t></w:r></w:p></w:tc>
</w:tr></w:sdtContent></w:sdt></w:tbl>` + docxDocumentSuffix
	result := convertTestDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}, nil)
	if !strings.Contains(result.Markdown, "wrapped header") || !strings.Contains(result.Markdown, "100") {
		t.Fatalf("wrapped table content was lost:\n%s", result.Markdown)
	}
}

func TestOMMLComplexStructures(t *testing.T) {
	root, err := parseXML([]byte(`<m:oMath xmlns:m="urn:math"><m:nary><m:naryPr><m:chr m:val="∑"/></m:naryPr><m:sub><m:r><m:t>i=1</m:t></m:r></m:sub><m:sup><m:r><m:t>n</m:t></m:r></m:sup><m:e><m:f><m:num><m:r><m:t>x</m:t></m:r></m:num><m:den><m:rad><m:e><m:r><m:t>y</m:t></m:r></m:e></m:rad></m:den></m:f></m:e></m:nary></m:oMath>`))
	if err != nil {
		t.Fatal(err)
	}
	latex := ommlToLatex(root.first("oMath"))
	want := `\sum_{i=1}^{n}\frac{x}{\sqrt{y}}`
	if latex != want {
		t.Fatalf("unexpected LaTeX:\nwant: %s\n got: %s", want, latex)
	}
}

func TestTableFormatHTMLAppliesToCSV(t *testing.T) {
	result, err := New(WithTableFormat(TableFormatHTML)).ConvertReader(
		context.Background(),
		strings.NewReader("Name,Value\nA,1"),
		StreamInfo{Name: "data.csv"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Markdown, "<table>") || !strings.Contains(result.Markdown, "<th>Name</th>") {
		t.Fatalf("HTML table format was not applied:\n%s", result.Markdown)
	}
}

func TestTablesUseHTMLByDefault(t *testing.T) {
	result, err := New().ConvertReader(
		context.Background(),
		strings.NewReader("Name,Value\nA,1"),
		StreamInfo{Name: "data.csv"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Markdown, "<table>") || strings.Contains(result.Markdown, "| --- |") {
		t.Fatalf("default table output should be HTML:\n%s", result.Markdown)
	}
}

func TestPPTXTablePreservesMergedRowsAndColumns(t *testing.T) {
	root, err := parseXML([]byte(`<a:tbl xmlns:a="urn:drawingml">
<a:tr>
<a:tc gridSpan="2"><a:p><a:r><a:t>Columns</a:t></a:r></a:p></a:tc>
<a:tc hMerge="1"><a:p/></a:tc>
<a:tc rowSpan="2"><a:p><a:r><a:t>Rows</a:t></a:r></a:p></a:tc>
</a:tr>
<a:tr>
<a:tc><a:p><a:r><a:t>A</a:t></a:r></a:p></a:tc>
<a:tc><a:p><a:r><a:t>B</a:t></a:r></a:p></a:tc>
<a:tc vMerge="1"><a:p/></a:tc>
</a:tr>
</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}
	result := renderPPTXNode(root.first("tbl"), TableFormatMarkdown)
	if !strings.Contains(result, `<th colspan="2">Columns</th>`) || !strings.Contains(result, `<th rowspan="2">Rows</th>`) {
		t.Fatalf("PPTX merged cells were not preserved:\n%s", result)
	}
}

func TestHTMLTablePreservesMergedRowsAndColumns(t *testing.T) {
	source := `<table><tr><th colspan="2">Columns</th><th rowspan="2">Rows</th></tr><tr><td>A</td><td>B</td></tr></table>`
	result, err := New().ConvertReader(
		context.Background(),
		strings.NewReader(source),
		StreamInfo{Name: "table.html"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Markdown, `<th colspan="2">Columns</th>`) || !strings.Contains(result.Markdown, `<th rowspan="2">Rows</th>`) {
		t.Fatalf("HTML merged cells were not preserved:\n%s", result.Markdown)
	}
}

func TestStandaloneImageUsesCustomImageHandler(t *testing.T) {
	called := false
	converter := New(WithImageHandler(func(_ context.Context, image Image) (string, error) {
		called = image.Name == "photo.png" && image.MIMEType == "image/png"
		return "https://cdn.example.com/photo.png", nil
	}))
	result, err := converter.ConvertReader(context.Background(), bytes.NewReader([]byte{0x89, 'P', 'N', 'G'}), StreamInfo{Name: "photo.png"})
	if err != nil {
		t.Fatal(err)
	}
	if !called || !strings.Contains(result.Markdown, "https://cdn.example.com/photo.png") {
		t.Fatalf("custom image handler was not used: called=%v markdown=%s", called, result.Markdown)
	}
}

func TestFileImageHandlerAvoidsNameCollisions(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "assets")
	handler := NewFileImageHandler(directory, "report_files")
	firstURL, err := handler(context.Background(), Image{Name: "../image.png", MIMEType: "image/png", Data: []byte("first")})
	if err != nil {
		t.Fatal(err)
	}
	secondURL, err := handler(context.Background(), Image{Name: "image.png", MIMEType: "image/png", Data: []byte("second")})
	if err != nil {
		t.Fatal(err)
	}
	if firstURL != "report_files/image.png" || secondURL != "report_files/image-2.png" {
		t.Fatalf("unexpected image links: %q, %q", firstURL, secondURL)
	}
	for name, expected := range map[string]string{"image.png": "first", "image-2.png": "second"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != expected {
			t.Fatalf("unexpected %s content: %q", name, data)
		}
	}
}

func TestFileImageHandlerDeduplicatesImagesByContent(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "assets")
	data := []byte("same image")
	handler := NewFileImageHandler(directory, "report_files")
	firstURL, err := handler(context.Background(), Image{Name: "page-001-image-001.png", MIMEType: "image/png", Data: data})
	if err != nil {
		t.Fatal(err)
	}
	secondURL, err := handler(context.Background(), Image{Name: "page-002-image-003.png", MIMEType: "image/png", Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if firstURL != "report_files/page-001-image-001.png" || secondURL != firstURL {
		t.Fatalf("identical images should share one link: first=%q second=%q", firstURL, secondURL)
	}

	// A newly created handler must also reuse files left by an earlier run.
	reopened := NewFileImageHandler(directory, "report_files")
	reopenedURL, err := reopened(context.Background(), Image{Name: "another-name.png", MIMEType: "image/png", Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if reopenedURL != firstURL {
		t.Fatalf("existing identical image was not reused: got %q want %q", reopenedURL, firstURL)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "page-001-image-001.png" {
		t.Fatalf("expected one stored image, got %#v", entries)
	}
}

func convertTestDOCX(t *testing.T, parts map[string][]byte, handler ImageHandler) *Result {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, data := range parts {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	options := make([]Option, 0)
	if handler != nil {
		options = append(options, WithImageHandler(handler))
	}
	result, err := New(options...).ConvertReader(context.Background(), bytes.NewReader(archive.Bytes()), StreamInfo{Name: "test.docx"})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
