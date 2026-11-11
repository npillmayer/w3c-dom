package css_test

import (
	"fmt"
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

func TestFontMetadataResolution(t *testing.T) {
	for _, tc := range []struct {
		key, initial, parent, override string
		inherited                      bool
	}{
		{"font-family", "serif", `'Example Sans', sans-serif`, "monospace", true},
		{"font-size", "medium", "20px", "18px", true},
		{"font-weight", "normal", "700", "500", true},
		{"font-style", "normal", "italic", "oblique", true},
		{"unicode-bidi", "normal", "isolate", "embed", false},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if style.InitialProperty(tc.key) != style.Property(tc.initial) || style.IsCascading(tc.key) != tc.inherited || style.GroupNameFromPropertyKey(tc.key) != style.PGText {
				t.Fatal("incorrect metadata")
			}
			for _, configured := range []bool{false, true} {
				markup := fmt.Sprintf(`<html style="%s:inherit"><body><p id="parent" style="%s:%s"><span id="plain"></span><span id="inherit" style="%s:inherit"></span><span id="initial" style="%s:initial"><b id="grandchild"></b></span><span id="unset" style="%s:unset"></span></p><p id="outside"></p></body></html>`, tc.key, tc.key, tc.parent, tc.key, tc.key, tc.key)
				h, err := html.Parse(strings.NewReader(markup))
				if err != nil {
					t.Fatal(err)
				}
				var defaults []style.KeyValue
				fallback := tc.initial
				if configured {
					defaults = []style.KeyValue{{Key: tc.key, Value: style.Property(tc.override)}}
					fallback = tc.override
				}
				engine := cssom.NewCSSOM(defaults)
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
				plain, unset, grand, outside := fallback, tc.initial, fallback, fallback
				if tc.inherited {
					plain = tc.parent
					unset = tc.parent
					grand = tc.initial
					outside = tc.initial
				} // html explicitly inherits initial
				for id, want := range map[string]string{"html": tc.initial, "parent": tc.parent, "plain": plain, "inherit": tc.parent, "initial": tc.initial, "unset": unset, "grandchild": grand, "outside": outside} {
					n := nodes[id]
					if n == nil {
						t.Fatalf("missing %s", id)
					}
					got, err := css.GetProperty(n, tc.key)
					if err != nil || string(got) != want {
						t.Errorf("configured=%v %s = %q %v want %q", configured, id, got, err, want)
					}
					if dom.NodeFromStyledNode(n).ComputedStyles().GetPropertyValue(tc.key) != got {
						t.Error("DOM accessor differs")
					}
				}
				if got, _ := css.GetProperty(root.Payload, tc.key); string(got) != fallback {
					t.Errorf("UA default = %q want %q", got, fallback)
				}
			}
			detached := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "span"}).Payload
			if got, _ := css.GetProperty(detached, tc.key); string(got) != tc.initial {
				t.Errorf("detached default = %q", got)
			}
		})
	}
}
