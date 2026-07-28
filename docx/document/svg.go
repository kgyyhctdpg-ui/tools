package document

import (
	"fmt"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
	"github.com/scoming-dev/tools/docx/media"
)

const (
	svgContentType       = "image/svg+xml"
	imageRelationship    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	packageRelsNS        = "http://schemas.openxmlformats.org/package/2006/relationships"
	contentTypesCloseTag = "</Types>"
	relsCloseTag         = "</Relationships>"
)

func (d *Document) newImageRef(img media.Image) *ImageRef {
	ref := &ImageRef{image: img}
	if img.Format != domain.ImageFormatSVG {
		return ref
	}

	d.nextSVGID++
	ref.svgID = d.nextSVGID
	ref.svgPlaceholder = fmt.Sprintf("__DOCX_SVG_IMAGE_%d__", ref.svgID)
	ref.svgRelID = fmt.Sprintf("rIdDocxSVGImage%d", ref.svgID)
	ref.svgMediaName = fmt.Sprintf("image%d.svg", ref.svgID)
	d.svgImages = append(d.svgImages, ref)
	return ref
}

func (r *ImageRef) isSVG() bool {
	return r != nil && r.image.Format == domain.ImageFormatSVG
}

func renderSVGImageRun(out domain.Paragraph, ref *ImageRef) error {
	run, err := out.AddRun()
	if err != nil {
		return err
	}
	return run.AddText(ref.svgPlaceholder)
}

func (d *Document) svgImageReplacements() map[string]string {
	replacements := map[string]string{}
	for _, ref := range d.svgImages {
		if ref == nil || ref.svgPlaceholder == "" {
			continue
		}
		replacements[ref.svgPlaceholder] = svgDrawingXML(ref)
	}
	return replacements
}

func svgDrawingXML(ref *ImageRef) string {
	size := imageSize(ref.width, ref.height)
	docPrID := ref.svgID
	if docPrID < 1 {
		docPrID = 1
	}
	name := ref.svgMediaName
	if name == "" {
		name = fmt.Sprintf("image%d.svg", docPrID)
	}

	return fmt.Sprintf(`<w:r xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:docPr id="%d" name="%s"/><wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="%d" name="%s"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`,
		size.WidthEMU,
		size.HeightEMU,
		docPrID,
		xmlEscapeAttr(name),
		docPrID,
		xmlEscapeAttr(name),
		xmlEscapeAttr(ref.svgRelID),
		size.WidthEMU,
		size.HeightEMU,
	)
}

func ensureSVGContentType(content string) string {
	if strings.Contains(content, `Extension="svg"`) || strings.Contains(content, svgContentType) {
		return content
	}
	return insertBeforeClosingTag(content, contentTypesCloseTag, `<Default Extension="svg" ContentType="`+svgContentType+`"/>`)
}

func ensureSVGRelationships(content string, refs []*ImageRef) string {
	if strings.TrimSpace(content) == "" {
		content = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="` + packageRelsNS + `"></Relationships>`
	}
	for _, ref := range refs {
		if ref == nil || ref.svgRelID == "" || ref.svgMediaName == "" {
			continue
		}
		if strings.Contains(content, `Id="`+ref.svgRelID+`"`) {
			continue
		}
		relationship := fmt.Sprintf(`<Relationship Id="%s" Type="%s" Target="media/%s"/>`,
			xmlEscapeAttr(ref.svgRelID),
			imageRelationship,
			xmlEscapeAttr(ref.svgMediaName),
		)
		content = insertBeforeClosingTag(content, relsCloseTag, relationship)
	}
	return content
}

func insertBeforeClosingTag(content, closeTag, insert string) string {
	index := strings.LastIndex(content, closeTag)
	if index < 0 {
		return content + insert
	}
	return content[:index] + insert + content[index:]
}
