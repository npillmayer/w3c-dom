package cssom_test

import (
	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestWrapAliasCascade(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"word-wrap:normal;overflow-wrap:break-word", "break-word"},
		{"overflow-wrap:normal;word-wrap:break-word", "break-word"},
		{"word-wrap:normal!important;overflow-wrap:break-word", "normal"},
		{"overflow-wrap:normal!important;word-wrap:break-word", "normal"},
		{"word-wrap:normal!important;overflow-wrap:break-word!important", "break-word"},
		{"overflow-wrap:normal!important;word-wrap:break-word!important", "break-word"},
	} {
		for _, inline := range []bool{false, true} {
			markup := "<p><span>x</span></p>"
			if inline {
				markup = `<p style="` + tc.body + `"><span>x</span></p>`
			}
			h, err := html.Parse(strings.NewReader(markup))
			if err != nil {
				t.Fatal(err)
			}
			engine := cssom.NewCSSOM(nil)
			if !inline {
				sheet, err := parser.Parse("p{" + tc.body + "}")
				if err != nil {
					t.Fatal(err)
				}
				if err := engine.AddStylesForScope(nil, douceuradapter.Wrap(sheet), cssom.Author); err != nil {
					t.Fatal(err)
				}
			}
			root, err := engine.Style(h)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			var visit func(*tree.Node[*styledtree.StyNode])
			visit = func(n *tree.Node[*styledtree.StyNode]) {
				if tag := n.Payload.HTMLNode().Data; tag == "p" || tag == "span" {
					count++
					for _, key := range []string{"word-wrap", "overflow-wrap"} {
						if got, err := css.GetProperty(n.Payload, key); err != nil || string(got) != tc.want {
							t.Errorf("inline=%v %s %s: %q %v", inline, tc.body, key, got, err)
						}
					}
				}
				for _, ch := range n.Children(true) {
					visit(ch)
				}
			}
			visit(root)
			if count != 2 {
				t.Fatal("missing test nodes")
			}
		}
	}
}
