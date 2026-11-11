package css_test

import (
	"github.com/npillmayer/tree"
	dom "github.com/npillmayer/w3c-dom"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestInheritanceAndDefaults(t *testing.T) {
	h, err := html.Parse(strings.NewReader(`<html style="width:inherit"><body><div id="parent" style="color:red;width:50px;position:absolute;overflow-wrap:break-word;hyphens:auto"><span id="plain"></span><span id="inherit" style="width:inherit;color:INHERIT;position:inherit"></span><span id="initial" style="color:initial;width:initial;display:initial;overflow-wrap:initial"><b id="grandchild"></b></span><span id="unset" style="color:unset;width:unset"></span><span id="custom" style="custom:initial"></span></div></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	engine := cssom.NewCSSOM([]style.KeyValue{{Key: "color", Value: "purple"}, {Key: "width", Value: "42px"}, {Key: "custom", Value: "fallback"}})
	root, err := engine.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]*styledtree.StyNode{}
	var visit func(*tree.Node[*styledtree.StyNode])
	visit = func(n *tree.Node[*styledtree.StyNode]) {
		for _, a := range n.Payload.HTMLNode().Attr {
			if a.Key == "id" {
				nodes[a.Val] = n.Payload
			}
		}
		if n.Payload.HTMLNode().Data == "html" {
			nodes["html"] = n.Payload
		}
		for _, ch := range n.Children(true) {
			visit(ch)
		}
	}
	visit(root)
	for _, tc := range []struct{ id, key, want string }{
		{"plain", "color", "red"}, {"plain", "width", "42px"}, {"plain", "position", "static"},
		{"plain", "overflow-wrap", "break-word"}, {"plain", "hyphens", "auto"}, {"plain", "word-wrap", "break-word"},
		{"inherit", "color", "red"}, {"inherit", "width", "50px"}, {"inherit", "position", "absolute"},
		{"initial", "color", "default"}, {"initial", "width", "auto"}, {"initial", "display", "inline"},
		{"grandchild", "color", "default"}, {"grandchild", "overflow-wrap", "normal"},
		{"unset", "color", "red"}, {"unset", "width", "auto"},
		{"plain", "custom", "fallback"}, {"custom", "custom", ""}, {"plain", "unknown", ""},
		{"html", "width", "auto"},
	} {
		n := nodes[tc.id]
		if n == nil {
			t.Fatalf("missing %s", tc.id)
		}
		got, err := css.GetProperty(n, tc.key)
		if err != nil || string(got) != tc.want {
			t.Errorf("%s %s = %q, %v; want %q", tc.id, tc.key, got, err, tc.want)
		}
		if v := dom.NodeFromStyledNode(n).ComputedStyles().GetPropertyValue(tc.key); v != got {
			t.Error("DOM lookup differs")
		}
	}
	// Detached elements still receive initial/UA defaults; initial ignores tag display.
	n := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "div"}).Payload
	if got, _ := css.GetProperty(n, "display"); got != "block" {
		t.Errorf("UA display = %q", got)
	}
	local := style.NewPropertyMap()
	local.Add("display", "initial")
	n.SetStyles(local)
	if got, _ := css.GetProperty(n, "display"); got != "inline" {
		t.Errorf("initial display = %q", got)
	}
	if got, _ := css.GetProperty(root.Payload, "color"); got != "purple" {
		t.Errorf("UA color override = %q", got)
	}
}
