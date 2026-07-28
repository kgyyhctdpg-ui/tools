package document

import (
	"archive/zip"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const officeMathNamespace = `xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"`

var (
	rawXMLParagraphPattern = regexp.MustCompile(`(?s)<w:p\b[^>]*>.*?</w:p>`)
	rawXMLRunPattern       = regexp.MustCompile(`(?s)<w:r\b[^>]*>.*?</w:r>`)
)

func (d *Document) replaceRawXMLPlaceholders(path string) error {
	replacements := d.rawXMLReplacements()
	for placeholder, rawXML := range d.svgImageReplacements() {
		replacements[placeholder] = rawXML
	}
	if len(replacements) == 0 && len(d.svgImages) == 0 {
		return nil
	}

	// docxgo escapes arbitrary run text, so raw OMML is written as a unique
	// placeholder first and replaced inside the saved DOCX package afterwards.
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	readerClosed := false
	defer func() {
		if !readerClosed {
			_ = reader.Close()
		}
	}()

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".docx-raw-*.docx")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	keepTmp := false
	defer func() {
		if !keepTmp {
			_ = os.Remove(tmpName)
		}
	}()

	writer := zip.NewWriter(tmp)
	hasDocumentRels := false
	for _, file := range reader.File {
		if file.Name == "word/_rels/document.xml.rels" {
			hasDocumentRels = true
		}
		if err := d.copyDocxPartWithRawXML(writer, file, replacements); err != nil {
			_ = writer.Close()
			_ = tmp.Close()
			return err
		}
	}
	if len(d.svgImages) > 0 && !hasDocumentRels {
		if err := writeDocumentRelationshipsWithSVG(writer, d.svgImages); err != nil {
			_ = writer.Close()
			_ = tmp.Close()
			return err
		}
	}
	if err := writeSVGMediaFiles(writer, d.svgImages); err != nil {
		_ = writer.Close()
		_ = tmp.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := reader.Close(); err != nil {
		return err
	}
	readerClosed = true

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	keepTmp = true
	return nil
}

func (d *Document) copyDocxPartWithRawXML(writer *zip.Writer, file *zip.File, replacements map[string]string) error {
	header := file.FileHeader
	partWriter, err := writer.CreateHeader(&header)
	if err != nil {
		return err
	}
	if file.FileInfo().IsDir() {
		return nil
	}

	reader, err := file.Open()
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}

	if shouldReplaceRawXMLInPart(file.Name) {
		data = []byte(replaceRawXMLPlaceholdersInXML(string(data), replacements))
	}
	if len(d.svgImages) > 0 {
		switch file.Name {
		case "[Content_Types].xml":
			data = []byte(ensureSVGContentType(string(data)))
		case "word/_rels/document.xml.rels":
			data = []byte(ensureSVGRelationships(string(data), d.svgImages))
		}
	}

	_, err = partWriter.Write(data)
	return err
}

func writeDocumentRelationshipsWithSVG(writer *zip.Writer, refs []*ImageRef) error {
	header := &zip.FileHeader{Name: "word/_rels/document.xml.rels", Method: zip.Deflate}
	partWriter, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	content := ensureSVGRelationships("", refs)
	_, err = partWriter.Write([]byte(content))
	return err
}

func writeSVGMediaFiles(writer *zip.Writer, refs []*ImageRef) error {
	for _, ref := range refs {
		if ref == nil || ref.svgMediaName == "" || len(ref.image.Data) == 0 {
			continue
		}
		header := &zip.FileHeader{Name: "word/media/" + ref.svgMediaName, Method: zip.Deflate}
		partWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := partWriter.Write(ref.image.Data); err != nil {
			return err
		}
	}
	return nil
}

func shouldReplaceRawXMLInPart(name string) bool {
	return strings.HasPrefix(name, "word/") &&
		strings.HasSuffix(name, ".xml") &&
		!strings.Contains(name, "/_rels/")
}

func replaceRawXMLPlaceholdersInXML(content string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return content
	}

	replaced := false
	content = rawXMLParagraphPattern.ReplaceAllStringFunc(content, func(paragraphXML string) string {
		for placeholder, rawXML := range replacements {
			if strings.Contains(paragraphXML, placeholder) && isRawParagraphXML(rawXML) {
				replaced = true
				return rawXML
			}
		}
		return paragraphXML
	})
	content = rawXMLRunPattern.ReplaceAllStringFunc(content, func(runXML string) string {
		for placeholder, rawXML := range replacements {
			if strings.Contains(runXML, placeholder) {
				replaced = true
				return rawXML
			}
		}
		return runXML
	})
	if replaced {
		content = ensureMathNamespace(content)
	}
	return content
}

func isRawParagraphXML(rawXML string) bool {
	return strings.HasPrefix(strings.TrimSpace(rawXML), "<w:p")
}

func ensureMathNamespace(content string) string {
	if strings.Contains(content, `xmlns:m=`) {
		return content
	}
	start := strings.Index(content, "<w:")
	if start < 0 {
		return content
	}
	end := strings.Index(content[start:], ">")
	if end < 0 {
		return content
	}
	insertAt := start + end
	return content[:insertAt] + " " + officeMathNamespace + content[insertAt:]
}

func (d *Document) rawXMLReplacements() map[string]string {
	replacements := map[string]string{}
	if d == nil {
		return replacements
	}
	collectHeaderFooterRawXML(replacements, d.defaultHeader)
	collectHeaderFooterRawXML(replacements, d.defaultFooter)
	for _, block := range d.blocks {
		collectBlockRawXML(replacements, block)
	}
	return replacements
}

func collectBlockRawXML(replacements map[string]string, block *blockModel) {
	if block == nil {
		return
	}
	if block.paragraph != nil {
		collectParagraphRawXML(replacements, block.paragraph)
	}
	if block.table != nil {
		collectTableRawXML(replacements, block.table)
	}
	if block.section != nil {
		collectHeaderFooterRawXML(replacements, block.section.header)
		collectHeaderFooterRawXML(replacements, block.section.footer)
	}
}

func collectTableRawXML(replacements map[string]string, table *tableModel) {
	if table == nil {
		return
	}
	for _, row := range table.rows {
		for _, cell := range row.cells {
			for _, para := range cell.paragraphs {
				collectParagraphRawXML(replacements, para)
			}
		}
	}
}

func collectHeaderFooterRawXML(replacements map[string]string, model *headerFooterModel) {
	if model == nil {
		return
	}
	for _, para := range model.paragraphs {
		collectParagraphRawXML(replacements, para)
	}
}

func collectParagraphRawXML(replacements map[string]string, para *paragraphModel) {
	if para == nil {
		return
	}
	for _, run := range para.runs {
		for _, content := range runContents(run) {
			if content.typ == runContentRawXML && strings.TrimSpace(content.rawXML) != "" {
				replacements[rawXMLPlaceholder(content.rawXML)] = content.rawXML
			}
		}
	}
}

func rawXMLPlaceholder(rawXML string) string {
	sum := sha1.Sum([]byte(rawXML))
	return fmt.Sprintf("__DOCX_RAW_XML_%x__", sum[:])
}
