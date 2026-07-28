package renderer

import (
	"strconv"
	"strings"
)

const orderedListStyleCount = 7

type listNumberingStyle struct {
	format  string
	pattern string
}

// orderedListNumberingStyle keeps list markers as native DOCX numbering.
func orderedListNumberingStyle(nestedLevel int) listNumberingStyle {
	switch normalizeOrderedListLevel(nestedLevel) {
	case 0:
		return listNumberingStyle{format: "chineseCounting", pattern: "（%1）"}
	case 1:
		return listNumberingStyle{format: "decimal", pattern: "     %1."}
	case 2:
		return listNumberingStyle{format: "decimal", pattern: " （%1）"}
	case 3:
		return listNumberingStyle{format: "decimalEnclosedCircle", pattern: "    %1"}
	case 4:
		return listNumberingStyle{format: "upperLetter", pattern: "     %1."}
	case 5:
		return listNumberingStyle{format: "lowerLetter", pattern: "     %1."}
	default:
		return listNumberingStyle{format: "decimal", pattern: "     %1）"}
	}
}

func normalizeOrderedListLevel(nestedLevel int) int {
	if nestedLevel < 0 {
		return 0
	}
	return nestedLevel % orderedListStyleCount
}

func (r *DocxRenderer) getNumberingLevel(nestedLevel int, num int) string {
	return numberingLevelLabel(nestedLevel, num)
}

// numberingLevelLabel mirrors the DOCX numbering styles for previews and fallbacks.
func numberingLevelLabel(nestedLevel int, num int) string {
	if num < 1 {
		num = 1
	}
	switch normalizeOrderedListLevel(nestedLevel) {
	case 0:
		return "（" + chineseNumber(num) + "）"
	case 1:
		return strconv.Itoa(num) + "."
	case 2:
		return "（" + strconv.Itoa(num) + "）"
	case 3:
		return circledNumber(num)
	case 4:
		return alphaNumber(num, true) + "."
	case 5:
		return alphaNumber(num, false) + "."
	default:
		return strconv.Itoa(num) + "）"
	}
}

func chineseNumber(num int) string {
	if num <= 0 {
		return "零"
	}

	sections := make([]int, 0, 5)
	for num > 0 {
		sections = append(sections, num%10000)
		num /= 10000
	}

	parts := make([]string, 0, len(sections)*2)
	zeroPending := false
	for index := len(sections) - 1; index >= 0; index-- {
		section := sections[index]
		if section == 0 {
			zeroPending = len(parts) > 0
			continue
		}
		if len(parts) > 0 && (zeroPending || section < 1000) {
			parts = append(parts, "零")
		}
		parts = append(parts, chineseSection(section)+chineseSectionUnit(index))
		zeroPending = false
	}
	return strings.Join(parts, "")
}

func chineseSection(section int) string {
	digits := []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	units := []string{"", "十", "百", "千"}

	parts := make([]string, 0, 8)
	zeroPending := false
	for unitIndex := 3; unitIndex >= 0; unitIndex-- {
		unitBase := 1
		for i := 0; i < unitIndex; i++ {
			unitBase *= 10
		}
		digit := section / unitBase % 10
		if digit == 0 {
			zeroPending = len(parts) > 0
			continue
		}
		if zeroPending {
			parts = append(parts, "零")
			zeroPending = false
		}
		if !(digit == 1 && unitIndex == 1 && len(parts) == 0 && section < 20) {
			parts = append(parts, digits[digit])
		}
		parts = append(parts, units[unitIndex])
	}
	return strings.Join(parts, "")
}

func chineseSectionUnit(index int) string {
	units := []string{"", "万", "亿", "兆", "京", "垓", "秭", "穰"}
	if index >= 0 && index < len(units) {
		return units[index]
	}
	return ""
}

func circledNumber(num int) string {
	switch {
	case num <= 0:
		return "①"
	case num <= 20:
		return string(rune(0x2460 + num - 1))
	case num <= 35:
		return string(rune(0x3251 + num - 21))
	case num <= 50:
		return string(rune(0x32B1 + num - 36))
	default:
		// Unicode only defines circled numbers up to 50.
		return "（" + strconv.Itoa(num) + "）"
	}
}

func alphaNumber(num int, upper bool) string {
	if num < 1 {
		num = 1
	}

	base := 'a'
	if upper {
		base = 'A'
	}

	letters := make([]rune, 0, 4)
	for num > 0 {
		num--
		letters = append([]rune{base + rune(num%26)}, letters...)
		num /= 26
	}
	return string(letters)
}
