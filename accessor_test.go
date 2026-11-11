package dom_test

import (
	dom "github.com/npillmayer/w3c-dom"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestComputedStylesDefaultAndMissing(t *testing.T) {
	h, err := html.Parse(strings.NewReader("<p>text</p>"))
	if err != nil {
		t.Fatal(err)
	}
	engine := cssom.NewCSSOM(nil)
	root, err := engine.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	styles := dom.NodeFromStyledNode(root.Payload).ComputedStyles()
	for key, want := range map[string]style.Property{"color": "default", "word-wrap": "normal", "flow-into": "none", "unknown": ""} {
		if got := styles.GetPropertyValue(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestEmptyAttributeAccessors(t *testing.T) {
	var nilNode *dom.W3CNode
	nodes := []*dom.W3CNode{nilNode}
	for _, kind := range []html.NodeType{html.DocumentNode, html.TextNode, html.ElementNode} {
		nodes = append(nodes, dom.NodeFromStyledNode(styledtree.NewNodeForHTMLNode(&html.Node{Type: kind}).Payload))
	}
	for _, node := range nodes {
		attrs := node.Attributes()
		if got := attrs.Length(); got != 0 {
			t.Errorf("Length = %d", got)
		}
		for _, i := range []int{-1, 0, 1} {
			if attrs.Item(i) != nil {
				t.Errorf("Item(%d) is nonnil", i)
			}
		}
		if attrs.GetNamedItem("id") != nil {
			t.Error("missing attribute found")
		}
	}
	h := &html.Node{Type: html.ElementNode, Attr: []html.Attribute{{Key: "id", Val: "example"}}}
	attrs := dom.NodeFromStyledNode(styledtree.NewNodeForHTMLNode(h).Payload).Attributes()
	if attrs.Length() != 1 || attrs.Item(0).Value() != "example" || attrs.GetNamedItem("id").Value() != "example" {
		t.Fatal("existing attribute lost")
	}
}

func TestFromHTMLParseTreeEmptyStyle(t *testing.T) {
	h, err := html.Parse(strings.NewReader(`<style></style><p>text</p>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := dom.FromHTMLParseTree(h, nil); got == nil {
		t.Fatal("empty style prevented DOM construction")
	}
	for _, h := range []*html.Node{{Type: html.DocumentNode}, {Type: html.ElementNode, Data: "div"}} {
		if got := dom.FromHTMLParseTree(h, nil); got == nil {
			t.Fatal("missing containers prevented DOM construction")
		}
	}
}
