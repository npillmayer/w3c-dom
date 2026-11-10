package cssom_test

import (
	"strings"
	"testing"

	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

// These expectations are cascade outcomes, independent of the selector parser's
// own specificity implementation: https://www.w3.org/TR/selectors-3/#specificity.
func TestSelectorSpecificity(t *testing.T) {
	for _, tc := range []struct {
		name, css, inline, want string
	}{
		{"compound ID", `p#x {color:red} .c {color:blue}`, "", "red"},
		{"multiple classes", `.c.d {color:red} .c {color:blue}`, "", "red"},
		{"attribute", `.c[data-v] {color:red} .c {color:blue}`, "", "red"},
		{"attribute value punctuation", `[data-v="a.b#c: d"] {color:red} p.c {color:blue}`, "", "blue"},
		{"pseudo class", `.c:first-child {color:red} .c {color:blue}`, "", "red"},
		{"negation argument", `p:not(#other) {color:red} p.c {color:blue}`, "", "red"},
		{"escaped ID", `#\78 {color:red} .c {color:blue}`, "", "red"},
		{"universal", `p {color:red} * {color:blue}`, "", "red"},
		{"combinator adds no weight", `body > p {color:red} body p {color:blue}`, "", "blue"},
		{"class beats many types", `.c {color:red} p` + strings.Repeat(`:not(span)`, 20) + ` {color:blue}`, "", "red"},
		{"ID beats many classes", `#x {color:red} ` + strings.Repeat(`.c`, 20) + ` {color:blue}`, "", "red"},
		{"source order cannot outweigh specificity", `#x {color:red}` + strings.Repeat(`.c {color:blue}`, 1200), "", "red"},
		{"equal specificity uses later rule", `.c {color:red} .d {color:blue}`, "", "blue"},
		{"strongest matching branch", `#x, p {color:red} .c {color:blue}`, "", "red"},
		{"nonmatching branch has no weight", `#other, p {color:red} .c {color:blue}`, "", "blue"},
		{"important higher specificity wins later", `.c {color:blue!important} #x {color:red!important}`, "", "red"},
		{"important retains specificity", `#x {color:red!important} .c {color:blue!important}`, "", "red"},
		{"important tie uses later rule", `.c {color:red!important} .d {color:blue!important}`, "", "blue"},
		{"important precedes specificity", `#x {color:red} .c {color:blue!important}`, "", "blue"},
		{"inline precedes selector", `#x#x {color:red}`, "color:green", "green"},
		{"important precedes normal inline", `.c {color:red!important}`, "color:green", "red"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			colors := specificityColors(t, `<p id="x" class="c d" data-v="a.b#c: d" style="`+tc.inline+`">text</p>`, tc.css)
			if got := colors["x"]; got != tc.want {
				t.Fatalf("color = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSelectorListSpecificityPerNode(t *testing.T) {
	colors := specificityColors(t, `<p id="x" class="c">one</p><p id="y" class="c">two</p>`, `#x, p {color:red} .c {color:blue}`)
	if colors["x"] != "red" || colors["y"] != "blue" {
		t.Fatalf("selector-list specificity leaked between nodes: %v", colors)
	}
}

func TestSpecificityAcrossStylesheets(t *testing.T) {
	for _, tc := range []struct{ first, want string }{
		{`.c {color:red}`, "blue"},
		{`#x {color:red}`, "red"},
	} {
		colors := specificityColors(t, `<p id="x" class="c">text</p>`, tc.first, `.c {color:blue}`)
		if colors["x"] != tc.want {
			t.Fatalf("%s: color = %q, want %q", tc.first, colors["x"], tc.want)
		}
	}
}

func specificityColors(t *testing.T, markup string, sheets ...string) map[string]string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	om := cssom.NewCSSOM(nil)
	for _, source := range sheets {
		sheet, err := parser.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := om.AddStylesForScope(nil, douceuradapter.Wrap(sheet), cssom.Author); err != nil {
			t.Fatal(err)
		}
	}
	root, err := om.Style(doc)
	if err != nil {
		t.Fatal(err)
	}
	colors := map[string]string{}
	var visit func(*tree.Node[*styledtree.StyNode])
	visit = func(n *tree.Node[*styledtree.StyNode]) {
		for _, attr := range n.Payload.HTMLNode().Attr {
			if attr.Key == "id" {
				value, _ := n.Payload.Styles().Property("color")
				colors[attr.Val] = string(value)
			}
		}
		for _, child := range n.Children(true) {
			visit(child)
		}
	}
	visit(root)
	return colors
}
