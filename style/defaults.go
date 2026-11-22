package style

import (
	"golang.org/x/net/html"
	"strings"
)

// propertyMetadata is the single registry for the supported property subset.
// This is not a full CSS registry. The print UA chooses black as its foreground.
type propertyMetadata struct {
	group     string
	initial   Property
	inherited bool
}

var propertyRegistry = map[string]propertyMetadata{
	"box-sizing":   {PGDimension, "content-box", false},
	"writing-mode": {PGText, "horizontal-tb", true},
	// CSS leaves the initial family to the UA; this module chooses generic serif.
	"font-family":                {PGText, "serif", true},
	"font-size":                  {PGText, "medium", true},
	"font-weight":                {PGText, "normal", true},
	"font-style":                 {PGText, "normal", true},
	"unicode-bidi":               {PGText, "normal", false},
	"margin-top":                 {PGMargins, "0", false},
	"margin-left":                {PGMargins, "0", false},
	"margin-right":               {PGMargins, "0", false},
	"margin-bottom":              {PGMargins, "0", false},
	"padding-top":                {PGPadding, "0", false},
	"padding-left":               {PGPadding, "0", false},
	"padding-right":              {PGPadding, "0", false},
	"padding-bottom":             {PGPadding, "0", false},
	"border-top-color":           {PGBorder, "currentcolor", false},
	"border-left-color":          {PGBorder, "currentcolor", false},
	"border-right-color":         {PGBorder, "currentcolor", false},
	"border-bottom-color":        {PGBorder, "currentcolor", false},
	"border-top-width":           {PGBorder, "medium", false},
	"border-left-width":          {PGBorder, "medium", false},
	"border-right-width":         {PGBorder, "medium", false},
	"border-bottom-width":        {PGBorder, "medium", false},
	"border-top-style":           {PGBorder, "none", false},
	"border-left-style":          {PGBorder, "none", false},
	"border-right-style":         {PGBorder, "none", false},
	"border-bottom-style":        {PGBorder, "none", false},
	"border-top-left-radius":     {PGBorder, "0", false},
	"border-top-right-radius":    {PGBorder, "0", false},
	"border-bottom-left-radius":  {PGBorder, "0", false},
	"border-bottom-right-radius": {PGBorder, "0", false},
	"width":                      {PGDimension, "auto", false},
	"height":                     {PGDimension, "auto", false},
	"min-width":                  {PGDimension, "auto", false},
	"min-height":                 {PGDimension, "auto", false},
	"max-width":                  {PGDimension, "none", false},
	"max-height":                 {PGDimension, "none", false},
	"flow-from":                  {PGRegion, "none", false},
	"flow-into":                  {PGRegion, "none", false},
	"display":                    {PGDisplay, "inline", false},
	"float":                      {PGDisplay, "none", false},
	"visibility":                 {PGDisplay, "visible", true},
	"position":                   {PGDisplay, "static", false},
	"color":                      {PGColor, "black", true},
	"background-color":           {PGColor, "transparent", false},
	"direction":                  {PGText, "ltr", true},
	"white-space":                {PGText, "normal", true},
	"word-spacing":               {PGText, "normal", true},
	"letter-spacing":             {PGText, "normal", true},
	"word-break":                 {PGText, "normal", true},
	"overflow-wrap":              {PGText, "normal", true},
	"hyphens":                    {PGText, "manual", true},
	"top":                        {PGDimension, "auto", false},
	"right":                      {PGDimension, "auto", false},
	"bottom":                     {PGDimension, "auto", false},
	"left":                       {PGDimension, "auto", false},
	"cursor":                     {PGText, "auto", true},
	"line-height":                {PGText, "normal", true},
	"quotes":                     {PGText, "auto", true},
	"list-style-type":            {PGText, "disc", true},
	"list-style-position":        {PGText, "outside", true},
	"list-style-image":           {PGText, "none", true},
}

// InitialProperty returns the initial value, independent of element or UA overrides.
// Unknown extension properties have no initial value.
func InitialProperty(key string) Property { return propertyRegistry[CanonicalPropertyKey(key)].initial }

// GetUserAgentDefaultProperty returns element defaults for the supported subset.
func GetUserAgentDefaultProperty(node *html.Node, key string) Property {
	if key == "display" {
		return DisplayPropertyForHTMLNode(node)
	}
	return InitialProperty(key)
}

// DisplayPropertyForHTMLNode returns the default `display` CSS property for an HTML node.
func DisplayPropertyForHTMLNode(node *html.Node) Property {
	if node == nil {
		return "none"
	}
	if node.Type == html.DocumentNode {
		return "block"
	}
	if node.Type != html.ElementNode {
		tracer().Debugf("cannot get display-property for non-element")
		return "none"
	}
	// HTML element names are ASCII case-insensitive. Unknown/custom elements
	// use CSS's initial inline display rather than an invented block default.
	switch strings.ToLower(node.Data) {
	case "head", "title", "base", "link", "meta", "style", "script", "template":
		return "none"
	case "html", "body", "p", "address", "article", "aside", "blockquote", "div",
		"figcaption", "figure", "footer", "form", "header", "hr", "main", "nav",
		"h1", "h2", "h3", "h4", "h5", "h6", "ol", "ul", "menu", "pre", "section",
		"dl", "dt", "dd", "fieldset", "details":
		return "block"
	case "li", "summary":
		return "list-item"
	case "button", "input", "select", "textarea":
		return "inline-block"
	case "table":
		return "table"
	case "caption":
		return "table-caption"
	case "tr":
		return "table-row"
	case "td", "th":
		return "table-cell"
	case "thead":
		return "table-header-group"
	case "tbody":
		return "table-row-group"
	case "tfoot":
		return "table-footer-group"
	case "col":
		return "table-column"
	case "colgroup":
		return "table-column-group"
	}
	return "inline"
}

// InitializeDefaultPropertyValues builds defaults from the property registry.
// Additional values override UA defaults in the property's own group.
func InitializeDefaultPropertyValues(additionalProps []KeyValue) *PropertyMap {
	pm := NewPropertyMap()
	pm.defaults = true
	pm.overrides = make(map[string]Property)
	pm = pm.AddAllFromGroup(NewPropertyGroup(PGX), true)
	for key, meta := range propertyRegistry {
		pm.Add(key, meta.initial)
	}
	for _, kv := range additionalProps {
		pm.Add(kv.Key, kv.Value)
		pm.overrides[CanonicalPropertyKey(kv.Key)] = kv.Value
	}
	return pm
}

// IsDefaults reports whether this map was constructed as user-agent defaults.
func (pm *PropertyMap) IsDefaults() bool { return pm != nil && pm.defaults }

// DefaultProperty applies configured UA overrides before element-specific defaults.
func (pm *PropertyMap) DefaultProperty(node *html.Node, key string) Property {
	if pm != nil {
		if value, ok := pm.overrides[CanonicalPropertyKey(key)]; ok {
			return value
		}
	}
	return GetUserAgentDefaultProperty(node, key)
}
