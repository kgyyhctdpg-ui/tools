package markdown

import (
	"html"
	"strings"
)

func joinMarkdownBlocks(blocks []string) string {
	filtered := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block != "" {
			filtered = append(filtered, block)
		}
	}
	return strings.Join(filtered, "\n\n")
}

func markdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	columns := 0
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return ""
	}

	var out strings.Builder
	writeRow := func(row []string) {
		out.WriteString("|")
		for column := 0; column < columns; column++ {
			value := ""
			if column < len(row) {
				value = markdownTableCell(row[column])
			}
			out.WriteString(" ")
			out.WriteString(value)
			out.WriteString(" |")
		}
		out.WriteString("\n")
	}

	writeRow(rows[0])
	out.WriteString("|")
	for column := 0; column < columns; column++ {
		out.WriteString(" --- |")
	}
	out.WriteString("\n")
	for _, row := range rows[1:] {
		writeRow(row)
	}
	return strings.TrimSpace(out.String())
}

func htmlTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	columns := 0
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("<table>\n<thead>\n<tr>")
	for column := 0; column < columns; column++ {
		value := ""
		if column < len(rows[0]) {
			value = html.EscapeString(rows[0][column])
		}
		out.WriteString("<th>" + value + "</th>")
	}
	out.WriteString("</tr>\n</thead>")
	if len(rows) > 1 {
		out.WriteString("\n<tbody>")
		for _, row := range rows[1:] {
			out.WriteString("\n<tr>")
			for column := 0; column < columns; column++ {
				value := ""
				if column < len(row) {
					value = html.EscapeString(row[column])
				}
				out.WriteString("<td>" + value + "</td>")
			}
			out.WriteString("</tr>")
		}
		out.WriteString("\n</tbody>")
	}
	out.WriteString("\n</table>")
	return out.String()
}

func markdownTableCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", "<br>")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}

func htmlEscapeText(value string) string {
	return html.EscapeString(value)
}

func markdownImage(altText, imageURL string) string {
	altText = strings.ReplaceAll(altText, "\\", "\\\\")
	altText = strings.ReplaceAll(altText, "]", "\\]")
	if strings.ContainsAny(imageURL, " ()") {
		return "![" + altText + "](<" + strings.ReplaceAll(imageURL, ">", "%3E") + ">)"
	}
	return "![" + altText + "](" + imageURL + ")"
}

func fencedCode(language, content string) string {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	return fence + language + "\n" + strings.TrimSpace(content) + "\n" + fence
}
