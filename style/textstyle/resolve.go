package textstyle

import (
	"fmt"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/internal/csssyntax"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/bidi"
)

// Resolve returns detached styles in the same order as nodes. It resolves their
// original ancestors once per call; unrelated subtrees are not checked. The
// caller selects nodes, so metadata, hidden content and layout eligibility remain
// application concerns. Keep the source DOM/CSSOM stable during the call. A
// failure returns nil, never a partial snapshot. No cache survives the call.
func Resolve(nodes ...*styledtree.StyNode) ([]Style, error) {
	type computed struct {
		style Style
		root  dimen.DU
	}
	initial := computed{Initial(), Initial().FontSize}
	cache := make(map[*styledtree.StyNode]computed)
	var resolve func(*styledtree.StyNode) (computed, error)
	resolve = func(node *styledtree.StyNode) (computed, error) {
		if node == nil || node.Styles().IsDefaults() {
			return initial, nil
		}
		if s, ok := cache[node]; ok {
			return s, nil
		}
		parent, err := resolve(styledtree.Node(node.Parent()))
		if err != nil {
			return computed{}, err
		}
		s, err := ResolveWithContext(node, Context{Parent: parent.style, RootFontSize: parent.root})
		if err != nil {
			return computed{}, err
		}
		result := computed{s, parent.root}
		if documentElement(node.HTMLNode()) {
			result.root = s.FontSize
		}
		cache[node] = result
		return result, nil
	}
	result := make([]Style, len(nodes))
	for i, node := range nodes {
		if node == nil {
			return nil, fmt.Errorf("text style: nil source node at index %d", i)
		}
		s, err := resolve(node)
		if err != nil {
			return nil, err
		}
		result[i] = s.style
	}
	return result, nil
}

// ResolveWithContext resolves one node using computed source parent/root values.
// Use Resolve when those values have not yet been computed. The node must still
// belong to its source document: dir=auto scans original descendants, including
// descendants outside the caller's selection. HTML hints precede author CSS.
func ResolveWithContext(node *styledtree.StyNode, context Context) (Style, error) {
	if node == nil || node.HTMLNode() == nil {
		return Style{}, fmt.Errorf("text style: missing source node")
	}
	h := node.HTMLNode()
	if !validSize(context.Parent.FontSize) || !validSize(context.RootFontSize) {
		return Style{}, &Error{Node: h, Property: "font context", Err: invalid("font context", "")}
	}
	parent, s := context.Parent, context.Parent
	if h.Type == html.TextNode {
		return s, nil
	}
	s.UnicodeBidi = "normal" // not inherited by elements
	if h.Type != html.ElementNode {
		return s, nil
	}
	switch h.Data {
	case "b", "strong":
		s.Weight = 700
	case "i", "em":
		s.Slant = "italic"
	case "pre":
		s.WhiteSpace = "pre"
	case "bdi":
		s.Direction = firstStrong(h, parent.Direction)
		s.UnicodeBidi = "isolate"
	}
	for _, a := range h.Attr {
		if a.Namespace != "" {
			continue
		}
		switch a.Key {
		case "lang":
			if a.Val == "" {
				s.Language = "und"
				continue
			}
			tag, err := language.Parse(a.Val)
			if err != nil {
				return Style{}, &Error{Node: h, Property: "lang", Value: a.Val, Err: err}
			}
			s.Language = tag.String()
		case "dir":
			switch csssyntax.Lower(a.Val) {
			case "ltr":
				s.Direction = LeftToRight
			case "rtl":
				s.Direction = RightToLeft
			case "auto":
				s.Direction = firstStrong(h, parent.Direction)
			default:
				return Style{}, &Error{Node: h, Property: "dir", Value: a.Val, Err: invalid("dir", a.Val)}
			}
			s.UnicodeBidi = "isolate"
		}
	}
	root := context.RootFontSize
	for _, key := range []string{"font-family", "font-size", "font-weight", "font-style", "color", "white-space", "direction", "unicode-bidi", "line-height", "hyphens"} {
		// rem in the root's other properties uses its own computed font size;
		// rem in the root's font-size uses the initial size.
		if key == "line-height" && documentElement(h) {
			root = s.FontSize
		}
		value, ok := node.Styles().Property(key)
		if !ok {
			continue
		}
		v := string(value)
		switch style.DeclarationKeyword(v) {
		case "inherit":
			assign(&s, parent, key)
			continue
		case "unset":
			if style.IsCascading(key) {
				assign(&s, parent, key)
			} else {
				assign(&s, Initial(), key)
			}
			continue
		case "initial":
			assign(&s, Initial(), key)
			continue
		case "revert", "revert-layer":
			return Style{}, &Error{Node: h, Property: key, Value: v, Err: invalid(key, v)}
		}
		var err error
		switch key {
		case "font-family":
			s.Family, err = family(v)
		case "font-size":
			s.FontSize, err = ResolveFontSize(v, parent.FontSize, root)
		case "font-weight":
			s.Weight, err = weight(v, parent.Weight)
		case "font-style":
			s.Slant, err = keyword(key, v, "normal italic oblique")
		case "color":
			var c style.ColorValue
			c, err = style.ParseColorValue(v)
			if err == nil {
				if c.Current {
					s.Color, s.Foreground = parent.Color, parent.Foreground
				} else {
					s.Foreground = c.Color
					s.Color, _ = csssyntax.Clean(v)
					if name, e := csssyntax.Ident(s.Color); e == nil {
						s.Color = csssyntax.Lower(name)
					}
				}
			}
		case "white-space":
			s.WhiteSpace, err = keyword(key, v, "normal nowrap pre pre-wrap pre-line break-spaces")
		case "direction":
			var word string
			word, err = keyword(key, v, "ltr rtl")
			if word == "rtl" {
				s.Direction = RightToLeft
			} else {
				s.Direction = LeftToRight
			}
		case "unicode-bidi":
			s.UnicodeBidi, err = keyword(key, v, "normal embed isolate bidi-override isolate-override plaintext")
		case "line-height":
			s.LineHeight, err = lineHeight(v, s.FontSize, root)
		case "hyphens":
			s.Hyphens, err = keyword(key, v, "none manual auto")
		}
		if err != nil {
			return Style{}, &Error{Node: h, Property: key, Value: v, Err: err}
		}
	}
	return s, nil
}

func assign(s *Style, from Style, key string) {
	switch key {
	case "font-family":
		s.Family = from.Family
	case "font-size":
		s.FontSize = from.FontSize
	case "font-weight":
		s.Weight = from.Weight
	case "font-style":
		s.Slant = from.Slant
	case "color":
		s.Color, s.Foreground = from.Color, from.Foreground
	case "white-space":
		s.WhiteSpace = from.WhiteSpace
	case "direction":
		s.Direction = from.Direction
	case "unicode-bidi":
		s.UnicodeBidi = from.UnicodeBidi
	case "line-height":
		s.LineHeight = from.LineHeight
	case "hyphens":
		s.Hyphens = from.Hyphens
	}
}

func documentElement(n *html.Node) bool {
	return n != nil && n.Type == html.ElementNode && (n.Parent == nil || n.Parent.Type == html.DocumentNode)
}

func firstStrong(root *html.Node, fallback Direction) Direction {
	var scan func(*html.Node) (Direction, bool)
	scan = func(n *html.Node) (Direction, bool) {
		if n != root && n.Type == html.ElementNode {
			// Inert/metadata text and independent HTML directionality do not
			// determine an enclosing auto element's first strong character.
			switch n.Data {
			case "bdi", "script", "style", "template", "textarea":
				return fallback, false
			}
			for _, a := range n.Attr {
				if a.Namespace == "" && a.Key == "dir" {
					switch csssyntax.Lower(a.Val) {
					case "ltr", "rtl", "auto":
						return fallback, false
					}
				}
			}
		}
		if n.Type == html.TextNode {
			for _, r := range n.Data {
				p, _ := bidi.LookupRune(r)
				switch p.Class() {
				case bidi.L:
					return LeftToRight, true
				case bidi.R, bidi.AL:
					return RightToLeft, true
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if dir, ok := scan(child); ok {
				return dir, true
			}
		}
		return fallback, false
	}
	dir, _ := scan(root)
	return dir
}
