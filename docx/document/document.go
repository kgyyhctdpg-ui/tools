package document

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strings"

	docxgo "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"
	compatcolor "github.com/scoming-dev/tools/docx/color"
	"github.com/scoming-dev/tools/docx/measurement"
	"github.com/scoming-dev/tools/docx/media"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

type Field int

const (
	FieldCurrentPage Field = iota
	FieldNumberOfPages
)

type Document struct {
	Styles    *Styles
	Numbering *Numbering

	blocks        []*blockModel
	paragraphs    []*paragraphModel
	defaultHeader *headerFooterModel
	defaultFooter *headerFooterModel
	lastLandscape bool
}

type blockModel struct {
	paragraph *paragraphModel
	table     *tableModel
	section   *sectionModel
}

type paragraphModel struct {
	runs         []*runModel
	props        paragraphPropsModel
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
}

type runContentType int

const (
	runContentText runContentType = iota
	runContentBreak
	runContentField
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
	charSpace measurement.Distance
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
	width   measurement.Distance
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

type DocumentXML struct{}

func New() *Document {
	numbering := &Numbering{}
	return &Document{
		Styles:    &Styles{},
		Numbering: numbering,
	}
}

func (d *Document) AddParagraph() Paragraph {
	model := newParagraphModel()
	d.paragraphs = append(d.paragraphs, model)
	d.blocks = append(d.blocks, &blockModel{paragraph: model})
	return Paragraph{model: model}
}

func (d *Document) AddTable() Table {
	model := &tableModel{}
	d.blocks = append(d.blocks, &blockModel{table: model})
	return Table{model: model}
}

func (d *Document) AddHeader() Header {
	model := &headerFooterModel{}
	d.defaultHeader = model
	return Header{model: model}
}

func (d *Document) AddFooter() Footer {
	model := &headerFooterModel{}
	d.defaultFooter = model
	return Footer{model: model}
}

func (d *Document) AddImage(img media.Image) (*ImageRef, error) {
	return &ImageRef{image: img}, nil
}

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

	return out.SaveAs(path)
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

func (d *Document) BodySection() Section {
	return Section{doc: d, model: &sectionModel{}}
}

func (d *Document) Paragraphs() []Paragraph {
	ret := make([]Paragraph, 0, len(d.paragraphs))
	for _, para := range d.paragraphs {
		ret = append(ret, Paragraph{model: para})
	}
	return ret
}

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

func (d *Document) X() *DocumentXML {
	return &DocumentXML{}
}

func (d *Document) AddSectionBreak(orientation domain.Orientation, pageSize domain.PageSize) Section {
	model := &sectionModel{orientation: orientation, pageSize: pageSize, header: d.defaultHeader, footer: d.defaultFooter}
	d.lastLandscape = orientation == domain.OrientationLandscape
	d.blocks = append(d.blocks, &blockModel{section: model})
	return Section{doc: d, model: model}
}

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

type Paragraph struct {
	model *paragraphModel
}

func (p Paragraph) AddRun() Run {
	if p.model == nil {
		return Run{}
	}
	model := &runModel{}
	p.model.runs = append(p.model.runs, model)
	return Run{model: model}
}

func (p Paragraph) Properties() ParagraphProperties {
	return ParagraphProperties{model: p.model}
}

func (p Paragraph) AddHyperLink() HyperLink {
	return HyperLink{para: p.model, model: &hyperlinkModel{}}
}

func (p Paragraph) AddBookmark(id string) {
	if p.model == nil {
		return
	}
	p.model.bookmarkID = id
	p.model.bookmarkName = id
}

func (p Paragraph) SetNumberingDefinition(def Definition) {
	if p.model == nil {
		return
	}
	p.model.numbering = &def
}

func (p Paragraph) X() *wml.CT_P {
	return &wml.CT_P{}
}

type ParagraphProperties struct {
	model *paragraphModel
}

func (p ParagraphProperties) SetAlignment(align wml.ST_Jc) {
	if p.model == nil {
		return
	}
	p.model.props.alignment = align
	p.model.props.hasAlignment = true
}

func (p ParagraphProperties) SetFirstLineIndent(indent measurement.Distance) {
	if p.model == nil {
		return
	}
	p.model.props.firstLineIndent = pointsToTwips(indent)
}

func (p ParagraphProperties) SetStartIndent(indent measurement.Distance) {
	if p.model == nil {
		return
	}
	p.model.props.startIndent = pointsToTwips(indent)
}

func (p ParagraphProperties) Spacing() Spacing {
	return Spacing{model: p.model}
}

func (p ParagraphProperties) X() *wml.CT_PPr {
	if p.model == nil {
		return &wml.CT_PPr{}
	}
	if p.model.props.x == nil {
		p.model.props.x = &wml.CT_PPr{}
	}
	return p.model.props.x
}

func (p ParagraphProperties) AddSection(mark wml.ST_SectionMark) Section {
	sectPr := wml.NewCT_SectPr()
	sectPr.EG_HdrFtrReferences = []*wml.EG_HdrFtrReferences{}
	p.X().SectPr = sectPr
	return Section{model: &sectionModel{}}
}

type Spacing struct {
	model *paragraphModel
}

func (s Spacing) SetBefore(value measurement.Distance) {
	if s.model == nil {
		return
	}
	s.model.props.spacing.before = pointsToTwips(value)
	s.model.props.spacing.hasBefore = true
}

func (s Spacing) SetAfter(value measurement.Distance) {
	if s.model == nil {
		return
	}
	s.model.props.spacing.after = pointsToTwips(value)
	s.model.props.spacing.hasAfter = true
}

func (s Spacing) SetLineSpacing(value measurement.Distance, rule wml.ST_LineSpacingRule) {
	if s.model == nil {
		return
	}
	s.model.props.spacing.line = pointsToTwips(value)
	s.model.props.spacing.lineRule = rule
	s.model.props.spacing.hasLine = true
}

type Run struct {
	model *runModel
}

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

func (r Run) AddBreak() {
	if r.model == nil {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentBreak, breakType: domain.BreakTypeLine})
}

func (r Run) AddPageBreak() {
	if r.model == nil {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentBreak, breakType: domain.BreakTypePage})
}

func (r Run) AddTab() {
	r.AddText("\t")
}

func (r Run) AddField(field Field) {
	if r.model == nil {
		return
	}
	r.model.contents = append(r.model.contents, runContentModel{typ: runContentField, field: field})
}

func (r Run) AddDrawingInline(ref *ImageRef) (*DrawingInline, error) {
	if r.model == nil {
		return &DrawingInline{}, nil
	}
	r.model.image = ref
	return &DrawingInline{ref: ref}, nil
}

func (r Run) Properties() RunProperties {
	if r.model == nil {
		return RunProperties{}
	}
	return RunProperties{props: &r.model.props}
}

type RunProperties struct {
	props *runPropsModel
}

func (p RunProperties) SetStyle(style string) {
	if p.props != nil {
		p.props.style = style
	}
}

func (p RunProperties) SetBold(bold bool) {
	if p.props != nil {
		p.props.bold = bold
	}
}

func (p RunProperties) SetItalic(italic bool) {
	if p.props != nil {
		p.props.italic = italic
	}
}

func (p RunProperties) SetStrikeThrough(strike bool) {
	if p.props != nil {
		p.props.strike = strike
	}
}

func (p RunProperties) SetFontFamily(name string) {
	if p.props == nil {
		return
	}
	p.props.font.Name = name
	p.props.font.EastAsia = name
}

func (p RunProperties) SetSize(size measurement.Distance) {
	if p.props == nil {
		return
	}
	p.props.sizePt = float64(size)
	p.props.hasSize = true
}

func (p RunProperties) SetCharacterSpacing(value measurement.Distance) {
	if p.props != nil {
		p.props.charSpace = value
	}
}

func (p RunProperties) SetColor(color compatcolor.Color) {
	if p.props == nil {
		return
	}
	p.props.color = color
	p.props.hasColor = true
}

func (p RunProperties) Fonts() Fonts {
	return Fonts{props: p.props}
}

type Fonts struct {
	props *runPropsModel
}

func (f Fonts) X() *FontX {
	if f.props == nil {
		return &FontX{}
	}
	return &f.props.fontX
}

type FontX struct {
	AsciiAttr    *string
	HAnsiAttr    *string
	EastAsiaAttr *string
	CsAttr       *string
}

type HyperLink struct {
	para  *paragraphModel
	model *hyperlinkModel
}

type hyperlinkModel struct {
	target string
	x      HyperLinkXML
}

type HyperLinkXML struct {
	AnchorAttr *string
}

func (h HyperLink) SetTarget(target string) {
	if h.model != nil {
		h.model.target = target
	}
}

func (h HyperLink) AddRun() Run {
	if h.para == nil {
		return Run{}
	}
	model := &runModel{hyperlink: h.model}
	h.para.runs = append(h.para.runs, model)
	return Run{model: model}
}

func (h HyperLink) X() *HyperLinkXML {
	if h.model == nil {
		return &HyperLinkXML{}
	}
	return &h.model.x
}

type Table struct {
	model *tableModel
}

func (t Table) AddRow() Row {
	if t.model == nil {
		return Row{}
	}
	model := &rowModel{}
	t.model.rows = append(t.model.rows, model)
	return Row{model: model}
}

func (t Table) Properties() TableProperties {
	return TableProperties{model: t.model}
}

type Row struct {
	model *rowModel
}

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

type Cell struct {
	model *cellModel
}

func (c Cell) AddParagraph() Paragraph {
	if c.model == nil {
		return Paragraph{}
	}
	model := newParagraphModel()
	c.model.paragraphs = append(c.model.paragraphs, model)
	return Paragraph{model: model}
}

func (c Cell) Properties() CellProperties {
	return CellProperties{model: c.model}
}

type TableProperties struct {
	model *tableModel
}

func (p TableProperties) SetWidthPercent(percent float64) {
	if p.model != nil {
		p.model.props.widthPercent = percent
	}
}

func (p TableProperties) SetAlignment(align wml.ST_Jc) {
	if p.model == nil {
		return
	}
	p.model.props.alignment = align
	p.model.props.hasAlignment = true
}

func (p TableProperties) Borders() Borders {
	if p.model == nil {
		return Borders{}
	}
	return Borders{border: &p.model.props.borders}
}

type CellProperties struct {
	model *cellModel
}

func (p CellProperties) SetVerticalAlignment(align wml.ST_VerticalJc) {
	if p.model == nil {
		return
	}
	p.model.props.verticalAlignment = align
	p.model.props.hasVAlign = true
}

func (p CellProperties) SetVerticalMerge(merge wml.ST_Merge) {
	if p.model != nil {
		p.model.props.verticalMerge = merge
	}
}

func (p CellProperties) SetColumnSpan(span int) {
	if p.model != nil && span > 0 {
		p.model.props.columnSpan = span
	}
}

func (p CellProperties) SetShading(_ wml.ST_Shd, fill compatcolor.Color, _ compatcolor.Color) {
	if p.model == nil {
		return
	}
	p.model.props.shading = fill
	p.model.props.hasShading = true
}

func (p CellProperties) Borders() Borders {
	if p.model == nil {
		return Borders{}
	}
	return Borders{border: &p.model.props.borders}
}

func (p CellProperties) X() *wml.CT_TcPr {
	if p.model == nil {
		return &wml.CT_TcPr{}
	}
	if p.model.props.x == nil {
		p.model.props.x = &wml.CT_TcPr{}
	}
	return p.model.props.x
}

type Borders struct {
	border *borderModel
}

func (b Borders) SetAll(_ wml.ST_Border, color compatcolor.Color, width measurement.Distance) {
	if b.border == nil {
		return
	}
	b.border.enabled = true
	b.border.color = color
	b.border.width = width
}

type ImageRef struct {
	image  media.Image
	width  measurement.Distance
	height measurement.Distance
}

type DrawingInline struct {
	ref *ImageRef
}

func (d *DrawingInline) SetSize(width, height measurement.Distance) {
	if d == nil || d.ref == nil {
		return
	}
	d.ref.width = width
	d.ref.height = height
}

type Header struct {
	model *headerFooterModel
}

func (h Header) AddParagraph() Paragraph {
	if h.model == nil {
		return Paragraph{}
	}
	model := newParagraphModel()
	h.model.paragraphs = append(h.model.paragraphs, model)
	return Paragraph{model: model}
}

type Footer struct {
	model *headerFooterModel
}

func (f Footer) AddParagraph() Paragraph {
	if f.model == nil {
		return Paragraph{}
	}
	model := newParagraphModel()
	f.model.paragraphs = append(f.model.paragraphs, model)
	return Paragraph{model: model}
}

type Section struct {
	doc   *Document
	model *sectionModel
}

func (s Section) SetHeader(header Header, _ wml.ST_HdrFtr) {
	if s.doc != nil {
		s.doc.defaultHeader = header.model
	}
	if s.model != nil {
		s.model.header = header.model
	}
}

func (s Section) SetFooter(footer Footer, _ wml.ST_HdrFtr) {
	if s.doc != nil {
		s.doc.defaultFooter = footer.model
	}
	if s.model != nil {
		s.model.footer = footer.model
	}
}

type Styles struct{}

func (s *Styles) AddStyle(_ string, _ wml.ST_StyleType, _ bool) Style {
	return Style{props: &runPropsModel{}}
}

type Style struct {
	props *runPropsModel
}

func (s Style) SetName(_ string)    {}
func (s Style) SetBasedOn(_ string) {}
func (s Style) RunProperties() RunProperties {
	return RunProperties{props: s.props}
}

type Numbering struct {
	definitions []*definitionModel
	nextID      int
}

func (n *Numbering) AddDefinition() Definition {
	if n == nil {
		return Definition{}
	}
	n.nextID++
	model := &definitionModel{
		id:         n.nextID,
		abstractID: n.nextID,
	}
	n.definitions = append(n.definitions, model)
	return Definition{model: model}
}

type Definition struct {
	model *definitionModel
}

type definitionModel struct {
	id         int
	abstractID int
	level      *levelModel
}

func (d *Definition) AddLevel() Level {
	if d == nil || d.model == nil {
		return Level{}
	}
	d.model.level = &levelModel{props: &runPropsModel{}, start: 1}
	return Level{model: d.model.level}
}

func (d Definition) NumberID() int {
	if d.model == nil {
		return 0
	}
	return d.model.id
}

func (d Definition) LevelIndex() int {
	if d.model == nil || d.model.level == nil {
		return 0
	}
	return d.model.level.index
}

type Level struct {
	model *levelModel
}

type levelModel struct {
	index      int
	format     wml.ST_NumberFormat
	text       string
	suffix     string
	start      int
	leftIndent int
	hanging    int
	props      *runPropsModel
	hasFormat  bool
	hasIndent  bool
}

func (l Level) SetFormat(format wml.ST_NumberFormat) {
	if l.model == nil {
		return
	}
	l.model.format = format
	l.model.hasFormat = true
}

func (l Level) SetText(text string) {
	if l.model != nil {
		l.model.text = text
	}
}

func (l Level) SetSuffix(suffix string) {
	if l.model == nil {
		return
	}
	l.model.suffix = normalizeNumberingSuffix(suffix)
}

func (l Level) SetStart(start int) {
	if l.model == nil {
		return
	}
	if start < 1 {
		start = 1
	}
	l.model.start = start
}

func (l Level) SetIndent(left, hanging measurement.Distance) {
	if l.model == nil {
		return
	}
	l.model.leftIndent = pointsToTwips(left)
	l.model.hanging = pointsToTwips(hanging)
	l.model.hasIndent = true
}

func (l Level) RunProperties() RunProperties {
	if l.model == nil {
		return RunProperties{}
	}
	if l.model.props == nil {
		l.model.props = &runPropsModel{}
	}
	return RunProperties{props: l.model.props}
}

func (n *Numbering) XML() []byte {
	if n == nil || len(n.definitions) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString(`<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	for _, def := range n.definitions {
		if def == nil || def.level == nil {
			continue
		}
		level := def.level
		buf.WriteString(fmt.Sprintf(`<w:abstractNum w:abstractNumId="%d">`, def.abstractID))
		buf.WriteString(`<w:multiLevelType w:val="singleLevel"/>`)
		buf.WriteString(fmt.Sprintf(`<w:lvl w:ilvl="%d">`, level.index))
		buf.WriteString(fmt.Sprintf(`<w:start w:val="%d"/>`, normalizeNumberingStart(level.start)))
		buf.WriteString(fmt.Sprintf(`<w:numFmt w:val="%s"/>`, level.formatValue()))
		if level.suffix != "" {
			buf.WriteString(fmt.Sprintf(`<w:suff w:val="%s"/>`, xmlEscapeAttr(level.suffix)))
		}
		buf.WriteString(fmt.Sprintf(`<w:lvlText w:val="%s"/>`, xmlEscapeAttr(level.textValue())))
		buf.WriteString(`<w:lvlJc w:val="left"/>`)
		if level.hasIndent {
			buf.WriteString(fmt.Sprintf(`<w:pPr><w:ind w:left="%d" w:hanging="%d"/></w:pPr>`, level.leftIndent, level.hanging))
		}
		if runPropsXML := numberingRunPropertiesXML(level.props); runPropsXML != "" {
			buf.WriteString(runPropsXML)
		}
		buf.WriteString(`</w:lvl>`)
		buf.WriteString(`</w:abstractNum>`)
		buf.WriteString(fmt.Sprintf(`<w:num w:numId="%d"><w:abstractNumId w:val="%d"/></w:num>`, def.id, def.abstractID))
	}
	buf.WriteString(`</w:numbering>`)
	return buf.Bytes()
}

func (l *levelModel) formatValue() string {
	if l == nil {
		return "decimal"
	}
	if l.hasFormat && l.format == wml.ST_NumberFormatBullet {
		return "bullet"
	}
	return "decimal"
}

func (l *levelModel) textValue() string {
	if l == nil {
		return "%1."
	}
	if l.text != "" {
		return l.text
	}
	if l.hasFormat && l.format == wml.ST_NumberFormatBullet {
		return "•"
	}
	return "%1."
}

func normalizeNumberingStart(start int) int {
	if start < 1 {
		return 1
	}
	return start
}

func normalizeNumberingSuffix(suffix string) string {
	switch strings.TrimSpace(suffix) {
	case "tab", "space", "nothing":
		return strings.TrimSpace(suffix)
	default:
		return "space"
	}
}

func numberingRunPropertiesXML(props *runPropsModel) string {
	if props == nil {
		return ""
	}

	var buf bytes.Buffer
	if props.font.Name != "" || props.font.EastAsia != "" || props.font.CS != "" || props.fontX.AsciiAttr != nil || props.fontX.EastAsiaAttr != nil {
		font := props.font
		if props.fontX.AsciiAttr != nil {
			font.Name = *props.fontX.AsciiAttr
		}
		if props.fontX.EastAsiaAttr != nil {
			font.EastAsia = *props.fontX.EastAsiaAttr
		}
		if font.Name == "" && font.EastAsia != "" {
			font.Name = font.EastAsia
		}
		buf.WriteString(`<w:rFonts`)
		if font.Name != "" {
			escaped := xmlEscapeAttr(font.Name)
			buf.WriteString(fmt.Sprintf(` w:ascii="%s" w:hAnsi="%s"`, escaped, escaped))
		}
		if font.EastAsia != "" {
			buf.WriteString(fmt.Sprintf(` w:eastAsia="%s"`, xmlEscapeAttr(font.EastAsia)))
		}
		if font.CS != "" {
			buf.WriteString(fmt.Sprintf(` w:cs="%s"`, xmlEscapeAttr(font.CS)))
		}
		buf.WriteString(`/>`)
	}
	if props.hasSize {
		size := pointsToHalfPoints(props.sizePt)
		buf.WriteString(fmt.Sprintf(`<w:sz w:val="%d"/>`, size))
		buf.WriteString(fmt.Sprintf(`<w:szCs w:val="%d"/>`, size))
	}
	if props.bold {
		buf.WriteString(`<w:b/>`)
	}
	if props.italic {
		buf.WriteString(`<w:i/>`)
	}
	if props.hasColor {
		buf.WriteString(fmt.Sprintf(`<w:color w:val="%02X%02X%02X"/>`, props.color.R, props.color.G, props.color.B))
	}
	if buf.Len() == 0 {
		return ""
	}
	return `<w:rPr>` + buf.String() + `</w:rPr>`
}

func xmlEscapeAttr(value string) string {
	var buf strings.Builder
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}

func renderParagraph(add func() (domain.Paragraph, error), model *paragraphModel) error {
	out, err := add()
	if err != nil {
		return err
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

func pointsToTwips(value measurement.Distance) int {
	return int(math.Round(float64(value) * 20))
}

func pointsToHalfPoints(value float64) int {
	halfPoints := int(math.Round(value * 2))
	if halfPoints < 2 {
		return 2
	}
	return halfPoints
}

func imageSize(width, height measurement.Distance) domain.ImageSize {
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
