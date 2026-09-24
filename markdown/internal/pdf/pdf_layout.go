package pdf

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/88250/lute"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

// The PDF engine reports a page as positioned text rectangles carrying a
// bounding box and font metrics. Turning those into Markdown runs in three
// stages:
//
//	glyphs -> lines    group the runs that share one visual line
//	lines  -> blocks   recognise tables, headings, lists and paragraphs
//	blocks -> markdown render each block
//
// Staging the output through semantic HTML is what lets lute do the final
// serialization. lute reads <h1>, <table>, <strong> and <li>, but it never looks
// at style/top/left, so the structure has to be reconstructed here first: fed
// the engine's raw positioned HTML it would flatten every table into a run of
// one-cell paragraphs.
//
// Only the first stage looks at raw geometry, and it looks at the vertical axis
// alone: a line keeps every run it contains, however wide the gaps between them.
//
// That split is deliberate. A horizontal gap means nothing on its own. A letter
// spaced title such as 施 工 图 设 计 is one line with 38pt between glyphs,
// while a form's two columns at 9pt sit barely 9pt apart; no threshold tells
// those apart. What tells them apart is the surrounding lines, because a table
// repeats its column positions on every row and a title does not. That decision
// therefore belongs to the block stage, which can see the neighbouring lines.
// Splitting on the gap any earlier is what turns a title into one paragraph per
// glyph.

// pdfInlineSpan is a run of characters sharing one style and one position.
type pdfInlineSpan struct {
	Text     string
	Bold     bool
	Italic   bool
	FontSize float64
	Left     float64
	Right    float64
}

// pdfTextLine is one visual line: every run that shares a vertical band.
//
// Spans are ordered left to right and the gaps between them are preserved, so
// the block stage can still tell a column break from letter spacing.
type pdfTextLine struct {
	Top      float64
	Bottom   float64
	Left     float64
	Right    float64
	FontSize float64
	Spans    []pdfInlineSpan
}

// Text returns the line as plain text, with its runs concatenated.
func (line pdfTextLine) Text() string {
	var builder strings.Builder
	for _, span := range line.Spans {
		builder.WriteString(collapsePDFWhitespace(span.Text))
	}
	return strings.TrimSpace(builder.String())
}

// HTML returns the line as semantic HTML with inline emphasis preserved.
func (line pdfTextLine) HTML() string {
	return renderPDFInlineHTML(line.Spans)
}

// Height is the vertical extent of the line's ink.
func (line pdfTextLine) Height() float64 {
	return math.Max(0, line.Bottom-line.Top)
}

// pdfEmbeddedImage is one embedded image extracted from the page.
type pdfEmbeddedImage struct {
	MIMEType string
	Data     []byte
	AltText  string
}

// pdfBlock is a finished piece of Markdown together with the number of visual
// lines it consumed, which is what lets images be re-inserted at the right
// place.
// pdfBlock is one rendered piece of a page. A raw block already carries its
// final text, which is the case for tables because their HTML/Markdown choice
// belongs to the caller. Every other block holds semantic HTML for lute to
// serialize.
type pdfBlock struct {
	html  string
	raw   bool
	lines int
}

// gapFloor is the horizontal gap below which two runs always count as word
// spacing, whatever the font size. It keeps small body text from being split on
// the ordinary side bearings between glyphs.
const gapFloor = 8.0

// gapScale lets the column break threshold grow with the font size, because the
// same absolute gap means very different things at 9pt and at 55pt.
const gapScale = 0.8

// pdfGapLimit is the widest gap that still reads as word spacing at this size.
func pdfGapLimit(size float64) float64 {
	if scaled := size * gapScale; scaled > gapFloor {
		return scaled
	}
	return gapFloor
}

// ---------------------------------------------------------------------------
// stage one: glyphs to lines
// ---------------------------------------------------------------------------

// lineTolerance is how far two runs may sit apart vertically and still share a
// line, as a fraction of the font size.
//
// Both extremes have to be covered. A comma hangs below the baseline and a lone
// stroke such as 一 or 二 floats above it, so their boxes sit up to half an em
// away from their neighbours'; while a large title and the small line beneath it
// can be only a third of an em apart and are still two separate lines. Measuring
// centre to centre, each run against its own size, is what satisfies both.
const lineTolerance = 0.45

// buildPDFTextLines groups the positioned rectangles of one page into visual
// lines.
//
// PDF coordinates put the origin at the bottom left while everything downstream
// expects a distance from the top of the page, so the vertical axis is flipped
// here.
//
// Rectangles are clustered by the vertical centre of their box, and a cluster
// becomes one line whatever the horizontal gaps between its runs. Membership
// does not depend on the order the engine emitted them: a generator that draws a
// form in two passes emits every label first and every value afterwards, and a
// drawing sheet scatters the fragments of one title around the sheet.
func buildPDFTextLines(rects []pdfTextRect, pageHeight float64) []pdfTextLine {
	clusters := make([]pdfRectCluster, 0, len(rects))
	for _, rect := range rects {
		text := collapsePDFWhitespace(rect.Text)
		if strings.TrimSpace(text) == "" {
			continue
		}
		size := rect.FontSize
		if size <= 0 {
			size = rect.Top - rect.Bottom
		}
		if size <= 0 {
			size = rect.Right - rect.Left
		}
		top, bottom := pageHeight-rect.Top, pageHeight-rect.Bottom
		if bottom < top {
			top, bottom = bottom, top
		}
		span := pdfInlineSpan{
			Text:     text,
			Bold:     rect.bold(),
			Italic:   rect.italic(),
			FontSize: size,
			Left:     rect.Left,
			Right:    rect.Right,
		}
		center := (top + bottom) / 2
		index := pdfClusterForCenter(clusters, center, size)
		if index < 0 {
			clusters = append(clusters, pdfRectCluster{
				center: center,
				size:   size,
				top:    top,
				bottom: bottom,
				left:   rect.Left,
				right:  rect.Right,
				spans:  []pdfInlineSpan{span},
			})
			continue
		}
		cluster := &clusters[index]
		cluster.spans = append(cluster.spans, span)
		if size > cluster.size {
			cluster.size = size
		}
		cluster.top = math.Min(cluster.top, top)
		cluster.bottom = math.Max(cluster.bottom, bottom)
		cluster.left = math.Min(cluster.left, rect.Left)
		cluster.right = math.Max(cluster.right, rect.Right)
	}

	lines := make([]pdfTextLine, 0, len(clusters))
	for _, cluster := range clusters {
		line := pdfTextLine{
			Top:    cluster.top,
			Bottom: cluster.bottom,
			Left:   cluster.left,
			Right:  cluster.right,
			Spans:  mergePDFInlineSpans(cluster.spans),
		}
		line.FontSize = pdfDominantSize(line.Spans)
		if text := line.Text(); text == "" {
			continue
		}
		lines = append(lines, line)
	}
	sort.SliceStable(lines, func(left, right int) bool {
		if lines[left].Top != lines[right].Top {
			return lines[left].Top < lines[right].Top
		}
		return lines[left].Left < lines[right].Left
	})
	return lines
}

// pdfRectCluster collects the rectangles that share one visual line.
//
// center is the vertical middle of the first rectangle that started the cluster.
// Later rectangles are matched against it and never against the last one added,
// so a cluster cannot creep down the page.
type pdfRectCluster struct {
	center float64
	// size is the largest font size in the cluster. A run set at a very
	// different size never joins, however the boxes overlap: a 20pt title and a
	// 9pt field label can be centred 4pt apart and still be two lines.
	size   float64
	top    float64
	bottom float64
	left   float64
	right  float64
	spans  []pdfInlineSpan
}

// pdfClusterForCenter returns the cluster a centre belongs to, or -1 for a new
// one. The tolerance follows the size of the run being placed.
func pdfClusterForCenter(clusters []pdfRectCluster, center, size float64) int {
	best, bestDelta := -1, 0.0
	for index := range clusters {
		tolerance := size * lineTolerance
		if !pdfSizesAreComparable(clusters[index].size, size) {
			// A run set at a very different size is usually a different element
			// that merely overlaps -- a field label beside a large title. It is
			// taken as part of the line only when it sits almost exactly on it,
			// which is how a small bullet glyph joins the text it precedes.
			tolerance = size * lineMixedTolerance
		}
		if tolerance < 1 {
			tolerance = 1
		}
		delta := math.Abs(center - clusters[index].center)
		if delta > tolerance {
			continue
		}
		if best < 0 || delta < bestDelta {
			best, bestDelta = index, delta
		}
	}
	return best
}

// lineMixedTolerance is the tighter vertical tolerance applied when a run's font
// size is far from the line's, as a fraction of the run's own size.
const lineMixedTolerance = 0.2

// lineSizeRatio is how far apart two font sizes may be and still belong to one
// line. Text on a line is set at one size, or at sizes a reader would call the
// same; anything wider apart is a different element that merely overlaps.
const lineSizeRatio = 1.5

// pdfSizesAreComparable reports whether two font sizes could share a line.
func pdfSizesAreComparable(left, right float64) bool {
	if left <= 0 || right <= 0 {
		return true
	}
	if left < right {
		left, right = right, left
	}
	return left <= right*lineSizeRatio
}

// mergePDFInlineSpans orders a line's runs from left to right and joins the
// neighbours that share one style.
func mergePDFInlineSpans(spans []pdfInlineSpan) []pdfInlineSpan {
	ordered := append([]pdfInlineSpan(nil), spans...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return ordered[left].Left < ordered[right].Left
	})
	merged := make([]pdfInlineSpan, 0, len(ordered))
	for _, span := range ordered {
		if span.Text == "" {
			continue
		}
		if len(merged) > 0 {
			last := &merged[len(merged)-1]
			// Runs are joined only while they touch. A wide gap has to survive
			// as a separate span, because the block stage needs it to tell a
			// table column from letter spacing.
			gap := pdfGapLimit(math.Max(last.FontSize, span.FontSize))
			if last.Bold == span.Bold && last.Italic == span.Italic &&
				last.FontSize == span.FontSize && span.Left-last.Right <= gap {
				last.Text += span.Text
				last.Right = math.Max(last.Right, span.Right)
				continue
			}
		}
		merged = append(merged, span)
	}
	return merged
}

// pdfDominantSize returns the size most of a line's characters use, which is the
// size a reader would call the size of that line.
func pdfDominantSize(spans []pdfInlineSpan) float64 {
	weights := make(map[int]int)
	for _, span := range spans {
		if span.FontSize <= 0 {
			continue
		}
		weights[int(math.Round(span.FontSize*10))] += utf8.RuneCountInString(span.Text)
	}
	best, bestWeight := 0, 0
	for size, weight := range weights {
		if weight > bestWeight || (weight == bestWeight && best != 0 && size < best) {
			best, bestWeight = size, weight
		}
	}
	if best > 0 {
		return float64(best) / 10
	}
	for _, span := range spans {
		if span.FontSize > 0 {
			return span.FontSize
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// stage two: lines to blocks
// ---------------------------------------------------------------------------

// pdfCell is one horizontal cell of a line: the runs between two wide gaps.
type pdfCell struct {
	Text     string
	Left     float64
	Right    float64
	FontSize float64
}

// empty reports whether the cell holds no text, which is how a table column
// given over to a picture shows up.
func (cell pdfCell) empty() bool {
	return strings.TrimSpace(cell.Text) == ""
}

// pdfLineCells proposes the cells a line's gaps imply.
//
// This is only a proposal. Whether the gaps really are columns is decided by
// pdfDetectTable, which can compare the line against its neighbours.
func pdfLineCells(line pdfTextLine) []pdfCell {
	cells := make([]pdfCell, 0, len(line.Spans))
	current := pdfCell{}
	for _, span := range line.Spans {
		text := collapsePDFWhitespace(span.Text)
		if strings.TrimSpace(text) == "" {
			continue
		}
		if current.empty() {
			current = pdfCell{Text: text, Left: span.Left, Right: span.Right, FontSize: span.FontSize}
			continue
		}
		if span.Left-current.Right > pdfGapLimit(math.Max(current.FontSize, span.FontSize)) {
			cells = append(cells, current)
			current = pdfCell{Text: text, Left: span.Left, Right: span.Right, FontSize: span.FontSize}
			continue
		}
		current.Text = pdfJoinRuns(current.Text, strings.TrimSpace(text))
		current.Right = math.Max(current.Right, span.Right)
		current.FontSize = math.Max(current.FontSize, span.FontSize)
	}
	if !current.empty() {
		cells = append(cells, current)
	}
	return cells
}

// buildPDFLineBlocks reconstructs the table, heading and paragraph structure of
// one page.
//
// imageOrdinals lists, for every image on the page, how many lines preceded it,
// which is how an image is tied back to the row it belongs to.
//
// Tables are looked for first, because a table row is otherwise
// indistinguishable from a line of text with unusual spacing. Everything that is
// not a table, a heading or a list item is reflowed into paragraphs.
func buildPDFLineBlocks(lines []pdfTextLine, imageOrdinals []int, options core.PDFOptions, tableFormat core.TableFormat) []pdfBlock {
	if len(lines) == 0 {
		return nil
	}
	bodySize := pdfBodyFontSize(lines)
	cells := make([][]pdfCell, len(lines))
	for index, line := range lines {
		cells[index] = pdfLineCells(line)
	}
	images := pdfImagesPerLine(lines, imageOrdinals)

	blocks := make([]pdfBlock, 0, len(lines))
	index := 0
	for index < len(lines) {
		if rows, used, ok := pdfDetectTable(cells, images, index, options); ok {
			// A table is left raw: its HTML/Markdown choice is the caller's,
			// and Markdown tables cannot carry merged cells anyway.
			blocks = append(blocks, pdfBlock{html: pdfRenderTable(rows, tableFormat), raw: true, lines: used})
			index += used
			continue
		}
		if level, ok := pdfHeadingLevel(lines[index], cells[index], bodySize, options); ok {
			tag := "h" + strconv.Itoa(level)
			blocks = append(blocks, pdfBlock{
				html:  "<" + tag + ">" + core.HTMLEscapeText(lines[index].Text()) + "</" + tag + ">",
				lines: 1,
			})
			index++
			continue
		}
		start := index
		for index < len(lines) {
			if _, _, ok := pdfDetectTable(cells, images, index, options); ok {
				break
			}
			if _, ok := pdfHeadingLevel(lines[index], cells[index], bodySize, options); ok {
				break
			}
			index++
		}
		blocks = append(blocks, reflowPDFLines(lines[start:index], options)...)
	}
	return blocks
}

// pdfImagesPerLine counts how many images start at each line boundary.
func pdfImagesPerLine(lines []pdfTextLine, imageOrdinals []int) []int {
	counts := make([]int, len(lines))
	for _, ordinal := range imageOrdinals {
		if ordinal < 0 {
			ordinal = 0
		}
		if ordinal >= len(lines) {
			continue
		}
		counts[ordinal]++
	}
	return counts
}

// pdfDetectTable reports whether the lines starting at from form a table, and
// how many lines it consumed.
//
// A table is a run of consecutive lines that repeat the same column positions.
// Requiring that repetition is what keeps a letter spaced title -- a single line
// whose gaps look exactly like columns -- from being read as a table, and
// requiring the cells to stay short is what keeps two columns of prose from
// being read as one.
func pdfDetectTable(cells [][]pdfCell, images []int, from int, options core.PDFOptions) ([][]pdfCell, int, bool) {
	if options.DisableTableReconstruction || from >= len(cells) {
		return nil, 0, false
	}
	header := cells[from]
	if len(header) < core.PDFMinTableColumns {
		return nil, 0, false
	}
	columns := pdfCellLefts(header)
	if !pdfColumnsAreDistinct(columns) {
		return nil, 0, false
	}
	tolerance := math.Max(6, pdfMedianCellSize(header)*1.5)
	rows := [][]pdfCell{header}
	for index := from + 1; index < len(cells); index++ {
		row, filled, ok := pdfAlignRow(cells[index], columns, tolerance)
		if !ok {
			break
		}
		if filled < len(columns) && pdfImageCount(images, index) == 0 {
			// A short row only continues the table when its missing cell is a
			// picture, as in a signature block.
			break
		}
		rows = append(rows, row)
	}
	if len(rows) < core.PDFMinTableRows || !pdfRowsAreShort(rows) {
		return nil, 0, false
	}
	return rows, len(rows), true
}

func pdfImageCount(counts []int, index int) int {
	if index < 0 || index >= len(counts) {
		return 0
	}
	return counts[index]
}

func pdfCellLefts(cells []pdfCell) []float64 {
	lefts := make([]float64, 0, len(cells))
	for _, cell := range cells {
		if cell.empty() {
			continue
		}
		lefts = append(lefts, cell.Left)
	}
	return lefts
}

// pdfColumnsAreDistinct reports whether the column starts are far enough apart
// to be separate columns rather than one ragged run.
func pdfColumnsAreDistinct(lefts []float64) bool {
	for index := 1; index < len(lefts); index++ {
		if lefts[index]-lefts[index-1] < gapFloor {
			return false
		}
	}
	return true
}

// pdfAlignRow fits a line's cells onto the table's columns. Cells are matched to
// the nearest column rather than to an exact offset, so a right aligned number
// still lands in its column. It reports how many slots ended up filled.
func pdfAlignRow(row []pdfCell, columns []float64, tolerance float64) ([]pdfCell, int, bool) {
	aligned := make([]pdfCell, len(columns))
	filled := 0
	for _, cell := range row {
		if cell.empty() {
			continue
		}
		slot, best := -1, tolerance
		for index, left := range columns {
			if delta := math.Abs(cell.Left - left); delta <= best {
				slot, best = index, delta
			}
		}
		if slot < 0 {
			return nil, 0, false
		}
		if aligned[slot].empty() {
			filled++
		}
		aligned[slot] = cell
	}
	return aligned, filled, true
}

// pdfRowsAreShort rejects a grid whose cells read like sentences, which is what
// two columns of prose produce.
func pdfRowsAreShort(rows [][]pdfCell) bool {
	lengths := make([]float64, 0, len(rows)*2)
	maximum := 0
	for _, row := range rows {
		for _, cell := range row {
			if cell.empty() {
				continue
			}
			count := utf8.RuneCountInString(strings.TrimSpace(cell.Text))
			if count > maximum {
				maximum = count
			}
			lengths = append(lengths, float64(count))
		}
	}
	if len(lengths) == 0 {
		return false
	}
	return maximum <= core.PDFMaxTableCellRunes && medianPDFFloat(lengths) <= core.PDFMaxTableMedian
}

func pdfMedianCellSize(cells []pdfCell) float64 {
	sizes := make([]float64, 0, len(cells))
	for _, cell := range cells {
		if cell.FontSize > 0 {
			sizes = append(sizes, cell.FontSize)
		}
	}
	return medianPDFFloat(sizes)
}

// pdfRenderTable writes a grid as a Markdown or HTML table.
func pdfRenderTable(rows [][]pdfCell, tableFormat core.TableFormat) string {
	grid := make([][]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, 0, len(row))
		for _, cell := range row {
			values = append(values, strings.TrimSpace(cell.Text))
		}
		grid = append(grid, values)
	}
	if tableFormat == core.TableFormatHTML {
		return core.HTMLTable(grid)
	}
	return core.MarkdownTable(grid)
}

// pdfHeadingLevel classifies a line as a heading by how far its font size stands
// out from the body size.
func pdfHeadingLevel(line pdfTextLine, cells []pdfCell, bodySize float64, options core.PDFOptions) (int, bool) {
	if options.DisableHeadingDetection || bodySize <= 0 || line.FontSize <= bodySize+0.4 {
		return 0, false
	}
	// A line split into columns is a table row, not a heading.
	if len(cells) != 1 {
		return 0, false
	}
	text := line.Text()
	if text == "" || utf8.RuneCountInString(text) > 100 || !pdfLooksLikeHeading(text) {
		return 0, false
	}
	switch ratio := line.FontSize / bodySize; {
	case ratio >= 1.6:
		return 1, true
	case ratio >= 1.3:
		return 2, true
	case ratio >= 1.15:
		return 3, true
	}
	return 0, false
}

// pdfLooksLikeHeading rejects text that is plainly not a heading however large it
// is set: page numbers, table ordinals, bullet glyphs.
func pdfLooksLikeHeading(text string) bool {
	letters := 0
	for _, character := range text {
		if unicode.IsLetter(character) {
			letters++
		}
	}
	return letters >= 2
}

// ---------------------------------------------------------------------------
// paragraph reflow
// ---------------------------------------------------------------------------

// reflowPDFLines merges wrapped visual lines into paragraphs and lists.
func reflowPDFLines(lines []pdfTextLine, options core.PDFOptions) []pdfBlock {
	if len(lines) == 0 {
		return nil
	}
	step := medianPDFTopStep(lines)
	blocks := make([]pdfBlock, 0, len(lines))
	type listEntry struct {
		marker string
		html   string
	}
	var (
		paragraph      strings.Builder
		paragraphLines int
		previous       *pdfTextLine
		listEntries    []listEntry
		listOrdered    bool
	)
	flushParagraph := func() {
		if text := strings.TrimSpace(paragraph.String()); text != "" {
			blocks = append(blocks, pdfBlock{html: "<p>" + text + "</p>", lines: paragraphLines})
		}
		paragraph.Reset()
		paragraphLines = 0
		previous = nil
	}
	flushList := func() {
		if len(listEntries) == 0 {
			return
		}
		var builder strings.Builder
		if listOrdered {
			builder.WriteString("<ol>")
		} else {
			builder.WriteString("<ul>")
		}
		for _, entry := range listEntries {
			// lute takes the marker from data-marker rather than counting, which
			// is what keeps the numbering the page used.
			builder.WriteString(`<li data-marker="` + entry.marker + `">` + entry.html + `</li>`)
		}
		if listOrdered {
			builder.WriteString("</ol>")
		} else {
			builder.WriteString("</ul>")
		}
		blocks = append(blocks, pdfBlock{html: builder.String(), lines: len(listEntries)})
		listEntries = nil
	}
	for index := range lines {
		line := lines[index]
		if item, number, ordered, ok := normalizePDFListMarker(line.Text()); ok {
			flushParagraph()
			if len(listEntries) > 0 && ordered != listOrdered {
				flushList()
			}
			marker := "-"
			if ordered {
				if number < 1 {
					number = 1
				}
				marker = strconv.Itoa(number) + "."
			}
			listOrdered = ordered
			listEntries = append(listEntries, listEntry{marker: marker, html: core.HTMLEscapeText(item)})
			continue
		}
		flushList()
		text := strings.TrimSpace(line.HTML())
		if text == "" {
			continue
		}
		continuation := false
		if !options.DisableParagraphReflow && previous != nil && paragraph.Len() > 0 {
			continuation = canJoinPDFLines(*previous, line, step)
		}
		if continuation {
			writePDFJoin(&paragraph, text)
		} else {
			flushParagraph()
			paragraph.WriteString(text)
		}
		paragraphLines++
		copied := line
		previous = &copied
	}
	flushParagraph()
	flushList()
	return blocks
}

// canJoinPDFLines decides whether a line continues the paragraph above it. A
// paragraph's lines are one leading apart; a larger step starts a new one.
func canJoinPDFLines(previous, line pdfTextLine, step float64) bool {
	delta := line.Top - previous.Top
	if delta <= 0 {
		return false
	}
	if step <= 0 {
		// Without a reliable leading, only an exact same-column continuation
		// qualifies.
		return math.Abs(line.Left-previous.Left) <= 2
	}
	if delta > step*1.7 || delta < step*0.5 {
		return false
	}
	tolerance := math.Max(6, 2.5*lineSize(line))
	return math.Abs(line.Left-previous.Left) <= tolerance
}

// pdfJoinRuns appends one run of text to another, following the spacing rules of
// the script in use: CJK runs are closed up, Latin words are separated, and a
// hyphenated word broken across two lines loses its hyphen.
func pdfJoinRuns(current, next string) string {
	if current == "" || next == "" {
		return current + next
	}
	last, _ := utf8.DecodeLastRuneInString(current)
	first, _ := utf8.DecodeRuneInString(next)
	switch {
	case unicode.IsSpace(last) || unicode.IsSpace(first):
		return current + next
	case last == '-' && first >= 'a' && first <= 'z':
		return strings.TrimRight(current, "-") + next
	case isCJKRune(last) && isCJKRune(first):
		return current + next
	}
	return current + " " + next
}

// writePDFJoin appends a continuation to the paragraph text.
func writePDFJoin(builder *strings.Builder, next string) {
	joined := pdfJoinRuns(builder.String(), next)
	builder.Reset()
	builder.WriteString(joined)
}

// ---------------------------------------------------------------------------
// measurements
// ---------------------------------------------------------------------------

// pdfBodyFontSize estimates the size most of the page's text is set in.
func pdfBodyFontSize(lines []pdfTextLine) float64 {
	weights := make(map[int]int)
	for _, line := range lines {
		if line.FontSize <= 0 {
			continue
		}
		weights[int(math.Round(line.FontSize*10))] += utf8.RuneCountInString(line.Text())
	}
	best, bestWeight := 0, -1
	for size, weight := range weights {
		if weight > bestWeight || (weight == bestWeight && size < best) {
			best, bestWeight = size, weight
		}
	}
	if bestWeight <= 0 {
		return 0
	}
	return float64(best) / 10
}

// medianPDFTopStep estimates the leading between two lines of one paragraph.
//
// A paragraph gap is much larger than the leading, so the smallest recurring step
// is the one that matters; steps far below it come from superscripts and
// footnote markers and are ignored.
func medianPDFTopStep(lines []pdfTextLine) float64 {
	steps := make([]float64, 0, len(lines))
	for index := 1; index < len(lines); index++ {
		delta := lines[index].Top - lines[index-1].Top
		if delta <= 0 || delta > 5*lineSize(lines[index-1]) {
			continue
		}
		steps = append(steps, delta)
	}
	if len(steps) == 0 {
		return 0
	}
	floor := medianPDFFloat(steps) * 0.5
	smallest := 0.0
	for _, step := range steps {
		if step < floor {
			continue
		}
		if smallest == 0 || step < smallest {
			smallest = step
		}
	}
	if smallest == 0 {
		return medianPDFFloat(steps)
	}
	return smallest
}

func lineSize(line pdfTextLine) float64 {
	if line.FontSize > 0 {
		return line.FontSize
	}
	if height := line.Height(); height > 0 {
		return height
	}
	return 10
}

func medianPDFFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

// ---------------------------------------------------------------------------
// text helpers
// ---------------------------------------------------------------------------

// collapsePDFWhitespace flattens the whitespace inside a run while remembering
// whether the run was padded, which is what keeps words apart when runs are
// concatenated.
func collapsePDFWhitespace(value string) string {
	core := strings.Join(strings.Fields(value), " ")
	if core == "" {
		if value == "" {
			return ""
		}
		return " "
	}
	if strings.TrimLeftFunc(value, unicode.IsSpace) != value {
		core = " " + core
	}
	if strings.TrimRightFunc(value, unicode.IsSpace) != value {
		core += " "
	}
	return core
}

// renderPDFInlineHTML writes the runs of a line as semantic HTML.
//
// Emphasis has to be spelled out as markup rather than as a font attribute:
// lute maps <strong>/<em> to Markdown but ignores font-size and font-weight
// entirely, so a bold run only survives the round trip as a tag.
func renderPDFInlineHTML(spans []pdfInlineSpan) string {
	var builder strings.Builder
	pendingSpace := false
	for _, span := range spans {
		text := collapsePDFWhitespace(span.Text)
		if text == "" {
			continue
		}
		leading := strings.HasPrefix(text, " ")
		trailing := strings.HasSuffix(text, " ")
		content := strings.TrimSpace(text)
		if content == "" {
			pendingSpace = true
			continue
		}
		if (leading || pendingSpace) && builder.Len() > 0 {
			builder.WriteString(" ")
		}
		pendingSpace = false
		escaped := core.HTMLEscapeText(content)
		switch {
		case span.Bold && span.Italic:
			builder.WriteString("<strong><em>" + escaped + "</em></strong>")
		case span.Bold:
			builder.WriteString("<strong>" + escaped + "</strong>")
		case span.Italic:
			builder.WriteString("<em>" + escaped + "</em>")
		default:
			builder.WriteString(escaped)
		}
		if trailing {
			pendingSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func isCJKRune(character rune) bool {
	switch {
	case character >= 0x4E00 && character <= 0x9FFF:
		return true
	case character >= 0x3400 && character <= 0x4DBF:
		return true
	case character >= 0x3000 && character <= 0x303F:
		return true
	case character >= 0x3040 && character <= 0x30FF:
		return true
	case character >= 0xAC00 && character <= 0xD7AF:
		return true
	case character >= 0xF900 && character <= 0xFAFF:
		return true
	case character >= 0xFF00 && character <= 0xFFEF:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// list markers
// ---------------------------------------------------------------------------

var (
	pdfOrderedMarkerSpace = regexp.MustCompile(`^(\d{1,3})[.)）]\s+`)
	pdfOrderedMarkerCJK   = regexp.MustCompile(`^(\d{1,3})\s*、\s*`)
	pdfParenOrderedMarker = regexp.MustCompile(`^[（(]\s*(\d{1,3})\s*[)）]\s*`)
	pdfBulletMarker       = regexp.MustCompile(`^[-*+•·▪◦‣∙※]\s+`)
)

// pdfListBlockMaxRunes stops a long paragraph that merely starts with a number
// and a dot from becoming a list item.
const pdfListBlockMaxRunes = 240

// normalizePDFListMarker recognises a list marker and returns the item text
// together with the list number, so the caller can rebuild the list as HTML
// without lute having to guess the numbering.
func normalizePDFListMarker(text string) (item string, number int, ordered bool, ok bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > pdfListBlockMaxRunes {
		return "", 0, false, false
	}
	for _, pattern := range []*regexp.Regexp{pdfOrderedMarkerSpace, pdfOrderedMarkerCJK, pdfParenOrderedMarker} {
		match := pattern.FindStringSubmatchIndex(trimmed)
		if match == nil {
			continue
		}
		rest := strings.TrimSpace(trimmed[match[1]:])
		if rest == "" {
			return "", 0, false, false
		}
		start, _ := strconv.Atoi(trimmed[match[2]:match[3]])
		return rest, start, true, true
	}
	if match := pdfBulletMarker.FindStringIndex(trimmed); match != nil {
		rest := strings.TrimSpace(trimmed[match[1]:])
		if rest == "" {
			return "", 0, false, false
		}
		return rest, 0, false, true
	}
	return "", 0, false, false
}

// renderPDFBlock turns one layout block into Markdown. Raw blocks already carry
// their final text; flow blocks are semantic HTML that lute serializes, which is
// what gives the output a real Markdown escaper.
func renderPDFBlock(engine *lute.Lute, block pdfBlock) (string, error) {
	if block.raw {
		return block.html, nil
	}
	return engine.HTML2Markdown(block.html)
}

// joinPDFBlocks renders a block list to Markdown.
func joinPDFBlocks(blocks []pdfBlock) (string, error) {
	engine := lute.New()
	rendered := make([]string, 0, len(blocks))
	for _, block := range blocks {
		text, err := renderPDFBlock(engine, block)
		if err != nil {
			return "", err
		}
		if text = strings.TrimSpace(text); text != "" {
			rendered = append(rendered, text)
		}
	}
	return core.JoinBlocks(rendered), nil
}
