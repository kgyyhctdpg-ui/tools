package renderer

import (
	"fmt"

	"github.com/88250/lute/ast"
	"github.com/scoming-dev/tools/docx/document"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

type listState struct {
	definition          document.Definition
	checkedDefinition   document.Definition
	uncheckedDefinition document.Definition
}

func (r *DocxRenderer) renderHeading(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		para := r.doc.AddParagraph()
		// Built-in heading styles let WPS/Word rebuild generated TOC fields.
		para.SetStyle(headingStyleID(node.HeadingLevel))
		r.setParagraphSpacing(&para)

		headingText := node.Text()
		if bookmarkID, exists := r.headingBookmarks[headingText]; exists {
			para.AddBookmark(bookmarkID)
		}

		run := para.AddRun()
		switch node.HeadingLevel {
		case 1:
			if r.headingStyle == HeadingStyleLevelOneCenterPageBreak {
				para.Properties().SetAlignment(r.config.Heading.LevelOneAlignment)
				lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
				para.Properties().Spacing().SetLineSpacing(float64(lineHeight), wml.ST_LineSpacingRuleAuto)
				props := run.Properties()
				r.SetTitleFont(&props, r.config.Fonts.Title)
				run.AddPageBreak()
			} else if r.headingStyle == HeadingStyleLevelOneLeftPageBreak {
				para.Properties().SetAlignment(wml.ST_JcLeft)
				lineHeight := float64(r.config.Text.ContentSize) * r.config.Paragraph.LineHeightMultiplier
				para.Properties().Spacing().SetLineSpacing(float64(lineHeight), wml.ST_LineSpacingRuleAuto)
				props := run.Properties()
				r.SetTitleFont(&props, r.config.Fonts.Title)
				run.AddPageBreak()
			}
		default:
			props := run.Properties()
			r.SetSubTitleFont(&props, r.config.Fonts.Title)
		}
		if r.headingStyle == HeadingStyleDefault {
			props := run.Properties()
			r.SetSubTitleFont(&props, r.config.Fonts.Title)
			r.setFontSize(&props, r.headingFontSize(node.HeadingLevel))
		}

		run.AddText(headingText)
	}
	return ast.WalkContinue
}

func headingStyleID(level int) string {
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}
	return fmt.Sprintf("Heading%d", level)
}

func (r *DocxRenderer) headingFontSize(level int) int {
	if level <= 0 {
		return r.config.Text.SubTitleSize
	}
	index := level - 1
	if index >= len(r.config.Heading.Sizes) {
		return r.config.Heading.Sizes[len(r.config.Heading.Sizes)-1]
	}
	return r.config.Heading.Sizes[index]
}

func (r *DocxRenderer) renderHeadingC8hMarker(node *ast.Node, entering bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderList(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.listStack = append(r.listStack, r.newListState(node, len(r.listStack)))
	} else {
		if len(r.listStack) > 0 {
			r.listStack = r.listStack[:len(r.listStack)-1]
		}
		r.Newline()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) renderListItem(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		paragraph := r.doc.AddParagraph()
		r.setParagraphSpacing(&paragraph)
		r.setListFirstLineIndent(&paragraph)
		r.pushPara(&paragraph)

		if definition := r.listDefinitionForItem(node); definition.NumberID() > 0 {
			paragraph.SetNumberingDefinition(definition)
		}
	} else {
		r.popPara()
	}
	return ast.WalkContinue
}

func (r *DocxRenderer) newListState(node *ast.Node, nestedLevel int) listState {
	if node == nil || node.ListData == nil {
		return listState{}
	}
	if isTaskList(node.ListData) {
		return listState{
			checkedDefinition:   r.newListDefinition(node.ListData, "☑", nestedLevel),
			uncheckedDefinition: r.newListDefinition(node.ListData, "□", nestedLevel),
		}
	}
	return listState{definition: r.newListDefinition(node.ListData, "", nestedLevel)}
}

func (r *DocxRenderer) listDefinitionForItem(node *ast.Node) document.Definition {
	if len(r.listStack) == 0 || node == nil || node.ListData == nil {
		return document.Definition{}
	}
	state := r.listStack[len(r.listStack)-1]
	if isTaskList(node.ListData) {
		if node.ListData.Checked {
			return state.checkedDefinition
		}
		return state.uncheckedDefinition
	}
	return state.definition
}

func (r *DocxRenderer) newListDefinition(data *ast.ListData, markerOverride string, nestedLevel int) document.Definition {
	definition := r.doc.Numbering.AddDefinition()
	level := definition.AddLevel()
	level.SetStart(listStart(data))
	level.SetSuffix(r.config.List.MarkerSuffix)

	switch {
	case isTaskList(data):
		level.SetFormat(wml.ST_NumberFormatBullet)
		level.RunProperties().SetSize(float64(r.config.List.SecondLevelFontSize))
		if markerOverride != "" {
			level.SetText(markerOverride)
		} else {
			level.SetText("□")
		}
	default:
		style := orderedListNumberingStyle(nestedLevel)
		level.SetFormatValue(style.format)
		level.RunProperties().SetSize(float64(r.config.List.ThirdLevelFontSize))
		level.RunProperties().SetFontFamily(r.config.List.ThirdLevelFont)
		level.SetText(style.pattern)
	}
	return definition
}

func isTaskList(data *ast.ListData) bool {
	return data != nil && data.Typ == 3 && data.BulletChar != 0
}

func listStart(data *ast.ListData) int {
	if data == nil {
		return 1
	}
	if data.Start > 0 {
		return data.Start
	}
	if data.Num > 0 {
		return data.Num
	}
	return 1
}

func (r *DocxRenderer) renderTaskListItemMarker(_ *ast.Node, _ bool) ast.WalkStatus {
	return ast.WalkContinue
}

func (r *DocxRenderer) renderThematicBreak(_ *ast.Node, _ bool) ast.WalkStatus {
	return ast.WalkContinue
}
