package document_test

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scoming-dev/tools/docx/document"
)

func TestRunAddBreakPreservesContentOrder(t *testing.T) {
	doc := document.New()
	para := doc.AddParagraph()
	run := para.AddRun()
	run.AddText("before")
	run.AddBreak()
	run.AddText("after")
	run.AddBreak()
	run.AddText("end")

	path := filepath.Join(t.TempDir(), "breaks.docx")
	if err := doc.SaveToFile(path); err != nil {
		t.Fatalf("save docx: %v", err)
	}

	xml := readDocxPart(t, path, "word/document.xml")
	before := strings.Index(xml, "before")
	firstBreak := strings.Index(xml, "<w:br")
	after := strings.Index(xml, "after")
	secondBreak := strings.LastIndex(xml, "<w:br")
	end := strings.Index(xml, "end")

	if before < 0 || firstBreak < 0 || after < 0 || secondBreak < 0 || end < 0 {
		t.Fatalf("expected text and breaks in document.xml, got: %s", xml)
	}
	if !(before < firstBreak && firstBreak < after && after < secondBreak && secondBreak < end) {
		t.Fatalf("expected AddBreak to preserve content order, got: %s", xml)
	}
}

func readDocxPart(t *testing.T, path, partName string) string {
	t.Helper()

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open docx as zip: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != partName {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", partName, err)
		}
		content, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			t.Fatalf("read %s: %v", partName, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close %s: %v", partName, closeErr)
		}
		return string(content)
	}

	t.Fatalf("docx does not contain %s", partName)
	return ""
}
