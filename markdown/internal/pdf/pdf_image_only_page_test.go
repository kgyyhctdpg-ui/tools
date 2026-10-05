package pdf

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

// 本文件覆盖「无文字层页面」的产出契约：
//
//   整页光栅产出的资产名形如 pdf-page-3.jpg；
//   页内嵌入图产出的资产名形如 pdf-page-3-image-1.jpg。
//
// 两者的区别就是本文件唯一的观测点——调用方（审查工作台）正是靠这个命名区分
// 「整页图」与「印章/签字」的，所以断言打在资产名上。

var (
	wholePageImageName = regexp.MustCompile(`^pdf-page-\d+\.\w+$`)
	embeddedImageName  = regexp.MustCompile(`^pdf-page-\d+-image-\d+\.\w+$`)
)

// pdfTestImage 是合成页上的一张嵌入图，坐标单位为点，原点在页面左下角。
type pdfTestImage struct {
	x, y, width, height float64
}

// writePDFTestPage 生成一页 PDF：可带文字层，可放任意张嵌入图。
// 用来在公开入口 convertPDF 上观察页面级行为，而不触碰任何内部结构。
func writePDFTestPage(pageWidth, pageHeight float64, text string, images []pdfTestImage) []byte {
	var content strings.Builder
	if text != "" {
		fmt.Fprintf(&content, "BT /F1 12 Tf 72 700 Td (%s) Tj ET\n", text)
	}
	for index, image := range images {
		fmt.Fprintf(&content, "q %.2f 0 0 %.2f %.2f %.2f cm /Im%d Do Q\n",
			image.width, image.height, image.x, image.y, index)
	}
	stream := content.String()

	resources := make([]string, 0, 2)
	firstImageObject := 5
	if text != "" {
		resources = append(resources, fmt.Sprintf("/Font << /F1 %d 0 R >>", firstImageObject+len(images)))
	}
	if len(images) > 0 {
		names := make([]string, 0, len(images))
		for index := range images {
			names = append(names, fmt.Sprintf("/Im%d %d 0 R", index, firstImageObject+index))
		}
		resources = append(resources, fmt.Sprintf("/XObject << %s >>", strings.Join(names, " ")))
	}

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << %s >> /Contents 4 0 R >>",
			pageWidth, pageHeight, strings.Join(resources, " ")),
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	for index := range images {
		// 每张图内容不同：转换器按内容去重，相同的图只会产出一个资产。
		rgb := strings.Repeat(string([]byte{byte(80 + index*40), 31, 90}), 64)
		objects = append(objects, fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width 8 /Height 8 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Length %d >>\nstream\n%s\nendstream",
			len(rgb), rgb))
	}
	if text != "" {
		objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	}

	var builder strings.Builder
	builder.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, body := range objects {
		offsets[index+1] = builder.Len()
		fmt.Fprintf(&builder, "%d 0 obj\n%s\nendobj\n", index+1, body)
	}
	startxref := builder.Len()
	fmt.Fprintf(&builder, "xref\n0 %d\n", len(objects)+1)
	builder.WriteString("0000000000 65535 f \n")
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&builder, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&builder, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, startxref)
	return []byte(builder.String())
}

// convertRecordingImages 转换一份合成 PDF，并记录转换器产出的所有资产名。
func convertRecordingImages(t *testing.T, data []byte, options core.PDFOptions) ([]string, *core.Result) {
	t.Helper()
	var names []string
	settings := &core.Settings{
		PDF: core.NormalizePDFOptions(options),
		ImageHandler: func(_ context.Context, image core.Image) (string, error) {
			names = append(names, image.Name)
			return image.Name, nil
		},
	}
	result, err := convertPDF(context.Background(), data, settings)
	if err != nil {
		t.Fatalf("convertPDF: %v", err)
	}
	return names, result
}

func countMatching(names []string, pattern *regexp.Regexp) []string {
	var matched []string
	for _, name := range names {
		if pattern.MatchString(name) {
			matched = append(matched, name)
		}
	}
	return matched
}

// 无文字层 + 页面上只有一小块图：那块图多半是印章/页眉标识，不是正文。
// 必须整页光栅，否则矢量正文会被整页丢掉（本次修复针对的就是这一条）。
func TestPDFVectorPageWithOnlyASmallImageIsRasterized(t *testing.T) {
	data := writePDFTestPage(400, 300, "", []pdfTestImage{{x: 350, y: 20, width: 36, height: 36}})
	names, _ := convertRecordingImages(t, data, core.PDFOptions{})

	if got := countMatching(names, wholePageImageName); len(got) != 1 {
		t.Fatalf("whole-page rasters = %v, want exactly 1（资产名: %v）", got, names)
	}
	if got := countMatching(names, embeddedImageName); len(got) != 0 {
		t.Fatalf("embedded images = %v, want none（印章不该被单独抠出来）", got)
	}
}

// 多张小图拼成的页面同理：按面积之和判断，而不是按张数。
func TestPDFVectorPageWithSeveralSmallImagesIsRasterized(t *testing.T) {
	data := writePDFTestPage(400, 300, "", []pdfTestImage{
		{x: 300, y: 20, width: 36, height: 36},
		{x: 250, y: 20, width: 36, height: 36},
		{x: 200, y: 20, width: 36, height: 36},
	})
	names, _ := convertRecordingImages(t, data, core.PDFOptions{})

	if got := countMatching(names, wholePageImageName); len(got) != 1 {
		t.Fatalf("whole-page rasters = %v, want exactly 1（资产名: %v）", got, names)
	}
	if got := countMatching(names, embeddedImageName); len(got) != 0 {
		t.Fatalf("embedded images = %v, want none", got)
	}
}

// 整页扫描件：那张图就是页面正文，必须保留原图，不能被整页光栅替代
// （这是修复前就正确的行为，本用例防止修复把它改坏）。
func TestPDFPageCoveredByItsImageKeepsThePicture(t *testing.T) {
	data := writePDFTestPage(400, 300, "", []pdfTestImage{{x: 0, y: 0, width: 400, height: 300}})
	names, _ := convertRecordingImages(t, data, core.PDFOptions{})

	if got := countMatching(names, embeddedImageName); len(got) != 1 {
		t.Fatalf("embedded images = %v, want exactly 1（整页图应保留原图）", got)
	}
	if got := countMatching(names, wholePageImageName); len(got) != 0 {
		t.Fatalf("whole-page rasters = %v, want none（整页图不必再光栅一遍）", got)
	}
}

// 有文字层的页面完全不受影响：图照旧抠出，不做整页光栅。
func TestPDFTextPageWithASmallImageKeepsThePicture(t *testing.T) {
	data := writePDFTestPage(612, 792, "Page heading", []pdfTestImage{{x: 500, y: 40, width: 36, height: 36}})
	names, result := convertRecordingImages(t, data, core.PDFOptions{})

	if got := countMatching(names, wholePageImageName); len(got) != 0 {
		t.Fatalf("whole-page rasters = %v, want none（有文字层的页面不整页光栅）", got)
	}
	if got := countMatching(names, embeddedImageName); len(got) != 1 {
		t.Fatalf("embedded images = %v, want exactly 1", got)
	}
	if !strings.Contains(result.Markdown, "Page heading") {
		t.Fatalf("文字未被提取:\n%s", result.Markdown)
	}
}

// 无文字层且页面上没有图：修复前就应整页光栅，本用例防止回归。
func TestPDFVectorPageWithoutImagesIsRasterized(t *testing.T) {
	data := writePDFTestPage(400, 300, "", nil)
	names, _ := convertRecordingImages(t, data, core.PDFOptions{})

	if got := countMatching(names, wholePageImageName); len(got) != 1 {
		t.Fatalf("whole-page rasters = %v, want exactly 1（资产名: %v）", got, names)
	}
}

// 显式关掉整页光栅兜底时，不能因为新判据而渲染：退回保留原图。
func TestPDFSmallImagePageHonoursDisabledRenderFallback(t *testing.T) {
	data := writePDFTestPage(400, 300, "", []pdfTestImage{{x: 350, y: 20, width: 36, height: 36}})
	names, _ := convertRecordingImages(t, data, core.PDFOptions{DisablePageRenderFallback: true})

	if got := countMatching(names, wholePageImageName); len(got) != 0 {
		t.Fatalf("whole-page rasters = %v, want none（已显式禁用整页渲染）", got)
	}
	if got := countMatching(names, embeddedImageName); len(got) != 1 {
		t.Fatalf("embedded images = %v, want exactly 1", got)
	}
}
