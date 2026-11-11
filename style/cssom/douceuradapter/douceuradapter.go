/*
Package douceuradapter is a concrete implementation of interface cssom.StyleSheet.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>
*/
package douceuradapter

import (
	"errors"
	"fmt"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"strings"
)

// CSSStyles is an adapter for interface cssom.StyleSheet.
// For an explanation of the motivation behind this design, please refer
// to documentation for interface cssom.StyleSheet.
type CSSStyles struct {
	css css.Stylesheet
}

// Wrap a douceur.css.Stylesheet into CssStyles.
// The stylesheet is now managed by the wrapper.
func Wrap(css *css.Stylesheet) *CSSStyles {
	sheet := &CSSStyles{*css}
	return sheet
}

// Empty checks if this stylesheet contains any rules.
//
// Interface cssom.StyleSheet
func (sheet *CSSStyles) Empty() bool {
	return len(sheet.css.Rules) == 0
}

// AppendRules appends rules from another stylesheet.
//
// Interface cssom.StyleSheet
func (sheet *CSSStyles) AppendRules(other cssom.StyleSheet) {
	othercss := other.(*CSSStyles)
	for _, r := range othercss.css.Rules { // append every rule from other
		sheet.css.Rules = append(sheet.css.Rules, r)
	}
}

// Rules returns all the rules of a stylesheet.
//
// Interface style.StyleSheet
func (sheet *CSSStyles) Rules() []cssom.Rule {
	rules := make([]cssom.Rule, len(sheet.css.Rules))
	for i := range sheet.css.Rules {
		r := sheet.css.Rules[i]
		rules[i] = Rule(*r)
	}
	return rules
}

var _ cssom.StyleSheet = &CSSStyles{}

// Rule is an adapter for interface cssom.Rule.
type Rule css.Rule

// Selector returns the prelude / selectors of the rule.
func (r Rule) Selector() string {
	return r.Prelude
}

// declarationKey normalizes standard property names without changing custom names.
func declarationKey(key string) string {
	if strings.HasPrefix(key, "--") {
		return key
	}
	return style.CanonicalPropertyKey(strings.ToLower(key))
}

// winningDeclarations retains the last declaration at equal importance.
// Indices preserve the winners' original positions for shorthand/longhand ties.
func (r Rule) winningDeclarations() map[string]int {
	winners := make(map[string]int)
	for i, d := range r.Declarations {
		key := declarationKey(d.Property)
		previous, exists := winners[key]
		if !exists || !r.Declarations[previous].Important || d.Important {
			winners[key] = i
		}
	}
	return winners
}

// Properties returns each winning property once, in declaration source order.
func (r Rule) Properties() []string {
	winners := r.winningDeclarations()
	props := make([]string, 0, len(winners))
	for i, d := range r.Declarations {
		key := declarationKey(d.Property)
		if winners[key] == i {
			props = append(props, key)
		}
	}
	return props
}

// Value returns the winning declaration value, or NullStyle if absent.
func (r Rule) Value(key string) style.Property {
	if i, ok := r.winningDeclarations()[declarationKey(key)]; ok {
		return style.Property(r.Declarations[i].Value)
	}
	return style.NullStyle
}

// IsImportant reports the importance of the winning declaration.
func (r Rule) IsImportant(key string) bool {
	i, ok := r.winningDeclarations()[declarationKey(key)]
	return ok && r.Declarations[i].Important
}

var _ cssom.Rule = &Rule{}

// StyleExtractionError identifies the style element whose CSS failed to parse.
// Node is a reference into the input tree, which extraction does not modify.
type StyleExtractionError struct {
	Node *html.Node
	Err  error
}

func (e *StyleExtractionError) Error() string {
	return fmt.Sprintf("cannot parse style element: %v", e.Err)
}
func (e *StyleExtractionError) Unwrap() error { return e.Err }

// ExtractStyleElements returns successfully parsed embedded sheets in tree order.
// It ignores parse errors for compatibility; use ExtractStyleElementsWithError
// when callers must distinguish complete extraction from partial results.
func ExtractStyleElements(root *html.Node) []*CSSStyles {
	sheets, _ := ExtractStyleElementsWithError(root)
	return sheets
}

// ExtractStyleElementsWithError recursively scans root (including root itself)
// for style elements, without requiring head/body containers. All descendant text
// nodes of each style element are concatenated without inserted separators.
// Empty/whitespace-only elements are skipped. Malformed sheets are omitted;
// scanning continues and returns successful sheets plus joined node-specific errors.
// Nil input returns no sheets and no error. The input tree is not modified.
func ExtractStyleElementsWithError(root *html.Node) ([]*CSSStyles, error) {
	var sheets []*CSSStyles
	var failures []error
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode && (n.DataAtom == atom.Style || strings.EqualFold(n.Data, "style")) {
			var content strings.Builder
			var text func(*html.Node)
			text = func(node *html.Node) {
				if node.Type == html.TextNode {
					content.WriteString(node.Data)
				}
				for ch := node.FirstChild; ch != nil; ch = ch.NextSibling {
					text(ch)
				}
			}
			text(n)
			if strings.TrimSpace(content.String()) != "" {
				sheet, err := parser.Parse(content.String())
				if err != nil {
					failures = append(failures, &StyleExtractionError{Node: n, Err: err})
				} else {
					sheets = append(sheets, Wrap(sheet))
				}
			}
			return // style content is CSS, not a subtree of independent stylesheets
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			visit(ch)
		}
	}
	visit(root)
	return sheets, errors.Join(failures...)
}
