package cssom_test

import (
	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestSubtreeStyles(t *testing.T) {
	h, err := html.Parse(strings.NewReader(`<div id="outer" style="width:99px"><p id="child"></p><div id="inner"><p id="deep"></p></div></div><p id="outside"></p>`))
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]*html.Node{}
	var scan func(*html.Node)
	scan = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "id" {
				nodes[a.Val] = n
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			scan(ch)
		}
	}
	scan(h)
	engine := cssom.NewCSSOM(nil)
	add := func(scope *html.Node, source string) {
		t.Helper()
		sheet, err := parser.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.AddStylesForScope(scope, douceuradapter.Wrap(sheet), cssom.Author); err != nil {
			t.Fatal(err)
		}
	}
	// Register inner first to verify the documented outer-to-inner scope order.
	add(nodes["inner"], `* {width:30px}`)
	add(nil, `* {width:10px;height:10px}`)
	add(nodes["outer"], `* {width:20px;height:20px} #deep {padding-top:7px} p {margin-top:8px!important}`)
	add(nodes["outer"], `* {height:25px}`)
	add(nodes["inner"], `p {padding-top:9px;margin-top:9px}`)
	root, err := engine.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*styledtree.StyNode{}
	var visit func(*tree.Node[*styledtree.StyNode])
	visit = func(n *tree.Node[*styledtree.StyNode]) {
		for _, a := range n.Payload.HTMLNode().Attr {
			if a.Key == "id" {
				found[a.Val] = n.Payload
			}
		}
		for _, ch := range n.Children(true) {
			visit(ch)
		}
	}
	visit(root)
	for id, want := range map[string]string{"outer": "99px", "child": "20px", "inner": "30px", "deep": "30px", "outside": "10px"} {
		n := found[id]
		if n == nil {
			t.Fatalf("missing %s", id)
		}
		if got, _ := n.Styles().Property("width"); string(got) != want {
			t.Errorf("%s width=%q want %s", id, got, want)
		}
	}
	for _, tc := range []struct{ id, key, want string }{
		{"deep", "height", "25px"}, {"outside", "height", "10px"},
		{"deep", "padding-top", "7px"}, {"deep", "margin-top", "8px"},
	} {
		if got, _ := found[tc.id].Styles().Property(tc.key); string(got) != tc.want {
			t.Errorf("%s %s=%q want %s", tc.id, tc.key, got, tc.want)
		}
	}
	if _, ok := found["outside"].Styles().Property("margin-top"); ok {
		t.Error("scoped declaration leaked outside subtree")
	}
}
