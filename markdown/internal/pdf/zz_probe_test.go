package pdf

import (
	"fmt"
	"os"
	"testing"

	"github.com/88250/lute"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

// TestDebugProbe is a scratch diagnostic: set PDF_DEBUG_DUMP to a PDF path (and
// PDF_DEBUG_PAGE to a page index) to inspect what the engine reports, from the
// raw runs parsed out of MuPDF's HTML down to the finished blocks.
func TestDebugProbe(t *testing.T) {
	path := os.Getenv("PDF_DEBUG_DUMP")
	if path == "" {
		t.Skip("set PDF_DEBUG_DUMP=/path/to.pdf")
	}
	page := 0
	if value := os.Getenv("PDF_DEBUG_PAGE"); value != "" {
		fmt.Sscanf(value, "%d", &page)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document, err := openPDFDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	width, height, err := document.PageSize(page)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := document.pageHTML(page)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("=== page %d  %.1fx%.1f  runs=%d images=%d", page+1, width, height, len(parsed.rects), len(parsed.images))
	for index, rect := range parsed.rects {
		t.Logf("  [%3d] top=%9.2f bottom=%9.2f left=%9.2f right=%9.2f size=%7.2f bold=%v italic=%v %q",
			index, rect.Top, rect.Bottom, rect.Left, rect.Right, rect.FontSize, rect.bold(), rect.italic(), truncate(rect.Text, 46))
	}
	for index, image := range parsed.images {
		t.Logf("  IMG[%3d] top=%9.2f bottom=%9.2f left=%9.2f right=%9.2f px=%dx%d %s bytes=%d",
			index, image.TopDown, image.Bottom, image.Left, image.Right,
			image.PixelWidth, image.PixelHeight, image.MIMEType, len(image.Data))
	}

	lines, err := document.PageTextLines(page)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("=== production lines=%d", len(lines))
	for index, line := range lines {
		if index >= 40 {
			break
		}
		t.Logf("  L[%3d] top=%9.2f left=%9.2f size=%6.2f %q", index, line.Top, line.Left, line.FontSize, line.Text())
	}
	blocks := buildPDFLineBlocks(lines, nil, core.NormalizePDFOptions(core.PDFOptions{}), core.TableFormatMarkdown)
	t.Logf("=== blocks=%d", len(blocks))
	engine := lute.New()
	for index, block := range blocks {
		if index >= 25 {
			break
		}
		text, err := renderPDFBlock(engine, block)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("  BLOCK[%d] lines=%d raw=%v %q", index, block.lines, block.raw, truncate(text, 70))
	}
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}
