package pptx

import (
	"strings"
	"testing"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

func TestPPTXTablePreservesMergedRowsAndColumns(t *testing.T) {
	root, err := core.ParseXML([]byte(`<a:tbl xmlns:a="urn:drawingml">
<a:tr>
<a:tc gridSpan="2"><a:p><a:r><a:t>Columns</a:t></a:r></a:p></a:tc>
<a:tc hMerge="1"><a:p/></a:tc>
<a:tc rowSpan="2"><a:p><a:r><a:t>Rows</a:t></a:r></a:p></a:tc>
</a:tr>
<a:tr>
<a:tc><a:p><a:r><a:t>A</a:t></a:r></a:p></a:tc>
<a:tc><a:p><a:r><a:t>B</a:t></a:r></a:p></a:tc>
<a:tc vMerge="1"><a:p/></a:tc>
</a:tr>
</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}
	result := renderPPTXNode(root.First("tbl"), core.TableFormatMarkdown)
	if !strings.Contains(result, `<th colspan="2">Columns</th>`) || !strings.Contains(result, `<th rowspan="2">Rows</th>`) {
		t.Fatalf("PPTX merged cells were not preserved:\n%s", result)
	}
}
