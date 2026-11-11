package styledtree

import (
	"github.com/npillmayer/w3c-dom/style"
	"golang.org/x/net/html"
	"testing"
)

func TestPropertyValueAncestorAndMissing(t *testing.T) {
	root := NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "div"})
	pm := style.NewPropertyMap()
	pm.Add("color", "red")
	pm.Add("width", "20px")
	root.Payload.SetStyles(pm)
	child := NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "p"})
	root.AddChild(child)
	if got := child.Payload.GetPropertyValue("color", nil); got != "red" {
		t.Fatalf("inherited color = %q", got)
	}
	if got := child.Payload.GetPropertyValue("width", nil); got != "auto" {
		t.Fatalf("non-inherited width = %q", got)
	}
	local := style.NewPropertyMap()
	local.Add("width", "inherit")
	if got := child.Payload.GetPropertyValue("width", local); got != "20px" {
		t.Fatalf("explicit inherit = %q", got)
	}
	local.Add("color", "blue")
	if got := child.Payload.GetPropertyValue("color", local); got != "blue" {
		t.Fatalf("local color = %q", got)
	}
	for _, node := range []*StyNode{nil, root.Payload, child.Payload} {
		if got := node.GetPropertyValue("unknown", nil); got != style.NullStyle {
			t.Fatalf("missing value = %q", got)
		}
	}
}
