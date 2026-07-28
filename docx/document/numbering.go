package document

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

// Numbering stores DOCX numbering definitions.
type Numbering struct {
	definitions []*definitionModel
	nextID      int
}

// AddDefinition creates a numbering definition.
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

// Definition represents one numbering definition.
type Definition struct {
	model *definitionModel
}

type definitionModel struct {
	id         int
	abstractID int
	level      *levelModel
}

// AddLevel creates the single supported numbering level for the definition.
func (d *Definition) AddLevel() Level {
	if d == nil || d.model == nil {
		return Level{}
	}
	d.model.level = &levelModel{props: &runPropsModel{}, start: 1}
	return Level{model: d.model.level}
}

// NumberID returns the DOCX numbering id.
func (d Definition) NumberID() int {
	if d.model == nil {
		return 0
	}
	return d.model.id
}

// LevelIndex returns the numbering level index.
func (d Definition) LevelIndex() int {
	if d.model == nil || d.model.level == nil {
		return 0
	}
	return d.model.level.index
}

// Level represents a numbering level.
type Level struct {
	model *levelModel
}

type levelModel struct {
	index      int
	format     wml.ST_NumberFormat
	formatName string
	text       string
	suffix     string
	start      int
	leftIndent int
	hanging    int
	props      *runPropsModel
	hasFormat  bool
	hasIndent  bool
}

// SetFormat sets the numbering format.
func (l Level) SetFormat(format wml.ST_NumberFormat) {
	if l.model == nil {
		return
	}
	l.model.format = format
	l.model.formatName = ""
	l.model.hasFormat = true
}

// SetFormatValue sets a raw OOXML numbering format value, such as upperLetter.
func (l Level) SetFormatValue(format string) {
	if l.model == nil {
		return
	}
	format = strings.TrimSpace(format)
	if format == "" {
		return
	}
	l.model.formatName = format
	l.model.hasFormat = true
}

// SetText sets the numbering marker pattern.
func (l Level) SetText(text string) {
	if l.model != nil {
		l.model.text = text
	}
}

// SetSuffix sets the spacing suffix after the numbering marker.
func (l Level) SetSuffix(suffix string) {
	if l.model == nil {
		return
	}
	l.model.suffix = normalizeNumberingSuffix(suffix)
}

// SetStart sets the starting number.
func (l Level) SetStart(start int) {
	if l.model == nil {
		return
	}
	if start < 1 {
		start = 1
	}
	l.model.start = start
}

// SetIndent sets numbering left and hanging indents.
func (l Level) SetIndent(left, hanging float64) {
	if l.model == nil {
		return
	}
	l.model.leftIndent = pointsToTwips(left)
	l.model.hanging = pointsToTwips(hanging)
	l.model.hasIndent = true
}

// RunProperties returns styling for the numbering marker.
func (l Level) RunProperties() RunProperties {
	if l.model == nil {
		return RunProperties{}
	}
	if l.model.props == nil {
		l.model.props = &runPropsModel{}
	}
	return RunProperties{props: l.model.props}
}

// XML renders the numbering definitions as a numbering.xml part.
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
	if l.formatName != "" {
		return l.formatName
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
