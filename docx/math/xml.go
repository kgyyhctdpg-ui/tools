package math

import (
	"encoding/xml"
	"strconv"
	"strings"

	mathSchema "github.com/scoming-dev/tools/docx/schema/soo/ofc/math"
)

type xmlStyle struct {
	FontSizePt int
}

// OMathXML serializes an inline Office Math expression to OMML XML.
func OMathXML(oMath *mathSchema.OMath) string {
	return buildOMathXML(oMath, xmlStyle{})
}

// OMathXMLWithFontSize serializes an inline Office Math expression using the given point size.
func OMathXMLWithFontSize(oMath *mathSchema.OMath, fontSizePt int) string {
	return buildOMathXML(oMath, xmlStyle{FontSizePt: fontSizePt})
}

func buildOMathXML(oMath *mathSchema.OMath, style xmlStyle) string {
	if oMath == nil {
		return ""
	}
	var out strings.Builder
	writeOMath(&out, oMath, style)
	return out.String()
}

// OMathParaXML serializes a block Office Math expression to OMML XML.
func OMathParaXML(oMathPara *mathSchema.OMathPara) string {
	return buildOMathParaXML(oMathPara, xmlStyle{})
}

// OMathParaXMLWithFontSize serializes a block Office Math expression using the given point size.
func OMathParaXMLWithFontSize(oMathPara *mathSchema.OMathPara, fontSizePt int) string {
	return buildOMathParaXML(oMathPara, xmlStyle{FontSizePt: fontSizePt})
}

func buildOMathParaXML(oMathPara *mathSchema.OMathPara, style xmlStyle) string {
	if oMathPara == nil {
		return ""
	}
	var out strings.Builder
	out.WriteString("<m:oMathPara>")
	for _, oMath := range oMathPara.OMath {
		writeOMath(&out, oMath, style)
	}
	out.WriteString("</m:oMathPara>")
	return out.String()
}

// OMathParaFromOMath wraps an inline Office Math expression as a block expression.
func OMathParaFromOMath(oMath *mathSchema.OMath) *mathSchema.OMathPara {
	oMathPara := mathSchema.NewOMathPara()
	if oMath != nil {
		oMathPara.OMath = append(oMathPara.OMath, oMath)
	}
	return oMathPara
}

// TextOMathPara creates a block Office Math expression from plain text.
func TextOMathPara(text string) *mathSchema.OMathPara {
	return OMathParaFromOMath(CreateTextOMath(text))
}

func writeOMath(out *strings.Builder, oMath *mathSchema.OMath, style xmlStyle) {
	out.WriteString("<m:oMath>")
	for _, elem := range oMath.EG_OMathMathElements {
		writeElement(out, elem, style)
	}
	out.WriteString("</m:oMath>")
}

func writeElement(out *strings.Builder, elem *mathSchema.EG_OMathMathElements, style xmlStyle) {
	if elem == nil {
		return
	}
	switch {
	case elem.R != nil:
		writeRun(out, elem.R, style)
	case elem.F != nil:
		writeFraction(out, elem.F, style)
	case elem.Rad != nil:
		writeRadical(out, elem.Rad, style)
	case elem.SSub != nil:
		writeScript(out, "m:sSub", elem.SSub.E, "m:sub", elem.SSub.Sub, style)
	case elem.SSup != nil:
		writeScript(out, "m:sSup", elem.SSup.E, "m:sup", elem.SSup.Sup, style)
	case elem.SSubSup != nil:
		writeSubSup(out, elem.SSubSup, style)
	case elem.Nary != nil:
		writeNary(out, elem.Nary, style)
	case elem.LimLow != nil:
		writeLimit(out, "m:limLow", elem.LimLow.E, elem.LimLow.Lim, style)
	case elem.LimUpp != nil:
		writeLimit(out, "m:limUpp", elem.LimUpp.E, elem.LimUpp.Lim, style)
	case elem.Acc != nil:
		writeAccent(out, elem.Acc, style)
	case elem.D != nil:
		writeDelimited(out, elem.D, style)
	case elem.M != nil:
		writeMatrix(out, elem.M, style)
	}
}

func writeRun(out *strings.Builder, run *mathSchema.CT_R, style xmlStyle) {
	out.WriteString("<m:r>")
	writeRunProperties(out, style)
	for _, choice := range run.Choice {
		for _, text := range choice.T {
			writeText(out, text.Content)
		}
	}
	out.WriteString("</m:r>")
}

func writeRunProperties(out *strings.Builder, style xmlStyle) {
	if style.FontSizePt <= 0 {
		return
	}
	size := strconv.Itoa(style.FontSizePt * 2)
	out.WriteString(`<w:rPr><w:sz w:val="`)
	out.WriteString(size)
	out.WriteString(`"/><w:szCs w:val="`)
	out.WriteString(size)
	out.WriteString(`"/></w:rPr>`)
}

func writeText(out *strings.Builder, text string) {
	if strings.TrimSpace(text) != text {
		out.WriteString(`<m:t xml:space="preserve">`)
	} else {
		out.WriteString("<m:t>")
	}
	out.WriteString(xmlEscape(text))
	out.WriteString("</m:t>")
}

func writeFraction(out *strings.Builder, fraction *mathSchema.CT_F, style xmlStyle) {
	out.WriteString("<m:f>")
	if fraction.FPr != nil && fraction.FPr.Type != nil && fraction.FPr.Type.ValAttr != "" {
		out.WriteString(`<m:fPr><m:type m:val="`)
		out.WriteString(xmlEscape(fraction.FPr.Type.ValAttr))
		out.WriteString(`"/></m:fPr>`)
	}
	writeArg(out, "m:num", fraction.Num, style)
	writeArg(out, "m:den", fraction.Den, style)
	out.WriteString("</m:f>")
}

func writeRadical(out *strings.Builder, radical *mathSchema.CT_Rad, style xmlStyle) {
	out.WriteString("<m:rad>")
	if radical.RadPr != nil && radical.RadPr.DegHide != nil {
		out.WriteString(`<m:radPr><m:degHide m:val="on"/></m:radPr>`)
	}
	if radical.Deg != nil {
		writeArg(out, "m:deg", radical.Deg, style)
	}
	writeArg(out, "m:e", radical.E, style)
	out.WriteString("</m:rad>")
}

func writeScript(out *strings.Builder, tag string, base *mathSchema.CT_OMathArg, scriptTag string, script *mathSchema.CT_OMathArg, style xmlStyle) {
	out.WriteString("<")
	out.WriteString(tag)
	out.WriteString(">")
	writeArg(out, "m:e", base, style)
	writeArg(out, scriptTag, script, style)
	out.WriteString("</")
	out.WriteString(tag)
	out.WriteString(">")
}

func writeSubSup(out *strings.Builder, subSup *mathSchema.CT_SSubSup, style xmlStyle) {
	out.WriteString("<m:sSubSup>")
	writeArg(out, "m:e", subSup.E, style)
	writeArg(out, "m:sub", subSup.Sub, style)
	writeArg(out, "m:sup", subSup.Sup, style)
	out.WriteString("</m:sSubSup>")
}

func writeNary(out *strings.Builder, nary *mathSchema.CT_Nary, style xmlStyle) {
	out.WriteString("<m:nary>")
	if nary.NaryPr != nil && nary.NaryPr.Chr != nil && nary.NaryPr.Chr.ValAttr != "" {
		out.WriteString(`<m:naryPr><m:chr m:val="`)
		out.WriteString(xmlEscape(nary.NaryPr.Chr.ValAttr))
		out.WriteString(`"/></m:naryPr>`)
	}
	writeArg(out, "m:sub", nary.Sub, style)
	writeArg(out, "m:sup", nary.Sup, style)
	writeArg(out, "m:e", nary.E, style)
	out.WriteString("</m:nary>")
}

func writeLimit(out *strings.Builder, tag string, base, limit *mathSchema.CT_OMathArg, style xmlStyle) {
	out.WriteString("<")
	out.WriteString(tag)
	out.WriteString(">")
	writeArg(out, "m:e", base, style)
	writeArg(out, "m:lim", limit, style)
	out.WriteString("</")
	out.WriteString(tag)
	out.WriteString(">")
}

func writeAccent(out *strings.Builder, accent *mathSchema.CT_Acc, style xmlStyle) {
	out.WriteString("<m:acc>")
	if accent.AccPr != nil && accent.AccPr.Chr != nil && accent.AccPr.Chr.ValAttr != "" {
		out.WriteString(`<m:accPr><m:chr m:val="`)
		out.WriteString(xmlEscape(accent.AccPr.Chr.ValAttr))
		out.WriteString(`"/></m:accPr>`)
	}
	writeArg(out, "m:e", accent.E, style)
	out.WriteString("</m:acc>")
}

func writeDelimited(out *strings.Builder, delimited *mathSchema.CT_D, style xmlStyle) {
	out.WriteString("<m:d>")
	if delimited.DPr != nil {
		out.WriteString("<m:dPr>")
		if delimited.DPr.BegChr != nil && delimited.DPr.BegChr.ValAttr != "" {
			out.WriteString(`<m:begChr m:val="`)
			out.WriteString(xmlEscape(delimited.DPr.BegChr.ValAttr))
			out.WriteString(`"/>`)
		}
		if delimited.DPr.EndChr != nil && delimited.DPr.EndChr.ValAttr != "" {
			out.WriteString(`<m:endChr m:val="`)
			out.WriteString(xmlEscape(delimited.DPr.EndChr.ValAttr))
			out.WriteString(`"/>`)
		}
		out.WriteString("</m:dPr>")
	}
	for _, elem := range delimited.E {
		writeArg(out, "m:e", elem, style)
	}
	out.WriteString("</m:d>")
}

func writeMatrix(out *strings.Builder, matrix *mathSchema.CT_M, style xmlStyle) {
	out.WriteString("<m:m>")
	if matrix.MPr != nil {
		out.WriteString("<m:mPr/>")
	}
	for _, row := range matrix.Mr {
		out.WriteString("<m:mr>")
		for _, cell := range row.E {
			writeArg(out, "m:e", cell, style)
		}
		out.WriteString("</m:mr>")
	}
	out.WriteString("</m:m>")
}

func writeArg(out *strings.Builder, tag string, arg *mathSchema.CT_OMathArg, style xmlStyle) {
	if arg == nil {
		return
	}
	out.WriteString("<")
	out.WriteString(tag)
	out.WriteString(">")
	for _, elem := range arg.EG_OMathMathElements {
		writeElement(out, elem, style)
	}
	out.WriteString("</")
	out.WriteString(tag)
	out.WriteString(">")
}

func xmlEscape(value string) string {
	var out strings.Builder
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
}
