package styledtree

import (
	"strings"

	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style"
	"golang.org/x/net/html"
)

// StyNode is a style node, the building block of the styled tree.
type StyNode struct {
	tree.Node[*StyNode] // we build on top of general purpose tree
	htmlNode            *html.Node
	computedStyles      *style.PropertyMap
}

func (sn *StyNode) String() string {
	h := sn.htmlNode
	switch h.Type {
	case html.DocumentNode:
		return "[#document]"
	case html.ElementNode:
		return "[" + h.Data + "]"
	case html.TextNode:
		return "[#text:" + shortText(h) + "]"
	}
	return "[styled node]"
}

// NewNodeForHTMLNode creates a new styled node linked to an HTML node.
func NewNodeForHTMLNode(html *html.Node) *tree.Node[*StyNode] {
	sn := &StyNode{}
	sn.Payload = sn // Payload will always reference the node itself
	sn.htmlNode = html
	return &sn.Node
}

// Node gets the styled node from a generic tree node.
func Node(n *tree.Node[*StyNode]) *StyNode {
	if n == nil {
		return nil
	}
	return n.Payload
}

// HTMLNode gets the HTML DOM node corresponding to this styled node.
func (sn *StyNode) HTMLNode() *html.Node {
	return sn.Payload.htmlNode
}

// StylesCascade gets the upwards to the enclosing style set.
// func (sn *StyNode) StylesCascade() *style.Styler {
// 	enclosingStyles := Node(sn.Parent())
// 	return enclosingStyles.AsStyler()
// }

// Styles is part of interface style.Styler.
func (sn *StyNode) Styles() *style.PropertyMap {
	return sn.computedStyles
}

// SetStyles sets the styling properties of a styled node.
func (sn *StyNode) SetStyles(styles *style.PropertyMap) {
	sn.computedStyles = styles
}

// GetPropertyValue resolves a local declaration, inheritance and UA defaults.
// pmap supplies the local declarations; nil means no local declarations.
// Explicit initial bypasses UA overrides. Unknown absent properties return NullStyle.
func (sn *StyNode) GetPropertyValue(key string, pmap *style.PropertyMap) style.Property {
	if sn == nil {
		return style.NullStyle
	}
	initial := style.InitialProperty(key)
	var defaults *style.PropertyMap
	for n := sn; n != nil; n = Node(n.Parent()) {
		if n.Styles().IsDefaults() {
			defaults = n.Styles()
			break
		}
	}
	ua := func(n *StyNode) style.Property {
		v := defaults.DefaultProperty(n.HTMLNode(), key)
		switch strings.ToLower(strings.TrimSpace(string(v))) {
		case "inherit", "initial", "unset":
			return initial
		}
		return v
	}
	var resolve func(*StyNode, *style.PropertyMap) style.Property
	resolve = func(n *StyNode, local *style.PropertyMap) style.Property {
		if n == nil || local.IsDefaults() {
			return initial
		}
		value, _ := local.Property(key)
		keyword := strings.ToLower(strings.TrimSpace(string(value)))
		if keyword == "initial" || (keyword == "unset" && !style.IsCascading(key)) {
			return initial
		}
		inherit := keyword == "inherit" || keyword == "unset" || (value.IsEmpty() && style.IsCascading(key))
		if inherit {
			parent := Node(n.Parent())
			// Document and synthetic root are bookkeeping, not CSS parent elements.
			if parent != nil && parent.HTMLNode().Type == html.ElementNode && !parent.Styles().IsDefaults() {
				return resolve(parent, parent.Styles())
			}
			if keyword == "inherit" || keyword == "unset" {
				return initial
			}
			return ua(n)
		}
		if !value.IsEmpty() {
			return value
		}
		return ua(n)
	}
	if pmap.IsDefaults() {
		return ua(sn)
	}
	return resolve(sn, pmap)
}

// --- Helpers ---------------------------------------------------------------

// Creator returns a style-creator for use in CSSOM.
// The returned style.NodeCreator will then build up an instance of a styled tree
// with node type styledtree.StyNode.
//
/*
func Creator() style.NodeCreator {
	return creator{}
}

type creator struct{}

func (c creator) ToStyler(n *tree.Node[*StyNode]) style.Styler {
	return Node(n)
}

func (c creator) StyleForHTMLNode(htmlnode *html.Node) *tree.Node[*StyNode] {
	return NewNodeForHTMLNode(htmlnode)
}

func (c creator) SetStyles(n *tree.Node[*StyNode], m *style.PropertyMap) {
	Node(n).SetStyles(m)
}

var _ style.NodeCreator = creator{}
*/

func shortText(h *html.Node) string {
	s := "\"\\\""
	if len(h.Data) > 10 {
		s += h.Data[:10] + "...\\\"\""
	} else {
		s += h.Data + "\\\"\""
	}
	s = strings.Replace(s, "\n", `\\n`, -1)
	s = strings.Replace(s, "\t", `\\t`, -1)
	s = strings.Replace(s, " ", "\u2423", -1)
	return s
}
