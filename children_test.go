package dom_test

import (
	"testing"

	dom "github.com/npillmayer/w3c-dom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestChildrenFiltersElements(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kinds []html.NodeType
	}{
		{"empty", nil},
		{"text only", []html.NodeType{html.TextNode, html.TextNode}},
		{"elements only", []html.NodeType{html.ElementNode, html.ElementNode}},
		{"mixed", []html.NodeType{html.TextNode, html.ElementNode, html.TextNode, html.ElementNode, html.TextNode}},
		{"comments", []html.NodeType{html.CommentNode, html.ElementNode, html.CommentNode}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "div"})
			var want []*html.Node
			for _, kind := range tc.kinds {
				h := &html.Node{Type: kind, Data: "span"}
				parent.AddChild(styledtree.NewNodeForHTMLNode(h))
				if kind == html.ElementNode {
					want = append(want, h)
				}
			}
			node := dom.NodeFromStyledNode(parent.Payload)
			list := node.Children()
			if list == nil || list.Length() != len(want) {
				t.Fatalf("Children = %v, want length %d", list, len(want))
			}
			for i, h := range want {
				got, ok := list.Item(i).(*dom.W3CNode)
				if !ok || got == nil || got.StyNode == nil {
					t.Fatalf("Item(%d) is nil or invalid", i)
				}
				if got.HTMLNode() != h {
					t.Errorf("Item(%d) has wrong element/order", i)
				}
			}
			for _, i := range []int{-1, len(want), len(want) + 1} {
				if got := list.Item(i); got != nil {
					t.Errorf("Item(%d) = %#v, want nil", i, got)
				}
			}
			if got := node.ChildNodes().Length(); got != len(tc.kinds) {
				t.Errorf("ChildNodes length = %d", got)
			}
		})
	}
	var node *dom.W3CNode
	if node.Children() != nil {
		t.Fatal("nil receiver must return nil")
	}
}
