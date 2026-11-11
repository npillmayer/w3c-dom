package style

import "golang.org/x/net/html"

// propertyMetadata is the single registry for the supported property subset.
// "default" remains a renderer sentinel for colors; this is not a full CSS registry.
type propertyMetadata struct {
	group     string
	initial   Property
	inherited bool
}

var propertyRegistry = map[string]propertyMetadata{
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
	"border-top-color":           {PGBorder, "black", false},
	"border-left-color":          {PGBorder, "black", false},
	"border-right-color":         {PGBorder, "black", false},
	"border-bottom-color":        {PGBorder, "black", false},
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
	"min-width":                  {PGDimension, "none", false},
	"min-height":                 {PGDimension, "none", false},
	"max-width":                  {PGDimension, "none", false},
	"max-height":                 {PGDimension, "none", false},
	"flow-from":                  {PGRegion, "none", false},
	"flow-into":                  {PGRegion, "none", false},
	"display":                    {PGDisplay, "inline", false},
	"float":                      {PGDisplay, "none", false},
	"visibility":                 {PGDisplay, "visible", true},
	"position":                   {PGDisplay, "static", false},
	"color":                      {PGColor, "default", true},
	"background-color":           {PGColor, "default", false},
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
	switch node.Data {
	case "head":
		return "none"
	case "p":
		return "block-inline"
	case "html", "aside", "body", "div", "h1", "h2", "h3",
		"h4", "h5", "h6", "it", "ol", "section",
		"ul":
		return "block"
	case "i", "b", "span", "strong":
		return "inline"
	}
	tracer().Infof("unknown HTML element %s/%d will be set to display: block",
		node.Data, node.Type)
	return "block"
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
