// Package document provides a small document model used by the Markdown renderer.
package document

import (
	"math"
	"strings"

	docxgo "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"
	compatcolor "github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/media"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

// Field identifies a supported dynamic DOCX field.
type Field int

const (
	// FieldCurrentPage renders the current page number.
	FieldCurrentPage Field = iota
	// FieldNumberOfPages renders the total page count.
	FieldNumberOfPages
)

// Document stores the paragraphs, tables, sections, numbering, and media before saving.
type Document struct {
	Numbering *Numbering

	blocks        []*blockModel
	paragraphs    []*paragraphModel
	defaultHeader *headerFooterModel
	defaultFooter *headerFooterModel
	lastLandscape bool
	svgImages     []*ImageRef
	nextSVGID     int
}

type blockModel struct {
	paragraph *paragraphModel
	table     *tableModel
	section   *sectionModel
}

type paragraphModel struct {
	runs         []*runModel
	props        paragraphPropsModel
	style        string
	numbering    *Definition
	bookmarkID   string
	bookmarkName string
}

type paragraphPropsModel struct {
	alignment       wml.ST_Jc
	hasAlignment    bool
	firstLineIndent int
	startIndent     int
	spacing         spacingModel
	x               *wml.CT_PPr
}

type spacingModel struct {
	before    int
	after     int
	line      int
	lineRule  wml.ST_LineSpacingRule
	hasBefore bool
	hasAfter  bool
	hasLine   bool
}

type runModel struct {
	text      string
	props     runPropsModel
	contents  []runContentModel
	image     *ImageRef
	hyperlink *hyperlinkModel
}

type runContentModel struct {
	typ       runContentType
	text      string
	breakType domain.BreakType
	field     Field
	rawXML    string
}

type runContentType int

const (
	runContentText runContentType = iota
	runContentBreak
	runContentField
	runContentRawXML
)

type runPropsModel struct {
	style     string
	font      domain.Font
	fontX     FontX
	sizePt    float64
	hasSize   bool
	bold      bool
	italic    bool
	strike    bool
	color     compatcolor.Color
	hasColor  bool
	charSpace float64
}

type tableModel struct {
	rows  []*rowModel
	props tablePropsModel
}

type tablePropsModel struct {
	widthPercent float64
	alignment    wml.ST_Jc
	hasAlignment bool
	borders      borderModel
}

type rowModel struct {
	cells []*cellModel
}

type cellModel struct {
	paragraphs []*paragraphModel
	props      cellPropsModel
}

type cellPropsModel struct {
	verticalAlignment wml.ST_VerticalJc
	hasVAlign         bool
	verticalMerge     wml.ST_Merge
	columnSpan        int
	shading           compatcolor.Color
	hasShading        bool
	borders           borderModel
	x                 *wml.CT_TcPr
}

type borderModel struct {
	enabled bool
	color   compatcolor.Color
	width   float64
}

type headerFooterModel struct {
	paragraphs []*paragraphModel
}

type sectionModel struct {
	orientation domain.Orientation
	pageSize    domain.PageSize
	header      *headerFooterModel
	footer      *headerFooterModel
}

// New creates an empty document model.
func New() *Document {
	numbering := &Numbering{}
	return &Document{
		Numbering: numbering,
	}
}

// AddParagraph appends a paragraph to the document body.
func (d *Document) AddParagraph() Paragraph {
	model := newParagraphModel()
	d.paragraphs = append(d.paragraphs, model)
	d.blocks = append(d.blocks, &blockModel{paragraph: model})
	return Paragraph{model: model}
}

// AddTable appends a table to the document body.
func (d *Document) AddTable() Table {
	model := &tableModel{}
	d.blocks = append(d.blocks, &blockModel{table: model})
	return Table{model: model}
}

// AddHeader creates the default document header.
func (d *Document) AddHeader() Header {
	model := &headerFooterModel{}
	d.defaultHeader = model
	return Header{model: model}
}

// AddFooter creates the default document footer.
func (d *Document) AddFooter() Footer {
	model := &headerFooterModel{}
	d.defaultFooter = model
	return Footer{model: model}
}

// AddImage registers an image so it can be inserted by a run.
func (d *Document) AddImage(img media.Image) (*ImageRef, error) {
	return d.newImageRef(img), nil
}

// SaveToFile renders the document model to a DOCX file.
func (d *Document) SaveToFile(path string) error {
	out := docxgo.NewDocument()
	if err := d.configureNumbering(out); err != nil {
		return err
	}
	if err := d.renderHeaderFooter(out); err != nil {
		return err
	}

	for _, block := range d.blocks {
		switch {
		case block.paragraph != nil:
			if err := renderParagraph(out.AddParagraph, block.paragraph); err != nil {
				return err
			}
		case block.table != nil:
			if err := renderTable(out, block.table); err != nil {
				return err
			}
		case block.section != nil:
			section, err := out.AddSectionWithBreak(domain.SectionBreakTypeNextPage)
			if err != nil {
				return err
			}
			_ = section.SetOrientation(block.section.orientation)
			_ = section.SetPageSize(block.section.pageSize)
			_ = renderSectionHeaderFooter(section, d.defaultHeader, d.defaultFooter)
		}
	}

	if len(d.blocks) == 0 {
		para, err := out.AddParagraph()
		if err != nil {
			return err
		}
		run, err := para.AddRun()
		if err != nil {
			return err
		}
		_ = run.SetText("")
	}

	if err := out.SaveAs(path); err != nil {
		return err
	}
	return d.replaceRawXMLPlaceholders(path)
}

func (d *Document) configureNumbering(out domain.Document) error {
	if d == nil || d.Numbering == nil || len(d.Numbering.definitions) == 0 {
		return nil
	}

	configurator, ok := out.(interface {
		SetNumberingPart([]byte, string)
	})
	if !ok {
		return nil
	}

	configurator.SetNumberingPart(d.Numbering.XML(), "numbering.xml")
	return nil
}

func (d *Document) renderHeaderFooter(out domain.Document) error {
	section, err := out.DefaultSection()
	if err != nil {
		return err
	}
	return renderSectionHeaderFooter(section, d.defaultHeader, d.defaultFooter)
}

func renderSectionHeaderFooter(section domain.Section, header, footer *headerFooterModel) error {
	if header != nil {
		outHeader, err := section.Header(domain.HeaderDefault)
		if err != nil {
			return err
		}
		for _, para := range header.paragraphs {
			if err := renderParagraph(outHeader.AddParagraph, para); err != nil {
				return err
			}
		}
	}
	if footer != nil {
		outFooter, err := section.Footer(domain.FooterDefault)
		if err != nil {
			return err
		}
		for _, para := range footer.paragraphs {
			if err := renderParagraph(outFooter.AddParagraph, para); err != nil {
				return err
			}
		}
	}
	return nil
}

// BodySection returns a handle for configuring the document body section.
func (d *Document) BodySection() Section {
	return Section{doc: d, model: &sectionModel{}}
}

// Paragraphs returns the body paragraphs in insertion order.
func (d *Document) Paragraphs() []Paragraph {
	ret := make([]Paragraph, 0, len(d.paragraphs))
	for _, para := range d.paragraphs {
		ret = append(ret, Paragraph{model: para})
	}
	return ret
}

// RemoveParagraph removes a paragraph from the document body.
func (d *Document) RemoveParagraph(para Paragraph) {
	if para.model == nil {
		return
	}
	for i, p := range d.paragraphs {
		if p == para.model {
			d.paragraphs = append(d.paragraphs[:i], d.paragraphs[i+1:]...)
			break
		}
	}
	for i, block := range d.blocks {
		if block.paragraph == para.model {
			d.blocks = append(d.blocks[:i], d.blocks[i+1:]...)
			return
		}
	}
}

// AddSectionBreak appends a section break with the requested orientation and page size.
func (d *Document) AddSectionBreak(orientation domain.Orientation, pageSize domain.PageSize) Section {
	model := &sectionModel{orientation: orientation, pageSize: pageSize, header: d.defaultHeader, footer: d.defaultFooter}
	d.lastLandscape = orientation == domain.OrientationLandscape
	d.blocks = append(d.blocks, &blockModel{section: model})
	return Section{doc: d, model: model}
}

// IsLastSectionLandscape reports whether the last added section break was landscape.
func (d *Document) IsLastSectionLandscape() bool {
	return d.lastLandscape
}

func newParagraphModel() *paragraphModel {
	return &paragraphModel{
		props: paragraphPropsModel{
			x: &wml.CT_PPr{},
		},
	}
}

// Paragraph represents one body, header, footer, or table-cell paragraph.
type Paragraph struct {
	model *paragraphModel
}

// AddRun appends a text run to the paragraph.
func (p Paragraph) AddRun() Run {
	if p.model == nil {
		return Run{}
	}
	model := &runModel{}
	p.model.runs = append(p.model.runs, model)
	return Run{model: model}
}

// Properties returns mutable paragraph properties.
func (p Paragraph) Properties() ParagraphProperties {
	return ParagraphProperties{model: p.model}
}

// SetStyle applies a built-in or custom paragraph style ID.
func (p Paragraph) SetStyle(style string) {
	if p.model == nil {
		return
	}
	p.model.style = style
}

// AddHyperLink appends a hyperlink container to the paragraph.
func (p Paragraph) AddHyperLink() HyperLink {
	return HyperLink{para: p.model, model: &hyperlinkModel{}}
}

// AddBookmark attaches a bookmark to the paragraph.
func (p Paragraph) AddBookmark(id string) {
	if p.model == nil {
		return
	}
	p.model.bookmarkID = id
	p.model.bookmarkName = id
}

// SetNumberingDefinition applies a DOCX numbering definition to the paragraph.
func (p Paragraph) SetNumberingDefinition(def Definition) {
	if p.model == nil {
		return
	}
	p.model.numbering = &def
}

// ParagraphProperties provides paragraph-level styling operations.
type ParagraphProperties struct {
	model *paragraphModel
}

// SetAlignment sets paragraph alignment.
func (p ParagraphProperties) SetAlignment(align wml.ST_Jc) {
	if p.model == nil {
		return
	}
	p.model.props.alignment = align
	p.model.props.hasAlignment = true
}

// SetFirstLineIndent sets the first-line indent.
func (p ParagraphProperties) SetFirstLineIndent(indent float64) {
	if p.model == nil {
		return
	}
	p.model.props.firstLineIndent = pointsToTwips(indent)
}

// SetStartIndent sets the paragraph start indent.
func (p ParagraphProperties) SetStartIndent(indent float64) {
	if p.model == nil {
		return
	}
	p.model.props.startIndent = pointsToTwips(indent)
}

// Spacing returns mutable paragraph spacing properties.
func (p ParagraphProperties) Spacing() Spacing {
	return Spacing{model: p.model}
}

// X exposes low-level paragraph properties for compatibility gaps.
func (p ParagraphProperties) X() *wml.CT_PPr {
	if p.model == nil {
		return &wml.CT_PPr{}
	}
	if p.model.props.x == nil {
		p.model.props.x = &wml.CT_PPr{}
	}
	return p.model.props.x
}

// AddSection attaches a section property object to the paragraph.
func (p ParagraphProperties) AddSection(_ wml.ST_SectionMark) Section {
	sectPr := wml.NewCT_SectPr()
	sectPr.EG_HdrFtrReferences = []*wml.EG_HdrFtrReferences{}
	p.X().SectPr = sectPr
	return Section{model: &sectionModel{}}
}

// Spacing provides paragraph spacing operations.
type Spacing struct {
	model *paragraphModel
}

// SetBefore sets paragraph spacing before.
func (s Spacing) SetBefore(value float64) {
	if s.model == nil {
		return
	}
	s.model.props.spacing.before = pointsToTwips(value)
	s.model.props.spacing.hasBefore = true
}

// SetAfter sets paragraph spacing after.
func (s Spacing) SetAfter(value float64) {
	if s.model == nil {
		return
	}
	s.model.props.spacing.after = pointsToTwips(value)
	s.model.props.spacing.hasAfter = true
}

// SetLineSpacing sets paragraph line spacing.
func (s Spacing) SetLineSpacing(value float64, rule wml.ST_LineSpacingRule) {
	if s.model == nil {
		return
	}
	s.model.props.spacing.line = pointsToTwips(value)
	s.model.props.spacing.lineRule = rule
	s.model.props.spacing.hasLine = true
}

// Run represents a sequence of text, field, break, image, or raw XML content with shared styling.
type Run struct {
	model *runModel
}

// AddText appends text to the run.
func (r Run) AddText(text string) {
	if r.model == nil {
		return
	}
	r.model.text += text
	if text == "" {
		return
	}
	last := len(r.model.contents) - 1
	if last >= 0 && r.model.contents[last].typ == runContentText {
		r.model.contents[last].text += text
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentText, text: text})
}

// AddBreak appends a line break to the run.
func (r Run) AddBreak() {
	if r.model == nil {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentBreak, breakType: domain.BreakTypeLine})
}

// AddPageBreak appends a page break to the run.
func (r Run) AddPageBreak() {
	if r.model == nil {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentBreak, breakType: domain.BreakTypePage})
}

// AddTab appends a tab character to the run.
func (r Run) AddTab() {
	r.AddText("\t")
}

// AddField appends a dynamic DOCX field to the run.
func (r Run) AddField(field Field) {
	if r.model == nil {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentField, field: field})
}

// AddRawXML appends trusted WordprocessingML or OMML that should be written as XML, not escaped text.
func (r Run) AddRawXML(rawXML string) {
	if r.model == nil || strings.TrimSpace(rawXML) == "" {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentRawXML, rawXML: rawXML})
}

// AddDrawingInline appends an inline image drawing to the run.
func (r Run) AddDrawingInline(ref *ImageRef) (*DrawingInline, error) {
	if r.model == nil {
		return &DrawingInline{}, nil
	}
	r.model.image = ref
	return &DrawingInline{ref: ref}, nil
}

// Properties returns mutable run properties.
func (r Run) Properties() RunProperties {
	if r.model == nil {
		return RunProperties{}
	}
	return RunProperties{props: &r.model.props}
}

// RunProperties provides run-level styling operations.
type RunProperties struct {
	props *runPropsModel
}

// SetStyle sets the run style name.
func (p RunProperties) SetStyle(style string) {
	if p.props != nil {
		p.props.style = style
	}
}

// SetBold toggles bold text.
func (p RunProperties) SetBold(bold bool) {
	if p.props != nil {
		p.props.bold = bold
	}
}

// SetItalic toggles italic text.
func (p RunProperties) SetItalic(italic bool) {
	if p.props != nil {
		p.props.italic = italic
	}
}

// SetStrikeThrough toggles strikethrough text.
func (p RunProperties) SetStrikeThrough(strike bool) {
	if p.props != nil {
		p.props.strike = strike
	}
}

// SetFontFamily sets the run font family.
func (p RunProperties) SetFontFamily(name string) {
	if p.props == nil {
		return
	}
	p.props.font.Name = name
	p.props.font.EastAsia = name
}

// SetSize sets the run font size in points.
func (p RunProperties) SetSize(size float64) {
	if p.props == nil {
		return
	}
	p.props.sizePt = float64(size)
	p.props.hasSize = true
}

// SetCharacterSpacing sets run character spacing.
func (p RunProperties) SetCharacterSpacing(value float64) {
	if p.props != nil {
		p.props.charSpace = value
	}
}

// SetColor sets the run text color.
func (p RunProperties) SetColor(color compatcolor.Color) {
	if p.props == nil {
		return
	}
	p.props.color = color
	p.props.hasColor = true
}

// Fonts returns low-level run font attributes.
func (p RunProperties) Fonts() Fonts {
	return Fonts{props: p.props}
}

// Fonts exposes low-level font attributes used by DOCX compatibility code.
type Fonts struct {
	props *runPropsModel
}

// X returns mutable low-level font XML attributes.
func (f Fonts) X() *FontX {
	if f.props == nil {
		return &FontX{}
	}
	return &f.props.fontX
}

// FontX contains low-level WordprocessingML font attributes.
type FontX struct {
	AsciiAttr    *string
	HAnsiAttr    *string
	EastAsiaAttr *string
	CsAttr       *string
}

// HyperLink represents a paragraph hyperlink.
type HyperLink struct {
	para  *paragraphModel
	model *hyperlinkModel
}

type hyperlinkModel struct {
	target string
	x      HyperLinkXML
}

// HyperLinkXML exposes low-level hyperlink XML attributes.
type HyperLinkXML struct {
	AnchorAttr *string
}

// SetTarget sets the external hyperlink target.
func (h HyperLink) SetTarget(target string) {
	if h.model != nil {
		h.model.target = target
	}
}

// AddRun appends a run inside the hyperlink.
func (h HyperLink) AddRun() Run {
	if h.para == nil {
		return Run{}
	}
	model := &runModel{hyperlink: h.model}
	h.para.runs = append(h.para.runs, model)
	return Run{model: model}
}

// X exposes low-level hyperlink properties for internal anchors.
func (h HyperLink) X() *HyperLinkXML {
	if h.model == nil {
		return &HyperLinkXML{}
	}
	return &h.model.x
}

// Table represents a DOCX table.
type Table struct {
	model *tableModel
}

// AddRow appends a row to the table.
func (t Table) AddRow() Row {
	if t.model == nil {
		return Row{}
	}
	model := &rowModel{}
	t.model.rows = append(t.model.rows, model)
	return Row{model: model}
}

// Properties returns mutable table properties.
func (t Table) Properties() TableProperties {
	return TableProperties{model: t.model}
}

// Row represents a DOCX table row.
type Row struct {
	model *rowModel
}

// AddCell appends a cell to the row.
func (r Row) AddCell() Cell {
	if r.model == nil {
		return Cell{}
	}
	model := &cellModel{
		props: cellPropsModel{x: &wml.CT_TcPr{}},
	}
	r.model.cells = append(r.model.cells, model)
	return Cell{model: model}
}

// Cell represents a DOCX table cell.
type Cell struct {
	model *cellModel
}

// AddParagraph appends a paragraph to the cell.
func (c Cell) AddParagraph() Paragraph {
	if c.model == nil {
		return Paragraph{}
	}
	model := newParagraphModel()
	c.model.paragraphs = append(c.model.paragraphs, model)
	return Paragraph{model: model}
}

// Properties returns mutable cell properties.
func (c Cell) Properties() CellProperties {
	return CellProperties{model: c.model}
}

// TableProperties provides table-level styling operations.
type TableProperties struct {
	model *tableModel
}

// SetWidthPercent sets table width as a percentage.
func (p TableProperties) SetWidthPercent(percent float64) {
	if p.model != nil {
		p.model.props.widthPercent = percent
	}
}

// SetAlignment sets table alignment.
func (p TableProperties) SetAlignment(align wml.ST_Jc) {
	if p.model == nil {
		return
	}
	p.model.props.alignment = align
	p.model.props.hasAlignment = true
}

// Borders returns mutable table borders.
func (p TableProperties) Borders() Borders {
	if p.model == nil {
		return Borders{}
	}
	return Borders{border: &p.model.props.borders}
}

// CellProperties provides cell-level styling operations.
type CellProperties struct {
	model *cellModel
}

// SetVerticalAlignment sets vertical cell alignment.
func (p CellProperties) SetVerticalAlignment(align wml.ST_VerticalJc) {
	if p.model == nil {
		return
	}
	p.model.props.verticalAlignment = align
	p.model.props.hasVAlign = true
}

// SetVerticalMerge sets the vertical merge mode.
func (p CellProperties) SetVerticalMerge(merge wml.ST_Merge) {
	if p.model != nil {
		p.model.props.verticalMerge = merge
	}
}

// SetColumnSpan sets the horizontal column span.
func (p CellProperties) SetColumnSpan(span int) {
	if p.model != nil && span > 0 {
		p.model.props.columnSpan = span
	}
}

// SetShading sets the cell background color.
func (p CellProperties) SetShading(_ wml.ST_Shd, fill compatcolor.Color, _ compatcolor.Color) {
	if p.model == nil {
		return
	}
	p.model.props.shading = fill
	p.model.props.hasShading = true
}

// Borders returns mutable cell borders.
func (p CellProperties) Borders() Borders {
	if p.model == nil {
		return Borders{}
	}
	return Borders{border: &p.model.props.borders}
}

// X exposes low-level cell properties for compatibility gaps.
func (p CellProperties) X() *wml.CT_TcPr {
	if p.model == nil {
		return &wml.CT_TcPr{}
	}
	if p.model.props.x == nil {
		p.model.props.x = &wml.CT_TcPr{}
	}
	return p.model.props.x
}

// Borders configures table or cell borders.
type Borders struct {
	border *borderModel
}

// SetAll applies the same border style to all sides.
func (b Borders) SetAll(_ wml.ST_Border, color compatcolor.Color, width float64) {
	if b.border == nil {
		return
	}
	b.border.enabled = true
	b.border.color = color
	b.border.width = width
}

// ImageRef references a loaded image in the document.
type ImageRef struct {
	image          media.Image
	width          float64
	height         float64
	svgID          int
	svgPlaceholder string
	svgRelID       string
	svgMediaName   string
}

// DrawingInline represents an inline image drawing.
type DrawingInline struct {
	ref *ImageRef
}

// SetSize sets the inline drawing dimensions.
func (d *DrawingInline) SetSize(width, height float64) {
	if d == nil || d.ref == nil {
		return
	}
	d.ref.width = width
	d.ref.height = height
}

// Header represents the default document header.
type Header struct {
	model *headerFooterModel
}

// AddParagraph appends a paragraph to the header.
func (h Header) AddParagraph() Paragraph {
	if h.model == nil {
		return Paragraph{}
	}
	model := newParagraphModel()
	h.model.paragraphs = append(h.model.paragraphs, model)
	return Paragraph{model: model}
}

// Footer represents the default document footer.
type Footer struct {
	model *headerFooterModel
}

// AddParagraph appends a paragraph to the footer.
func (f Footer) AddParagraph() Paragraph {
	if f.model == nil {
		return Paragraph{}
	}
	model := newParagraphModel()
	f.model.paragraphs = append(f.model.paragraphs, model)
	return Paragraph{model: model}
}

// Section represents section-level document properties.
type Section struct {
	doc   *Document
	model *sectionModel
}

// SetHeader applies a header to the section.
func (s Section) SetHeader(header Header, _ wml.ST_HdrFtr) {
	if s.doc != nil {
		s.doc.defaultHeader = header.model
	}
	if s.model != nil {
		s.model.header = header.model
	}
}

// SetFooter applies a footer to the section.
func (s Section) SetFooter(footer Footer, _ wml.ST_HdrFtr) {
	if s.doc != nil {
		s.doc.defaultFooter = footer.model
	}
	if s.model != nil {
		s.model.footer = footer.model
	}
}

func renderParagraph(add func() (domain.Paragraph, error), model *paragraphModel) error {
	out, err := add()
	if err != nil {
		return err
	}

	if model.style != "" {
		_ = out.SetStyle(model.style)
	}
	if model.props.hasAlignment {
		_ = out.SetAlignment(toDomainAlignment(model.props.alignment))
	}
	indent := domain.Indentation{}
	if model.props.firstLineIndent > 0 {
		indent.FirstLine = model.props.firstLineIndent
	}
	if model.props.startIndent > 0 {
		indent.Left = model.props.startIndent
	}
	if indent != (domain.Indentation{}) {
		_ = out.SetIndent(indent)
	}
	if model.props.spacing.hasBefore {
		_ = out.SetSpacingBefore(model.props.spacing.before)
	}
	if model.props.spacing.hasAfter {
		_ = out.SetSpacingAfter(model.props.spacing.after)
	}
	if model.props.spacing.hasLine {
		_ = out.SetLineSpacing(domain.LineSpacing{Rule: domain.LineSpacingAuto, Value: model.props.spacing.line})
	}
	if model.bookmarkID != "" {
		if setter, ok := out.(interface{ SetBookmark(string, string) }); ok {
			setter.SetBookmark(model.bookmarkID, model.bookmarkName)
		}
	}
	if model.numbering != nil && model.numbering.NumberID() > 0 {
		_ = out.SetNumbering(domain.NumberingReference{
			ID:    model.numbering.NumberID(),
			Level: model.numbering.LevelIndex(),
		})
	}

	for _, run := range model.runs {
		if run.image != nil {
			if err := renderImageRun(out, run.image); err != nil {
				return err
			}
			continue
		}

		if run.hyperlink != nil && run.hyperlink.target != "" {
			outRun, err := out.AddHyperlink(run.hyperlink.target, run.text)
			if err != nil {
				return err
			}
			applyRunProperties(outRun, run.props)
			continue
		}

		for _, content := range runContents(run) {
			outRun, err := out.AddRun()
			if err != nil {
				return err
			}
			applyRunProperties(outRun, run.props)
			switch content.typ {
			case runContentText:
				_ = outRun.AddText(content.text)
			case runContentBreak:
				_ = outRun.AddBreak(content.breakType)
			case runContentField:
				switch content.field {
				case FieldCurrentPage:
					_ = outRun.AddField(docxgo.NewPageNumberField())
				case FieldNumberOfPages:
					_ = outRun.AddField(docxgo.NewPageCountField())
				}
			case runContentRawXML:
				_ = outRun.AddText(rawXMLPlaceholder(content.rawXML))
			}
		}
	}
	return nil
}

func runContents(run *runModel) []runContentModel {
	if run == nil {
		return nil
	}
	if len(run.contents) > 0 {
		return run.contents
	}
	if run.text != "" {
		return []runContentModel{{typ: runContentText, text: run.text}}
	}
	return nil
}

func renderImageRun(out domain.Paragraph, ref *ImageRef) error {
	if ref == nil {
		return nil
	}
	if ref.isSVG() {
		return renderSVGImageRun(out, ref)
	}
	size := imageSize(ref.width, ref.height)
	_, err := out.AddImageFromBytesWithSize(ref.image.Data, ref.image.Format, size)
	return err
}

func renderTable(out domain.Document, model *tableModel) error {
	if len(model.rows) == 0 {
		return nil
	}
	cols := tableColumnCount(model)
	if cols < 1 {
		cols = 1
	}
	table, err := out.AddTable(len(model.rows), cols)
	if err != nil {
		return err
	}
	if model.props.widthPercent > 0 {
		_ = table.SetWidth(domain.TableWidth{Type: domain.WidthPct, Value: int(model.props.widthPercent * 50)})
	}
	if model.props.hasAlignment {
		_ = table.SetAlignment(toDomainAlignment(model.props.alignment))
	}

	for rowIdx, rowModel := range model.rows {
		row, err := table.Row(rowIdx)
		if err != nil {
			return err
		}
		colIdx := 0
		for _, cellModel := range rowModel.cells {
			if colIdx >= cols {
				break
			}
			cell, err := row.Cell(colIdx)
			if err != nil {
				return err
			}
			applyCellProperties(cell, cellModel.props)
			for _, para := range cellModel.paragraphs {
				if err := renderParagraph(cell.AddParagraph, para); err != nil {
					return err
				}
			}
			colIdx += cellSpan(cellModel.props)
		}
	}
	return nil
}

func applyCellProperties(cell domain.TableCell, props cellPropsModel) {
	span := cellSpan(props)
	if span > 1 {
		_ = cell.Merge(span, 1)
	}
	if props.hasVAlign {
		_ = cell.SetVerticalAlignment(domain.VerticalAlignCenter)
	}
	switch props.verticalMerge {
	case wml.ST_MergeRestart:
		_ = cell.SetVMerge(domain.VMergeRestart)
	case wml.ST_MergeContinue:
		_ = cell.SetVMerge(domain.VMergeContinue)
	}
	if props.hasShading {
		_ = cell.SetShading(props.shading)
	}
	if props.borders.enabled {
		_ = cell.SetBorders(domain.TableBorders{
			Top:    borderStyle(props.borders),
			Bottom: borderStyle(props.borders),
			Left:   borderStyle(props.borders),
			Right:  borderStyle(props.borders),
		})
	}
}

func applyRunProperties(run domain.Run, props runPropsModel) {
	font := props.font
	if props.fontX.EastAsiaAttr != nil {
		font.EastAsia = *props.fontX.EastAsiaAttr
	}
	if props.fontX.AsciiAttr != nil {
		font.Name = *props.fontX.AsciiAttr
	}
	if font.Name == "" && font.EastAsia != "" {
		font.Name = font.EastAsia
	}
	if font.Name != "" {
		_ = run.SetFont(font)
	}
	if props.hasSize {
		_ = run.SetSize(pointsToHalfPoints(props.sizePt))
	}
	if props.bold {
		_ = run.SetBold(true)
	}
	if props.italic {
		_ = run.SetItalic(true)
	}
	if props.strike {
		_ = run.SetStrike(true)
	}
	if props.hasColor {
		_ = run.SetColor(props.color)
	}
}

func tableColumnCount(model *tableModel) int {
	maxCols := 0
	for _, row := range model.rows {
		cols := 0
		for _, cell := range row.cells {
			cols += cellSpan(cell.props)
		}
		if cols > maxCols {
			maxCols = cols
		}
	}
	return maxCols
}

func cellSpan(props cellPropsModel) int {
	span := props.columnSpan
	if props.x != nil && props.x.GridSpan != nil && props.x.GridSpan.ValAttr > 0 {
		span = int(props.x.GridSpan.ValAttr)
	}
	if span < 1 {
		span = 1
	}
	return span
}

func borderStyle(border borderModel) domain.BorderStyle {
	return domain.BorderStyle{
		Style: domain.BorderSingle,
		Width: int(math.Max(1, float64(border.width*8))),
		Color: border.color,
	}
}

func toDomainAlignment(align wml.ST_Jc) domain.Alignment {
	switch align {
	case wml.ST_JcCenter:
		return domain.AlignmentCenter
	case wml.ST_JcRight:
		return domain.AlignmentRight
	case wml.ST_JcBoth:
		return domain.AlignmentJustify
	default:
		return domain.AlignmentLeft
	}
}

func pointsToTwips(value float64) int {
	return int(math.Round(float64(value) * 20))
}

func pointsToHalfPoints(value float64) int {
	halfPoints := int(math.Round(value * 2))
	if halfPoints < 2 {
		return 2
	}
	return halfPoints
}

func imageSize(width, height float64) domain.ImageSize {
	wPt := float64(width)
	hPt := float64(height)
	if wPt <= 0 {
		wPt = 300
	}
	if hPt <= 0 {
		hPt = 200
	}
	const emuPerPoint = 12700
	return domain.ImageSize{
		WidthPx:   int(math.Round(wPt * 96 / 72)),
		HeightPx:  int(math.Round(hPt * 96 / 72)),
		WidthEMU:  int(math.Round(wPt * emuPerPoint)),
		HeightEMU: int(math.Round(hPt * emuPerPoint)),
	}
}
