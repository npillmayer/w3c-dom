package css_test

import (
	"github.com/npillmayer/tree"
	dom "github.com/npillmayer/w3c-dom"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestHTMLDisplayResolution(t *testing.T) {
	h, err := html.Parse(strings.NewReader(`<div style="display:block"><a id="a">link</a><em id="em">em</em><code id="code">code</code><label id="label">label</label><span id="override" style="display:block">x</span><em id="inherit" style="display:inherit">x</em><div id="initial" style="display:initial">x</div><span id="unset" style="display:unset">x</span></div>`))
	if err != nil {
		t.Fatal(err)
	}
	engine := cssom.NewCSSOM(nil)
	root, err := engine.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"a": "inline", "em": "inline", "code": "inline", "label": "inline", "override": "block", "inherit": "block", "initial": "inline", "unset": "inline"}
	var visit func(*tree.Node[*styledtree.StyNode])
	visit = func(n *tree.Node[*styledtree.StyNode]) {
		for _, a := range n.Payload.HTMLNode().Attr {
			if a.Key == "id" {
				expected, ok := want[a.Val]
				if !ok {
					t.Fatalf("unexpected node %s", a.Val)
				}
				got := dom.NodeFromStyledNode(n.Payload).ComputedStyles().GetPropertyValue("display")
				if string(got) != expected {
					t.Errorf("%s=%q want %s", a.Val, got, expected)
				}
				delete(want, a.Val)
			}
		}
		for _, ch := range n.Children(true) {
			visit(ch)
		}
	}
	visit(root)
	if len(want) != 0 {
		t.Fatalf("missing nodes: %v", want)
	}
}
