package cssom

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"github.com/gorilla/css/scanner"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// CSSOM is the "CSS Object Model", similar to the DOM for HTML.
// Our CSSOM consists of a set of stylesheets, each relevant for a sub-tree
// of the HTML parse tree. This sub-tree is called the "scope" of the stylesheet.
// Sub-trees are identified through the top node.
//
// Stylesheets are wrapped into an internal rules tree.
type CSSOM struct {
	rulesTree         *rulesTreeType               // style sheets
	defaultProperties *style.PropertyMap           // "user agent" style properties
	compoundSplitters []CompoundPropertiesSplitter // split up compound properties
}

// NewCSSOM creates an empty CSSOM.
// Clients are allowed to supply a map of additional/custom CSS property values.
// These may override values of the default ("user-agent") style sheet,
// or introduce completely new styling properties.
func NewCSSOM(additionalProperties []style.KeyValue) CSSOM {
	cssom := CSSOM{}
	cssom.rulesTree = newRulesTree()
	cssom.defaultProperties = style.InitializeDefaultPropertyValues(additionalProperties)
	cssom.compoundSplitters = make([]CompoundPropertiesSplitter, 1)
	cssom.compoundSplitters[0] = style.SplitCompoundProperty
	return cssom
}

// AddStylesForScope includes a stylesheet to a CSSOM and sets the scope for
// the stylesheet. If a stylesheet for the scope already exists, the
// styles are merged. A nonnil scope applies to that element and its descendants.
// A nil scope applies to the entire document. At equal importance and specificity,
// sheets are ordered globally, then from outer to inner scope; registration order
// is retained within each scope. Inline declarations apply only to their own node.
//
// The stylsheet may not be nil.
// source hints to where the stylesheet comes from.
// Its value will affect the calculation of specifity for rules of this
// stylesheet.
//
// Inline-styles will be handled on the fly, generating "mini-stylesheets"
// while walking the HTML parse tree. For `<style>`-elements, clients have to extract
// the styles in advance and wrap them into stylesheets.
func (cssom CSSOM) AddStylesForScope(scope *html.Node, css StyleSheet, source PropertySource) error {
	if scope != nil && scope.Type != html.ElementNode {
		return errors.New("Can style element nodes only")
	}
	if css == nil {
		return errors.New("Style sheet is nil")
	}
	cssom.rulesTree.StoreStylesheetForHTMLNode(scope, css, source)
	return nil
}

// --- A rules tree -----------------------------------------------------

// RulesTree holds the styling rules of a stylesheet.
//
// Status: Currently this is not really a tree.
// Optimize some day (see
// https://hacks.mozilla.org/2017/08/inside-a-super-fast-css-engine-quantum-css-aka-stylo/).
type rulesTreeType struct {
	stylesheets *sync.Map      // of type html.Node -> []stylesheetType
	selectors   *sync.Map      // string -> cascadia.SelectorGroup
	source      PropertySource // where do these rules come from?
}

// ad-hoc container type for stylesheets and their origin.
// To be stored in a map (per HTML node).
type stylesheetType struct {
	stylesheet StyleSheet
	source     PropertySource
}

func newRulesTree() *rulesTreeType {
	rt := &rulesTreeType{}
	rt.stylesheets = &sync.Map{}
	rt.selectors = &sync.Map{}
	return rt
}

// StylesheetsForHTMLNode retrieves all style sheets registered for
// an html node. If h is nil it is interpreted as the root scope.
func (rt rulesTreeType) StylesheetsForHTMLNode(h *html.Node) []stylesheetType {
	if h == nil {
		h = rootElement
	}
	sheets, found := rt.stylesheets.Load(h)
	if !found {
		return nil
	}
	return sheets.([]stylesheetType)
}

// StoreStylesheetForHTMLNode registers a style sheet for
// an html node. If h is nil it is interpreted as the root scope.
func (rt rulesTreeType) StoreStylesheetForHTMLNode(h *html.Node, sheet StyleSheet,
	source PropertySource) {
	//
	if h == nil {
		h = rootElement
	}
	sheets := rt.StylesheetsForHTMLNode(h)
	if sheets == nil {
		tracer().Debugf("Adding first style sheet for HTML node %v", h)
		rt.stylesheets.Store(h, []stylesheetType{{sheet, source}})
	} else {
		tracer().Debugf("Adding another style sheet for HTML node %v", h)
		sheets = append(sheets, stylesheetType{sheet, source})
		rt.stylesheets.Store(h, sheets)
	}
}

// Empty is a predicate wether a rulestree is empty, i.e. does not contain
// any rules.
func (rt *rulesTreeType) Empty() bool {
	if rt == nil {
		return true
	}
	csscnt := 0
	rt.stylesheets.Range(func(interface{}, interface{}) bool {
		csscnt++
		return true
	})
	tracer().Debugf("Style sheet entries in rules tree for %d scopes", csscnt)
	return csscnt == 0
}

// CompoundPropertiesSplitter splits compound properties into atomic properties.
// Compunt properties are properties which abbreviate the
// setting of more fine grained propertes. An example is
//
//	padding: 10px 20px
//
// which sets the following detail properties:
//
//	padding-top:    10px
//	padding-right:  20px
//	padding-bottom: 10px
//	padding-left:   20px
//
// Standard CSS compound properties are known by default, but clients are
// allowed to extend the set of compound properties.
type CompoundPropertiesSplitter func(string, style.Property) ([]style.KeyValue, error)

// RegisterCompoundSplitter appends a splitter to this CSSOM. Nil splitters are
// ignored. During styling, built-in splitters run first, then registered splitters
// in registration order until one succeeds. Register before calling Style;
// registration is not synchronized with styling.
func (cssom *CSSOM) RegisterCompoundSplitter(splitter CompoundPropertiesSplitter) {
	if splitter != nil {
		cssom.compoundSplitters = append(cssom.compoundSplitters, splitter)
	}
}

// --- Style Rule Matching ----------------------------------------------

// PropertySource denotes where CSS properties come from and therewith determines
// the specifity of properties. Properties may be defined at different places in HTML:
// as a sytlesheet reference link, within a <script> element in the HTML file, or in an
// attribute value.
//
// PropertySource affects the specifity of rules: attribute values bind the closest,
// then come script elements within the HTML source, then external style sheets and
// finally global (user-agent level) default properties.
type PropertySource uint8

// Values for property sources, used when adding style sheets.
const (
	Global    PropertySource = iota + 1 // "browser" globals
	Author                              // CSS author (stylesheet link)
	Script                              // <script> element
	Attribute                           // in an element's attribute(s)
)

// rootElement is a symbolic node to denote the body element of a future
// HTML document. AddStylesFor(...) with nil as a scope will replace it
// with this marker for scoping the complete document body.
var rootElement = &html.Node{Data: "root"}

// Internal helper for applying rules to an HTML node.
// In a first step it holds all the rules matching for an HTML node.
// In a second step it collects all the properties set in those rules,
// then orderes them by specifity.
type matchesList struct {
	matchingRules   []matchedRule
	propertiesTable []propertyPlusSpecifityType
}

// matchedRule retains the specificity of the strongest selector-list branch
// that matched this node. Unmatched branches must not increase its priority.
type matchedRule struct {
	rule Rule
	spec cascadia.Specificity
}

// Rule-matchings are collected from more than one stylesheet. Matching
// rules from these stylesheets will be merged to one list.
func (matches *matchesList) mergeMatchesWith(m *matchesList) *matchesList {
	if matches == nil {
		return m
	}
	if m != nil {
		for _, r := range m.matchingRules {
			matches.matchingRules = append(matches.matchingRules, r)
		}
	}
	return matches
}

// Rule-matchings have to be sorted by specifity. We'll sort the highest
// specifity up and won't overwrite earlier matches with later matches.

// sorter
type byHighestSpecifity []propertyPlusSpecifityType

// make specifities sortable by highest sp.spec
func (sp byHighestSpecifity) Len() int      { return len(sp) }
func (sp byHighestSpecifity) Swap(i, j int) { sp[i], sp[j] = sp[j], sp[i] }
func (sp byHighestSpecifity) Less(i, j int) bool {
	if sp[i].important != sp[j].important {
		return sp[i].important
	}
	_, inlineI := sp[i].rule.(localPseudoRuleType)
	_, inlineJ := sp[j].rule.(localPseudoRuleType)
	if inlineI != inlineJ {
		return inlineI
	}
	if sp[i].spec != sp[j].spec {
		return sp[j].spec.Less(sp[i].spec)
	}
	return sp[i].sourceOrder > sp[j].sourceOrder
}

// This is a small helper to print out a table with rule-matches for a node.
func (matches *matchesList) String() string {
	s := fmt.Sprintf("match of %d rules:\n", len(matches.matchingRules))
	s += "Src +-- Spec. --+------------- Key --------------+------- Value ---------------\n"
	for _, sp := range matches.propertiesTable {
		s += fmt.Sprintf("%3d | %9v | %30s | %s\n", sp.source, sp.spec, sp.propertyKey, sp.propertyValue)
	}
	return s
}

// FilterMatchesFor(node) iterates through all the rules relevant at this
// point and looks for rules matching the current HTML node h.
// The heavy lifting is done by cascadia. We have to 'compile' all rules
// and will cache compiled rules.
//
// Will return a slice of CSS rules matched for h.
func (rt *rulesTreeType) FilterMatchesFor(h *html.Node) *matchesList {
	matchingRules := make([]matchedRule, 0, 3)
	collect := func(sheets []stylesheetType, local bool) {
		for _, sheet := range sheets {
			if _, inline := sheet.stylesheet.(*localPseudoStylesheetType); inline && !local {
				continue
			}
			for _, rule := range sheet.stylesheet.Rules() {
				if spec, ok := rt.matchRuleForHTMLNode(h, rule); ok {
					matchingRules = append(matchingRules, matchedRule{rule: rule, spec: spec})
				}
			}
		}
	}
	// Slice order retains registration/rule order across sheets. The matched
	// rule index is a separate tie breaker, never part of selector specificity.
	collect(rt.StylesheetsForHTMLNode(rootElement), false)
	var scopes []*html.Node
	for node := h; node != nil; node = node.Parent {
		scopes = append(scopes, node)
	}
	for i := len(scopes) - 1; i >= 0; i-- {
		collect(rt.StylesheetsForHTMLNode(scopes[i]), scopes[i] == h)
	}
	return &matchesList{matchingRules: matchingRules}
}

func (rt *rulesTreeType) matchRuleForHTMLNode(h *html.Node, rule Rule) (cascadia.Specificity, bool) {
	var spec cascadia.Specificity
	selectorString := rule.Selector()
	if selectorString == "" { // style-attribute local for this HTML node
		return spec, true
	}
	cached, found := rt.selectors.Load(selectorString)
	if !found {
		selectors, err := cascadia.ParseGroup(selectorString)
		if err != nil {
			tracer().Errorf("CSS selector seems not to work: %s", selectorString)
			return spec, false
		}
		cached, _ = rt.selectors.LoadOrStore(selectorString, selectors)
	}
	matched := false
	for _, selector := range cached.(cascadia.SelectorGroup) {
		if selector.Match(h) {
			candidate := selector.Specificity()
			if !matched || spec.Less(candidate) {
				spec = candidate
			}
			matched = true
		}
	}
	return spec, matched
}

// SortProperties takes a slice of CSS rules (matched for an HTML node) and
// extracts all the properties set within the rules. These properties are
// then split into atomic properties, if they are compound properties
// (e.g.,
//
//	"margin" ⟹ "margin-top", "margin-right", ...
//
// Finally all property entries are sorted by specifity of the enclosing rule.
func (matches *matchesList) SortProperties(splitters []CompoundPropertiesSplitter) {
	var proptable []propertyPlusSpecifityType
	for _, match := range matches.matchingRules {
		rule := match.rule
		for _, propertyKey := range rule.Properties() {
			value := style.Property(rule.Value(propertyKey))
			props, err := splitCompoundProperty(splitters, propertyKey, value)
			if err == nil {
				//tracer().Debugf("%s is a compound style", propertyKey)
				for _, kv := range props {
					key := kv.Key
					val := kv.Value
					sp := propertyPlusSpecifityType{
						source: Author, rule: rule, propertyKey: key, propertyValue: val,
						important: rule.IsImportant(propertyKey), spec: match.spec, sourceOrder: len(proptable),
					}
					proptable = append(proptable, sp)
				}
			} else {
				sp := propertyPlusSpecifityType{
					source: Author, rule: rule, propertyKey: propertyKey, propertyValue: value,
					important: rule.IsImportant(propertyKey), spec: match.spec, sourceOrder: len(proptable),
				}
				proptable = append(proptable, sp)
			}
		}
	}
	if len(proptable) > 0 {
		sort.Stable(byHighestSpecifity(proptable))
		matches.propertiesTable = proptable
	}
	if tracer().GetTraceLevel() >= tracing.LevelDebug {
		tracer().Debugf(matches.String())
	}
}

// --- Specifity of rules -----------------------------------------------

type propertyPlusSpecifityType struct {
	source        PropertySource       // where the property has been defined
	rule          Rule                 // the rule containing the property definition
	propertyKey   string               // CSS property name
	propertyValue style.Property       // raw string value
	important     bool                 // marked as !IMPORTANT ?
	spec          cascadia.Specificity // IDs, classes/attributes/pseudo-classes, types
	sourceOrder   int                  // later rule wins only when specificity ties
}

// --- Style Property Groups --------------------------------------------

func (matches *matchesList) createStyleGroups(parent *tree.Node[*styledtree.StyNode]) *style.PropertyMap {
	//
	pmap := style.NewPropertyMap()
	done := make(map[string]bool, len(matches.propertiesTable))
	for _, pspec := range matches.propertiesTable { // for every specifity entry
		pspec.propertyKey = style.CanonicalPropertyKey(pspec.propertyKey)
		if done[pspec.propertyKey] {
			// already present in current properties map
			// this must be from previous set with higher specifity
			// => do nothing
			continue
		}
		groupname := style.GroupNameFromPropertyKey(pspec.propertyKey)
		group := pmap.Group(groupname)
		if group != nil {
			group.Set(pspec.propertyKey, pspec.propertyValue)
		} else {
			tracer().Infof("parent is %s, searching for prop group %s", parent, groupname)
			_, pg := findAncestorWithPropertyGroup(parent, groupname) // must succeed
			if pg == nil {
				panic(fmt.Sprintf("Cannot find ancestor with prop-group %s -- did you create global properties?", groupname))
			}
			// An explicit declaration must remain local even if its value equals
			// an ancestor's default. Reusing the ancestor also lets a later
			// declaration in this group mutate styles shared by other nodes.
			group = style.NewPropertyGroup(groupname)
			group.Parent = pg
			group.Set(pspec.propertyKey, pspec.propertyValue)
			pmap = pmap.AddAllFromGroup(group, true)
		}
		done[pspec.propertyKey] = true // remember we're done with this property
	}
	if pmap.Size() == 0 { // no property groups created, no properties set
		return nil
	}
	// if tracing.EngineTracer.GetTraceLevel() >= tracing.LevelDebug {
	// 	for _, v := range m {
	// 		tracing.EngineTracer.Debugf(v.String())
	// 	}
	// }
	return pmap
}

// --- Styled Node Tree -------------------------------------------------

// setupStyledNodeTree sets up the root nodes of the style tree.
// It creates a "root" node and a node for the HTML-document-node as its child.
func setupStyledNodeTree(domRoot *html.Node, defaults *style.PropertyMap) *tree.Node[*styledtree.StyNode] {
	//
	//rootNode := creator.StyleForHTMLNode(domRoot)
	rootNode := styledtree.NewNodeForHTMLNode(domRoot)
	//creator.SetStyles(rootNode, defaults)
	rootNode.Payload.SetStyles(defaults)
	//tracer().Debugf("UA node has styles = %s", creator.ToStyler(rootNode).ComputedStyles())
	//docNode := creator.StyleForHTMLNode(domRoot)
	docNode := styledtree.NewNodeForHTMLNode(domRoot)
	rootNode.AddChild(docNode)
	return docNode
}

// func findAncestorWithPropertyGroup(sn StyledNode, group string, builder StyledTreeBuilder) (StyledNode, *style.PropertyGroup) {
func findAncestorWithPropertyGroup(sn *tree.Node[*styledtree.StyNode], group string) (*tree.Node[*styledtree.StyNode], *style.PropertyGroup) {
	//
	var pg *style.PropertyGroup
	if sn == nil {
		tracer().Errorf("Search for ancestor with property group %s started with nil", group)
		return nil, nil
	}
	it := sn // start search at styled node itself, then proceed upwards
	last := sn
	for it != nil && pg == nil {
		styles := it.Payload.Styles()
		//styles := creator.ToStyler(it).Styles()
		if styles != nil {
			pg = styles.Group(group)
		}
		it = it.Parent()
		if it != nil {
			last = it
		}
	}
	// if it == nil {
	// 	tracer().Debugf("At root of tree searching for property group %s", group)
	// 	if pg == nil {
	// 		tracer().Errorf("Property group %s not found", group)
	// 		tracer().Debugf("Property map of last node %v =\n%s", last, creator.ToStyler(last).ComputedStyles())
	// 	}
	// }
	return last, pg
}

// Style gets things rolling. It styles an HTML parse tree, referred to by the root
// node, and returns a tree of styled nodes.
// For an explanation what's going on here, refer to
// https://hacks.mozilla.org/2017/08/inside-a-super-fast-css-engine-quantum-css-aka-stylo/
// and
// https://limpet.net/mbrubeck/2014/08/23/toy-layout-engine-4-style.html
//
// If either dom or creator are nil, no tree is returned (but an error).
func (cssom CSSOM) Style(dom *html.Node) (*tree.Node[*styledtree.StyNode], error) {
	if dom == nil {
		return nil, errors.New("Nothing to style: empty document")
	}
	// if creator == nil {
	// 	return nil, errors.New("Cannot style: no builder to create styles nodes")
	// }
	if cssom.rulesTree.Empty() {
		tracer().Infof("Styling HTML tree without having any CSS rules")
	}
	tracer().Debugf("--- Creating style nodes for HTML nodes ----")
	styledRootNode := setupStyledNodeTree(dom, cssom.defaultProperties)
	walker := tree.NewWalker(styledRootNode) // create a concurrent tree walker
	createNodes := func(node *tree.Node[*styledtree.StyNode], parent *tree.Node[*styledtree.StyNode],
		pos int) (*tree.Node[*styledtree.StyNode], error) {
		//
		return createStyledChildren(node, cssom.rulesTree) // provide closure with style creator
	}
	future := walker.TopDown(createNodes).Promise() // build the style tree
	if _, err := future(); err != nil {
		tracer().Errorf("Error while creating styled tree: %v", err)
		return nil, err
	}
	// TODO: Possibly do not sync after creating the nodes, but rather
	// continue with styling as a walker.Filter(...).
	// It then is possible for a child to overtake its parent, but this
	// is probably acceptable: In the worst case a property group will
	// not point to a possible group of its parent, but rather to an
	// ancestor (and the parent may point to the same ancestor). This is
	// a loss of space efficiency, but we may gain performance by
	// overlapping the operations.
	tracer().Debugf("--- Now styling newly created nodes --------")
	walker = tree.NewWalker(styledRootNode)
	createStyles := func(node *tree.Node[*styledtree.StyNode], parent *tree.Node[*styledtree.StyNode], pos int) (*tree.Node[*styledtree.StyNode], error) {
		return createStylesForNode(node, cssom.rulesTree, cssom.compoundSplitters)
	}
	future = walker.TopDown(createStyles).Promise() // build the style tree
	if _, err := future(); err != nil {
		tracer().Errorf("Error while creating style properties: %v", err)
		return nil, err
	}
	return styledRootNode, nil
}

// Pre-condition: sn has been styled and points to an HTML node.
// Now iterate through the HTML children and create styled nodes for each.
func createStyledChildren(parent *tree.Node[*styledtree.StyNode], rulesTree *rulesTreeType) (*tree.Node[*styledtree.StyNode], error) {
	//
	domnode := parent.Payload
	//domnode := creator.ToStyler(parent)
	tracer().Debugf("Input node = %v, creating styled children", domnode)
	h := domnode.HTMLNode()
	if h.Type == html.ElementNode || h.Type == html.DocumentNode {
		ch := h.FirstChild
		for ch != nil {
			if ch.DataAtom == atom.Style { // <style> element
				tracer().Infof("<style> nodes have to be extracted in advance")
			} else if isInDom(ch.Type, ch.DataAtom) {
				//} else if isStylable(ch.DataAtom) {
				//sn := creator.StyleForHTMLNode(ch)
				sn := styledtree.NewNodeForHTMLNode(ch)
				parent.AddChild(sn) // sn will be sent to next pipeline stage
				if styleAttr := getStyleAttribute(ch); styleAttr != nil {
					// attach local style attributes
					rulesTree.StoreStylesheetForHTMLNode(ch, styleAttr, Attribute)
				}
			}
			ch = ch.NextSibling
		}
	} else if h.Type == html.TextNode {
		// do not send text node to next pipeline stage
		return nil, nil
	}
	return parent, nil
}

func isInDom(nt html.NodeType, a atom.Atom) bool {
	if nt == html.ElementNode || nt == html.DocumentNode {
		return true
	}
	if nt == html.TextNode {
		return true
	}
	return false
}

func isStylable(a atom.Atom) bool {
	switch a {
	case atom.A, atom.Address, atom.Acronym, atom.Article, atom.Aside,
		atom.B, atom.Blink, atom.Blockquote, atom.Body, atom.Br,
		atom.Button, atom.Label, atom.Canvas, atom.Caption,
		atom.Code, atom.Content, atom.Div, atom.Em, atom.Figcaption,
		atom.Figure, atom.Footer, atom.Form, atom.Frame, atom.Hr,
		atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Html,
		atom.I, atom.Img, atom.Input, atom.Li, atom.Main, atom.Math,
		atom.Menu, atom.Menuitem, atom.Nav, atom.Ol, atom.Option,
		atom.P, atom.Picture, atom.Pre, atom.Poster, atom.Q, atom.S,
		atom.Section, atom.Span, atom.Spacer, atom.Strong, atom.Summary,
		atom.Svg, atom.Sup, atom.Table, atom.Td, atom.Tr, atom.Th,
		atom.Textarea, atom.Tfoot, atom.Title, atom.Ul, atom.Video:
		return true
	}
	return false
}

func createStylesForNode(node *tree.Node[*styledtree.StyNode], rulesTree *rulesTreeType,
	splitters []CompoundPropertiesSplitter) (*tree.Node[*styledtree.StyNode], error) {
	//
	//styler := creator.ToStyler(node)
	h := node.Payload.HTMLNode()
	//h := styler.HTMLNode()
	if h.Type == html.DocumentNode || h.Type == html.ElementNode {
		if isStylable(h.DataAtom) {
			matchlist := rulesTree.FilterMatchesFor(h)
			if matchlist != nil && len(matchlist.matchingRules) != 0 {
				matchlist.SortProperties(splitters)
				pmap := matchlist.createStyleGroups(node.Parent())
				tracer().Debugf("Setting styles for node %v =\n%s", node, pmap)
				//creator.SetStyles(node, pmap)
				node.Payload.SetStyles(pmap)
			} else {
				tracer().Debugf("Node %v matched no style rules", node)
			}
		}
		return node, nil
	}
	return nil, nil
}

// --- Helpers ----------------------------------------------------------

var errNoSuchCompoundProperty = errors.New("No such compound property")

// Try to split up a property (which may or may not be a compound
// property) using a set of splitter functions.
// Return a slice of key-value pairs or nil.
func splitCompoundProperty(splitters []CompoundPropertiesSplitter,
	key string, value style.Property) ([]style.KeyValue, error) {
	for _, splitter := range splitters {
		kv, err := splitter(key, value)
		if err == nil {
			return kv, nil
		}
	}
	return nil, errNoSuchCompoundProperty
}

// Which HTML node type needs a corresponding styled node?
func createsStyledNode(nodeType html.NodeType) bool {
	if nodeType == html.ElementNode || nodeType == html.TextNode {
		return true
	}
	return false
}

// Helper to find nodes matching a predicate. Currently works recursive.
// Returns a node or nil.
func findNode(node *html.Node, matcher func(n *html.Node) bool) *html.Node {
	if node == nil {
		return nil
	}
	if matcher(node) {
		return node
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if f := findNode(c, matcher); f != nil {
			return f
		}
	}
	return nil
}

func findThisNode(tree *html.Node, nodeToFind *html.Node) *html.Node {
	return findNode(tree, func(n *html.Node) bool {
		return n == nodeToFind
	})
}

func findBodyElement(tree *html.Node) *html.Node {
	return findNode(tree, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "body"
	})
}

// shorten a string
func shorten(s string) string {
	if len(s) > 10 {
		return s[:10] + "..."
	}
	return s
}

// small helper to debug-print out a node. TODO
func nodePath(node *html.Node) string {
	s := ""
	if node.Type == html.TextNode {
		s += "(text)"
	} else if node.Type == html.ElementNode {
		s += fmt.Sprintf("%s", node.Data)
	} else {
		s += "(unknown)"
	}
	return s
}

// --- Local pseudo rules for style-attributes --------------------------

func getStyleAttribute(h *html.Node) *localPseudoStylesheetType {
	if h != nil && h.Type == html.ElementNode {
		for _, attr := range h.Attr {
			if attr.Key == "style" {
				return &localPseudoStylesheetType{newLocalPseudoRule(attr.Val)}
			}
		}
	}
	return nil
}

type localPseudoStylesheetType struct {
	rule localPseudoRuleType
}

type localDeclaration struct {
	style.KeyValue
	important bool
}
type localPseudoRuleType []localDeclaration

// Tokenize before splitting so strings, URLs and function arguments retain delimiters.
func newLocalPseudoRule(styleAttr string) localPseudoRuleType {
	var result localPseudoRuleType
	var tokens []*scanner.Token
	depth := 0
	flush := func() {
		defer func() { tokens = nil }()
		significant := make([]*scanner.Token, 0, len(tokens))
		for _, token := range tokens {
			if token.Type != scanner.TokenS && token.Type != scanner.TokenComment {
				significant = append(significant, token)
			}
		}
		if len(significant) < 3 || significant[0].Type != scanner.TokenIdent || significant[1].Value != ":" {
			return
		}
		key := significant[0].Value
		if !strings.HasPrefix(key, "--") {
			key = style.CanonicalPropertyKey(strings.ToLower(key))
		}
		important := false
		end := len(tokens)
		if len(significant) >= 4 && significant[len(significant)-1].Type == scanner.TokenIdent && strings.EqualFold(significant[len(significant)-1].Value, "important") && significant[len(significant)-2].Value == "!" {
			important = true
			for i, t := range tokens {
				if t == significant[len(significant)-2] {
					end = i
					break
				}
			}
		}
		var value strings.Builder
		started := false
		for _, t := range tokens[:end] {
			if !started {
				if t == significant[1] {
					started = true
				}
				continue
			}
			if t.Type == scanner.TokenComment {
				value.WriteByte(' ')
			} else {
				value.WriteString(t.Value)
			}
		}
		v := strings.TrimSpace(value.String())
		if v == "" {
			return
		}
		for i, d := range result {
			if d.Key == key {
				if d.important && !important {
					return
				}
				result = append(result[:i], result[i+1:]...)
				break
			}
		}
		result = append(result, localDeclaration{style.KeyValue{Key: key, Value: style.Property(v)}, important})
	}
	lexer := scanner.New(styleAttr)
	for {
		token := lexer.Next()
		if token.Type == scanner.TokenError {
			break
		}
		if token.Type == scanner.TokenEOF {
			if depth == 0 {
				flush()
			}
			break
		}
		if token.Type == scanner.TokenChar && token.Value == ";" && depth == 0 {
			flush()
			continue
		}
		if token.Type == scanner.TokenFunction || (token.Type == scanner.TokenChar && (token.Value == "(" || token.Value == "[" || token.Value == "{")) {
			depth++
		}
		if token.Type == scanner.TokenChar && (token.Value == ")" || token.Value == "]" || token.Value == "}") {
			if depth > 0 {
				depth--
			}
		}
		tokens = append(tokens, token)
	}
	return result
}

func (pseudorule localPseudoRuleType) Selector() string {
	return ""
}

func (pseudorule localPseudoRuleType) Properties() []string {
	var p []string
	for _, kv := range pseudorule {
		p = append(p, kv.Key)
	}
	return p
}
func (pseudorule localPseudoRuleType) Value(key string) style.Property {
	key = style.CanonicalPropertyKey(key)
	for _, kv := range pseudorule {
		if key == kv.Key {
			return kv.Value
		}
	}
	return style.NullStyle
}

func (pseudorule localPseudoRuleType) IsImportant(key string) bool {
	key = style.CanonicalPropertyKey(key)
	for _, d := range pseudorule {
		if d.Key == key {
			return d.important
		}
	}
	return false
}

func (pseudosheet *localPseudoStylesheetType) AppendRules(s StyleSheet) {
	for _, r := range s.Rules() {
		for _, k := range r.Properties() {
			pseudosheet.rule = append(pseudosheet.rule, localDeclaration{KeyValue: style.KeyValue{
				Key:   k,
				Value: r.Value(k),
			}, important: r.IsImportant(k)})
		}
	}
}

func (pseudosheet *localPseudoStylesheetType) Empty() bool {
	return false
}

func (pseudosheet *localPseudoStylesheetType) Rules() []Rule {
	return []Rule{pseudosheet.rule}
}
