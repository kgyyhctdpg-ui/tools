package docx_test

import (
	"testing"

	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	"github.com/scoming-dev/tools/docx"
)

type testHTMLBlockPlugin struct {
	called bool
}

func (p *testHTMLBlockPlugin) Type() string {
	return "test"
}

func (p *testHTMLBlockPlugin) RenderHTML(_ *docx.DocxRenderer, _ string) error {
	p.called = true
	return nil
}

func TestHTMLBlockPluginDispatch(t *testing.T) {
	md := "\n\n<s-tag type=\"test\">\npayload\n</s-tag>\n\n"

	parseOptions := parse.NewOptions()
	parseOptions.HTMLTag2TextMark = true
	tree := parse.Parse("", []byte(md), parseOptions)
	renderOptions := render.NewOptions()

	plugin := &testHTMLBlockPlugin{}
	renderer := docx.NewDocxRenderer(tree, renderOptions, 0, docx.WithHTMLBlockPlugins(plugin))
	renderer.Render()

	if !plugin.called {
		t.Fatal("expected HTML block plugin to be called")
	}
}
