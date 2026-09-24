package docx

import (
	"testing"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

func TestOMMLComplexStructures(t *testing.T) {
	root, err := core.ParseXML([]byte(`<m:oMath xmlns:m="urn:math"><m:nary><m:naryPr><m:chr m:val="∑"/></m:naryPr><m:sub><m:r><m:t>i=1</m:t></m:r></m:sub><m:sup><m:r><m:t>n</m:t></m:r></m:sup><m:e><m:f><m:num><m:r><m:t>x</m:t></m:r></m:num><m:den><m:rad><m:e><m:r><m:t>y</m:t></m:r></m:e></m:rad></m:den></m:f></m:e></m:nary></m:oMath>`))
	if err != nil {
		t.Fatal(err)
	}
	latex := ommlToLatex(root.First("oMath"))
	want := `\sum_{i=1}^{n}\frac{x}{\sqrt{y}}`
	if latex != want {
		t.Fatalf("unexpected LaTeX:\nwant: %s\n got: %s", want, latex)
	}
}
