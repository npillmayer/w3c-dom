package cssom_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestRegisteredSplittersUsedDuringStyling(t *testing.T) {
	engine := cssom.NewCSSOM(nil)
	engine.RegisterCompoundSplitter(nil)
	engine.RegisterCompoundSplitter(func(key string, value style.Property) ([]style.KeyValue, error) {
		if key != "custom-gap" {
			return nil, fmt.Errorf("unsupported: %s", key)
		}
		return []style.KeyValue{{Key: "margin-left", Value: value}}, nil
	})
	engine.RegisterCompoundSplitter(func(key string, value style.Property) ([]style.KeyValue, error) {
		switch key {
		case "custom-gap":
			return []style.KeyValue{{Key: "margin-left", Value: "wrong"}}, nil
		case "custom-size":
			return []style.KeyValue{{Key: "width", Value: value}}, nil
		case "padding":
			return []style.KeyValue{{Key: "padding-top", Value: "wrong"}}, nil
		}
		return nil, fmt.Errorf("unsupported: %s", key)
	})
	// A second styling call must retain the same registrations.
	for _, value := range []string{"7px", "9px"} {
		h, err := html.Parse(strings.NewReader(`<p style="custom-gap:` + value + `;custom-size:12px;padding:3px;unknown:kept">text</p>`))
		if err != nil {
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
				for key, want := range map[string]string{"margin-left": value, "width": "12px", "padding-top": "3px", "unknown": "kept"} {
					if got, ok := n.Payload.Styles().Property(key); !ok || string(got) != want {
						t.Errorf("%s = %q, %v; want %s", key, got, ok, want)
					}
				}
				for _, key := range []string{"custom-gap", "custom-size"} {
					if _, ok := n.Payload.Styles().Property(key); ok {
						t.Errorf("unexpanded %s", key)
					}
				}
			}
			for _, child := range n.Children(true) {
				visit(child)
			}
		}
		visit(root)
		if !found {
			t.Fatal("missing paragraph")
		}
	}
}
