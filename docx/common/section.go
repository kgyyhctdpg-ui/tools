package common

import (
	"github.com/mmonterroca/docxgo/v2/domain"
	"github.com/scoming-dev/tools/docx/document"
)

// PaperSize is a supported DOCX page size.
type PaperSize string

const (
	PaperSizeA1 PaperSize = "A1"
	PaperSizeA2 PaperSize = "A2"
	PaperSizeA3 PaperSize = "A3"
	PaperSizeA4 PaperSize = "A4"
)

// Orientation describes section page orientation.
type Orientation string

const (
	OrientationH Orientation = "landscape"
	OrientationV Orientation = "portrait"
)

// paperDimensions stores portrait paper dimensions in twips.
// 1 inch = 1440 twips
// A1: 594mm x 841mm = 23.39" x 33.11"
// A2: 420mm x 594mm = 16.54" x 23.39"
// A3: 297mm x 420mm = 11.69" x 16.54"
// A4: 210mm x 297mm = 8.27" x 11.69"
var paperDimensions = map[PaperSize]struct {
	width  uint64
	height uint64
}{
	PaperSizeA1: {width: 33681, height: 47682},
	PaperSizeA2: {width: 23811, height: 33681},
	PaperSizeA3: {width: 16834, height: 23811},
	PaperSizeA4: {width: 11909, height: 16834},
}

// AddSectionBreak inserts an A4 portrait section break.
func AddSectionBreak(doc *document.Document) {
	doc.AddSectionBreak(domain.OrientationPortrait, domain.PageSizeA4)
}

// AddSectionBreakWithOrientation inserts a section break with the requested orientation and paper size.
func AddSectionBreakWithOrientation(doc *document.Document, orientation Orientation, paperSize ...PaperSize) {
	size := PaperSizeA4
	if len(paperSize) > 0 {
		size = paperSize[0]
	}

	dim, ok := paperDimensions[size]
	if !ok {
		dim = paperDimensions[PaperSizeA4]
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

// IsLastSectionLandscape reports whether the last section break uses landscape orientation.
func IsLastSectionLandscape(doc *document.Document) bool {
	return doc.IsLastSectionLandscape()
}
