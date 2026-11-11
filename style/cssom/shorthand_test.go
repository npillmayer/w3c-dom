package cssom_test

import (
	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"strings"
	"testing"

	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestShorthandsProduceLocalLonghands(t *testing.T) {
	h, err := html.Parse(strings.NewReader(`<p style="margin:1px 2px 3px 4px;border-radius:5px 6px 7px 8px">text</p>`))
	if err != nil {
		t.Fatal(err)
	}
	engine := cssom.NewCSSOM(nil)
	root, err := engine.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	var target *styledtree.StyNode
	var visit func(*tree.Node[*styledtree.StyNode])
	visit = func(n *tree.Node[*styledtree.StyNode]) {
		if n.Payload.HTMLNode().Data == "p" {
			target = n.Payload
		}
		for _, ch := range n.Children(true) {
			visit(ch)
		}
	}
	visit(root)
	if target == nil {
		t.Fatal("missing paragraph")
	}
	for key, want := range map[string]string{"margin-top": "1px", "margin-right": "2px", "margin-bottom": "3px", "margin-left": "4px", "border-top-left-radius": "5px", "border-top-right-radius": "6px", "border-bottom-right-radius": "7px", "border-bottom-left-radius": "8px"} {
		if got, ok := target.Styles().Property(key); !ok || string(got) != want {
			t.Errorf("%s = %q, %v", key, got, ok)
		}
	}
	for _, key := range []string{"margin", "border-radius", "border-top-left-style"} {
		if _, ok := target.Styles().Property(key); ok {
			t.Errorf("unexpected property %s", key)
		}
	}
}

func TestDuplicateShorthandSourceOrder(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"margin:1px;margin-left:2px;margin:3px", "3px"},
		{"margin:1px!important;margin-left:2px!important;margin:3px", "2px"},
		{"margin-left:2px;margin:1px;margin-left:3px", "3px"},
		{"margin:1px;margin-left:2px;margin:3px!important", "3px"},
	} {
		h, err := html.Parse(strings.NewReader("<p>x</p>"))
		if err != nil {
			t.Fatal(err)
		}
		sheet, err := parser.Parse("p{" + tc.body + "}")
		if err != nil {
			t.Fatal(err)
		}
		engine := cssom.NewCSSOM(nil)
		if err := engine.AddStylesForScope(nil, douceuradapter.Wrap(sheet), cssom.Author); err != nil {
			t.Fatal(err)
		}
		root, err := engine.Style(h)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		var visit func(*tree.Node[*styledtree.StyNode])
		visit = func(n *tree.Node[*styledtree.StyNode]) {
			if n.Payload.HTMLNode().Data == "p" {
				found = true
				if got, _ := n.Payload.Styles().Property("margin-left"); string(got) != tc.want {
					t.Errorf("%s: got %q", tc.body, got)
				}
			}
			for _, ch := range n.Children(true) {
				visit(ch)
			}
		}
		visit(root)
		if !found {
			t.Fatal("missing paragraph")
		}
	}
}
