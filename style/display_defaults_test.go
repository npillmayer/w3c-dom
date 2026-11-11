package style

import (
	"golang.org/x/net/html"
	"testing"
)

func TestHTMLDisplayDefaults(t *testing.T) {
	for want, tags := range map[Property][]string{
		"inline":       {"a", "acronym", "b", "blink", "br", "canvas", "code", "em", "i", "img", "label", "math", "picture", "q", "s", "span", "strong", "sup", "svg", "video", "small", "sub", "abbr", "custom-element", "EM"},
		"block":        {"html", "body", "div", "article", "aside", "address", "blockquote", "figure", "figcaption", "footer", "form", "hr", "main", "nav", "menu", "pre", "ol", "ul", "section", "h1", "h6"},
		"none":         {"head", "title", "style", "script", "template", "meta", "link"},
		"inline-block": {"button", "input", "textarea", "select"},
		"list-item":    {"li", "summary"}, "block-inline": {"p"},
		"table": {"table"}, "table-cell": {"td", "th"}, "table-row": {"tr"}, "table-caption": {"caption"}, "table-footer-group": {"tfoot"},
	} {
		for _, tag := range tags {
			if got := DisplayPropertyForHTMLNode(&html.Node{Type: html.ElementNode, Data: tag}); got != want {
				t.Errorf("%s=%q want %q", tag, got, want)
			}
		}
	}
	if DisplayPropertyForHTMLNode(nil) != "none" || DisplayPropertyForHTMLNode(&html.Node{Type: html.DocumentNode}) != "block" || DisplayPropertyForHTMLNode(&html.Node{Type: html.TextNode}) != "none" {
		t.Error("non-element defaults changed")
	}
}
