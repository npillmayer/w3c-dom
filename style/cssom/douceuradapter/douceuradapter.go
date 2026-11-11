/*
Package douceuradapter is a concrete implementation of interface cssom.StyleSheet.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>
*/
package douceuradapter

import (
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

// ExtractStyleElements visits <head> and <body> elements in an HTML parse
// tree and searches for embedded <style>s. It returns the content of
// style-elements as style sheets. Nil input and missing head/body elements are
// allowed. Empty style elements without content are skipped.
func ExtractStyleElements(htmldoc *html.Node) []*CSSStyles {
	head := findElement(atom.Head, htmldoc)
	body := findElement(atom.Body, htmldoc)
	css := extractStyles(head)
	css2 := extractStyles(body)
	for _, c := range css2 {
		css = append(css, c)
	}
	return css
}

func extractStyles(h *html.Node) []*CSSStyles {
	var css []*CSSStyles
	if h == nil {
		return css
	}
	ch := h.FirstChild
	for ch != nil {
		if ch.DataAtom == atom.Style && ch.FirstChild != nil {
			c, err := parser.Parse(ch.FirstChild.Data)
			if err != nil {
				break
			}
			css = append(css, Wrap(c))
		}
		ch = ch.NextSibling
	}
	return css
}

func findElement(a atom.Atom, h *html.Node) *html.Node {
	if h == nil {
		return nil
	}
	if h.DataAtom == a {
		return h
	}
	ch := h.FirstChild
	for ch != nil {
		r := findElement(a, ch)
		if r != nil && r.DataAtom == a {
			return r
		}
		ch = ch.NextSibling
	}
	return nil
}
