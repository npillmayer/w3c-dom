package css_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/style/display"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func displayNodes(t *testing.T, markup string, defaults []style.KeyValue, sheets ...cssom.StyleSheet) map[string]*styledtree.StyNode {
	t.Helper()
	h, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	engine := cssom.NewCSSOM(defaults)
	for _, sheet := range sheets {
		if err := engine.AddStylesForScope(nil, sheet, cssom.Author); err != nil {
			t.Fatal(err)
		}
	}
	root, err := engine.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]*styledtree.StyNode{}
	var walk func(*tree.Node[*styledtree.StyNode])
	walk = func(n *tree.Node[*styledtree.StyNode]) {
		for _, a := range n.Payload.HTMLNode().Attr {
			if a.Key == "id" {
				nodes[a.Val] = n.Payload
			}
		}
		for _, child := range n.Children(true) {
			walk(child)
		}
	}
	walk(root)
	return nodes
}
func displaySheet(t *testing.T, source string) cssom.StyleSheet {
	t.Helper()
	s, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	return douceuradapter.Wrap(s)
}
func wantDisplay(t *testing.T, node *styledtree.StyNode, want string) {
	t.Helper()
	if node == nil {
		t.Fatal("missing node")
	}
	got, err := css.GetDisplay(node)
	expected, e := display.Parse(want)
	if e != nil {
		t.Fatal(e)
	}
	if err != nil || got != expected {
		t.Fatalf("GetDisplay(%s)=%v, %v; want %s", node, got, err, want)
	}
}

func TestGetDisplayDefaultsAndInheritance(t *testing.T) {
	nodes := displayNodes(t, `<html id="root" style="display:inherit"><body><div id="parent" style="display:grid"><p id="p">text</p><span id="span">text</span><p id="initial" style="display:initial">x</p><p id="unset" style="display:unset">x</p><span id="inherit" style="display:inherit"><b id="chain" style="display:inherit">x</b></span><span id="escaped" style="display:/**/\69 nherit/**/">x</span></div></body></html>`, nil)
	for id, want := range map[string]string{"root": "inline", "parent": "grid", "p": "block", "span": "inline", "initial": "inline", "unset": "inline", "inherit": "grid", "chain": "grid", "escaped": "grid"} {
		wantDisplay(t, nodes[id], want)
	}
	// Configured UA defaults apply only when no declaration wins. Explicit initial
	// and root inherit select the actual CSS initial value, not the UA override.
	nodes = displayNodes(t, `<html id="root" style="display:inherit"><body><p id="p">x</p><p id="initial" style="display:INITIAL">x</p><p id="unset" style="display:unset">x</p></body></html>`, []style.KeyValue{{Key: "display", Value: "inline flow-root"}})
	for id, want := range map[string]string{"root": "inline", "p": "inline-block", "initial": "inline", "unset": "inline"} {
		wantDisplay(t, nodes[id], want)
	}
}

func TestDisplayDeclarationSelection(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"display:inline;display:bogus", "inline"},
		{"display:inline;display:bogus!important", "inline"},
		{"display:bogus!important;display:inline", "inline"},
		{"display:inline!important;display:grid", "inline"},
		{"display:inline!important;display:grid!important", "grid"},
		{"display:inline;display:inline flow-root", "inline-block"},
		{"display:block inline;display:bogus", "block"},
		{"display:inline;display:initial", "inline"},
		{`display:inline;DISPLAY:/**/\69 nherit/**/`, "grid"},
		{"display:inline;display:unset", "inline"},
		{"display:inline;display:revert-layer;display:grid", "grid"},
		{"display:inline;display:initial bogus", "inline"},
		{"display:bl/**/ock;display:bogus", "block"},
		{"display:inline;display:block-inline", "inline"},
	} {
		for _, inline := range []bool{false, true} {
			t.Run(tc.body+map[bool]string{true: "/inline", false: "/sheet"}[inline], func(t *testing.T) {
				markup := `<div style="display:grid"><p id="x" style="color:red;` + tc.body + `">x</p></div>`
				var sheets []cssom.StyleSheet
				if !inline {
					markup = `<div style="display:grid"><p id="x">x</p></div>`
					sheets = append(sheets, displaySheet(t, `p{color:red;`+tc.body+`}`))
				}
				nodes := displayNodes(t, markup, nil, sheets...)
				wantDisplay(t, nodes["x"], tc.want)
				if color, err := css.GetProperty(nodes["x"], "color"); err != nil || color != "red" {
					t.Fatalf("other property lost: %s %v", color, err)
				}
			})
		}
	}
}

func TestDisplayCascadePriority(t *testing.T) {
	for _, tc := range []struct{ sheet, inline, want string }{
		{`p{display:inline} #x{display:bogus!important}`, "", "inline"},
		{`#x{display:inline} p{display:grid}`, "", "inline"},
		{`#x{display:inline} p{display:grid!important}`, "", "grid"},
		{`p{display:inline!important}`, "display:grid", "inline"},
		{`p{display:inline!important}`, "display:grid!important", "grid"},
		{`p{display:inline}`, "display:bogus!important", "inline"},
		{`p{display:grid}`, "display:InLiNe FLOW", "inline"},
		{`p{display:inline} p{display:grid}`, "", "grid"},
	} {
		nodes := displayNodes(t, `<p id="x" style="`+tc.inline+`">x</p>`, nil, displaySheet(t, tc.sheet))
		wantDisplay(t, nodes["x"], tc.want)
	}
	nodes := displayNodes(t, `<p id="x">x</p>`, nil, displaySheet(t, `p{display:inline}`), displaySheet(t, `p{display:bogus!important}`))
	wantDisplay(t, nodes["x"], "inline")
	nodes = displayNodes(t, `<p id="x">x</p>`, nil, displaySheet(t, `p{display:inline}`), displaySheet(t, `p{display:grid}`))
	wantDisplay(t, nodes["x"], "grid")
}

func TestGetDisplayPreservesValuesAndSpelling(t *testing.T) {
	for _, input := range []string{"InLiNe FLOW", `\69 nline flow-root`, "block grid", "inline-flex", "table-cell", "ruby", "none", "contents", "inline list-item", "run-in flow-root"} {
		nodes := displayNodes(t, `<p id="x" style="display:`+input+`">x</p>`, nil)
		wantDisplay(t, nodes["x"], input)
		if raw, err := css.GetProperty(nodes["x"], "display"); err != nil || string(raw) != input {
			t.Fatalf("ordinary spelling changed: %q %v", raw, err)
		}
	}
}

func TestGetDisplayUnsupportedRollback(t *testing.T) {
	for _, keyword := range []string{"revert", "revert-layer", `\72 evert-layer`} {
		for _, inline := range []bool{false, true} {
			body := `display:inline;display:` + keyword + `;display:bogus!important`
			markup := `<p id="x" style="` + body + `"><span id="inherit" style="display:inherit">x</span><b id="normal">x</b></p>`
			var sheets []cssom.StyleSheet
			if !inline {
				markup = `<p id="x"><span id="inherit" style="display:inherit">x</span><b id="normal">x</b></p>`
				sheets = append(sheets, displaySheet(t, `p{`+body+`}`))
			}
			nodes := displayNodes(t, markup, nil, sheets...)
			for _, id := range []string{"x", "inherit"} {
				v, err := css.GetDisplay(nodes[id])
				if v != (display.Value{}) || !errors.Is(err, css.ErrUnsupportedDisplayCascade) {
					t.Fatalf("%s inline=%v: %v %v", keyword, inline, v, err)
				}
			}
			wantDisplay(t, nodes["normal"], "inline")
		}
	}
}

func TestGetDisplayDirectMapsAndInvalidNodes(t *testing.T) {
	for _, node := range []*styledtree.StyNode{nil, {}, styledtree.NewNodeForHTMLNode(nil).Payload, styledtree.NewNodeForHTMLNode(&html.Node{Type: html.TextNode, Data: "x"}).Payload, styledtree.NewNodeForHTMLNode(&html.Node{Type: html.DocumentNode}).Payload} {
		v, err := css.GetDisplay(node)
		if err == nil || v != (display.Value{}) {
			t.Fatalf("accepted non-element: %v %v", v, err)
		}
	}
	for _, raw := range []style.Property{`/**/\69 nitial/**/`, `\69 nherit`, "UNSET"} {
		nodes := displayNodes(t, `<p id="x">x</p>`, []style.KeyValue{{Key: "display", Value: raw}})
		wantDisplay(t, nodes["x"], "inline")
	}
	n := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "p"}).Payload
	wantDisplay(t, n, "block")
	for _, tc := range []struct{ raw, want string }{{`/**/\69 nitial/**/`, "inline"}, {`\69 nherit`, "inline"}, {"unset", "inline"}, {"grid", "grid"}} {
		p := style.NewPropertyMap()
		p.Add("display", style.Property(tc.raw))
		n.SetStyles(p)
		wantDisplay(t, n, tc.want)
	}
	for _, raw := range []string{"bogus", "block-inline", "\u00a0initial", "İNİTİAL"} {
		p := style.NewPropertyMap()
		p.Add("display", style.Property(raw))
		n.SetStyles(p)
		v, err := css.GetDisplay(n)
		if v != (display.Value{}) || !errors.Is(err, display.ErrInvalidValue) {
			t.Fatalf("bad direct property %q: %v %v", raw, v, err)
		}
	}
}

// External Rule implementations are also filtered at final cascade selection,
// even when they do not implement the built-in adapters' duplicate validation.
type displayRule struct{ value string }

func (r displayRule) Selector() string            { return "p" }
func (r displayRule) Properties() []string        { return []string{"display"} }
func (r displayRule) Value(string) style.Property { return style.Property(r.value) }
func (r displayRule) IsImportant(string) bool     { return true }

type displayRules []cssom.Rule

func (s displayRules) Rules() []cssom.Rule          { return s }
func (s displayRules) Empty() bool                  { return len(s) == 0 }
func (s displayRules) AppendRules(cssom.StyleSheet) { panic("unused") }
func TestDisplayValidationForExternalRules(t *testing.T) {
	nodes := displayNodes(t, `<p id="x">x</p>`, nil, displayRules{displayRule{"inline"}, displayRule{"bogus"}})
	wantDisplay(t, nodes["x"], "inline")
	nodes = displayNodes(t, `<p id="x">x</p>`, nil, displayRules{displayRule{"bogus"}})
	wantDisplay(t, nodes["x"], "block")
}
