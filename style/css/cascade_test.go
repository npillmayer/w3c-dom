package css

import (
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"testing"
)

func TestCascadedPropertyBoundaries(t *testing.T) {
	root := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.DocumentNode})
	root.Payload.SetStyles(style.InitializeDefaultPropertyValues(nil))
	child := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "p"})
	root.AddChild(child)
	for _, node := range []*styledtree.StyNode{root.Payload, child.Payload} {
		if got, err := GetProperty(node, "color"); err != nil || got != "default" {
			t.Fatalf("default color = %q, %v", got, err)
		}
		for _, key := range []string{"unknown"} {
			if got, err := GetCascadedProperty(node, key); err == nil || got != style.NullStyle {
				t.Fatalf("missing %s = %q, %v", key, got, err)
			}
		}
	}
	local := style.NewPropertyMap()
	local.Add("background-color", "blue") // same group, absent requested key
	child.Payload.SetStyles(local)
	if got, err := GetProperty(child.Payload, "color"); err != nil || got != "default" {
		t.Fatalf("partial group lookup = %q, %v", got, err)
	}
	local.Add("color", "red")
	if got, err := GetProperty(child.Payload, "color"); err != nil || got != "red" {
		t.Fatalf("local color = %q, %v", got, err)
	}
	detached := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: "p"}).Payload
	if got, err := GetProperty(detached, "color"); err != nil || got != "default" {
		t.Fatalf("detached default = %q, %v", got, err)
	}
	for _, node := range []*styledtree.StyNode{nil} {
		if got, err := GetProperty(node, "color"); err == nil || got != style.NullStyle {
			t.Fatalf("absent color = %q, %v", got, err)
		}
	}
	if _, err := GetProperty(nil, "width"); err == nil {
		t.Fatal("nil non-inherited lookup must return error")
	}
	if got := GetLocalProperty(nil, "color"); got != style.NullStyle {
		t.Fatalf("nil local value = %q", got)
	}
}
