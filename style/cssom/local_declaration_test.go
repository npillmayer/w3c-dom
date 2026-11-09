package cssom_test

import (
	"strings"
	"testing"

	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestExplicitDefaultRemainsLocal(t *testing.T) {
	input, err := html.Parse(strings.NewReader(`<div><span id="explicit" style="display:block;visibility:hidden">one</span><span id="sibling">two</span></div>`))
	if err != nil {
		t.Fatal(err)
	}
	sheet := cssom.NewCSSOM(nil)
	root, err := sheet.Style(input)
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
		for _, child := range n.Children(true) {
			visit(child)
		}
	}
	visit(root)
	explicit := nodes["explicit"]
	if explicit == nil {
		t.Fatal("missing styled node")
	}
	if value, ok := explicit.Styles().Property("display"); !ok || value != "block" {
		t.Fatalf("explicit display was lost: %q", value)
	}
	if value, ok := explicit.Styles().Property("visibility"); !ok || value != "hidden" {
		t.Fatalf("visibility lost: %q", value)
	}
	if _, ok := nodes["sibling"].Styles().Property("visibility"); ok {
		t.Fatal("sibling acquired local visibility")
	}
	defaults := root.Parent().Payload.Styles()
	if value, _ := defaults.Property("visibility"); value != "visible" {
		t.Fatalf("ancestor default mutated: %q", value)
	}
}
