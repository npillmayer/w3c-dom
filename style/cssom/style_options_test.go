package cssom

import (
	"strings"
	"testing"

	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestStyleSubtreeBoundaryPreservesSourceSelectors(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<body><section style="color:blue"><template id="inert"><p id="stored" style="color:red"></p></template><p id="active">text</p></section></body>`))
	if err != nil {
		t.Fatal(err)
	}
	var before strings.Builder
	if err := html.Render(&before, doc); err != nil {
		t.Fatal(err)
	}
	engine := NewCSSOM(nil)
	sheet := InspectStylesheet(`section > template + p:nth-child(2) { margin-left:7px }`)
	if err := engine.AddStylesForScope(nil, sheet.Sheet, Author); err != nil {
		t.Fatal(err)
	}
	root, err := engine.StyleWithOptions(doc, StyleOptions{SkipSubtree: func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "template"
	}})
	if err != nil {
		t.Fatal(err)
	}
	var active *styledtree.StyNode
	var walk func(*styledtree.StyNode)
	walk = func(n *styledtree.StyNode) {
		for _, a := range n.HTMLNode().Attr {
			if a.Key == "id" {
				if a.Val == "inert" || a.Val == "stored" {
					t.Errorf("excluded styled node: %s", a.Val)
				}
				if a.Val == "active" {
					active = n
				}
			}
		}
		for _, child := range n.Children(true) {
			walk(child.Payload)
		}
	}
	walk(root.Payload)
	if active == nil {
		t.Fatal("missing active node")
	}
	for key, want := range map[string]string{"color": "blue", "margin-left": "7px"} {
		got, err := css.GetProperty(active, key)
		if err != nil || string(got) != want {
			t.Errorf("%s=%s, %v; want %s", key, got, err, want)
		}
	}
	// The excluded inline declarations never entered this CSSOM.
	var scan func(*html.Node)
	scan = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == "stored" && len(engine.rulesTree.StylesheetsForHTMLNode(n)) != 0 {
				t.Error("collected inline CSS from excluded subtree")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			scan(c)
		}
	}
	scan(doc)
	var after strings.Builder
	if err := html.Render(&after, doc); err != nil || before.String() != after.String() {
		t.Fatal("styling modified the source DOM", err)
	}
	// A subsequent ordinary call on the same engine is not permanently filtered.
	full, err := engine.Style(doc)
	if err != nil || full == nil {
		t.Fatal("ordinary styling failed", err)
	}
	found := false
	var check func(*styledtree.StyNode)
	check = func(n *styledtree.StyNode) {
		for _, a := range n.HTMLNode().Attr {
			found = found || a.Key == "id" && a.Val == "stored"
		}
		for _, child := range n.Children(true) {
			check(child.Payload)
		}
	}
	check(full.Payload)
	if !found {
		t.Fatal("Style retained an earlier traversal boundary")
	}
}

func TestStyleWithOptionsRejectsExcludedRoot(t *testing.T) {
	for _, doc := range []*html.Node{nil, {Type: html.DocumentNode}} {
		root, err := NewCSSOM(nil).StyleWithOptions(doc, StyleOptions{SkipSubtree: func(*html.Node) bool { return true }})
		if root != nil || err == nil {
			t.Fatal("expected atomic root failure")
		}
	}
}

func TestStylingDoesNotUseLayoutElementAllowlist(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<body><bdi>a</bdi><small>b</small><x-custom>c</x-custom><dl><dt>d</dt><dd>e</dd></dl></body>`))
	if err != nil {
		t.Fatal(err)
	}
	engine := NewCSSOM(nil)
	sheet := InspectStylesheet(`bdi,small,x-custom,dl,dt,dd {color:red}`)
	if err := engine.AddStylesForScope(nil, sheet.Sheet, Author); err != nil {
		t.Fatal(err)
	}
	root, err := engine.Style(doc)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var walk func(*styledtree.StyNode)
	walk = func(n *styledtree.StyNode) {
		switch n.HTMLNode().Data {
		case "bdi", "small", "x-custom", "dl", "dt", "dd":
			count++
			if got, err := css.GetProperty(n, "color"); err != nil || got != "red" {
				t.Errorf("%s lost authored color: %s, %v", n.HTMLNode().Data, got, err)
			}
		}
		for _, child := range n.Children(true) {
			walk(child.Payload)
		}
	}
	walk(root.Payload)
	if count != 6 {
		t.Fatalf("checked %d included source elements, want 6", count)
	}
}
