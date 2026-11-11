package style

/*
License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>

*/

import (
	"fmt"
	"strings"

	"github.com/npillmayer/schuko/tracing"
)

// tracer will return a tracer. We are tracing to 'tyse.dom'
func tracer() tracing.Trace {
	return tracing.Select("tyse.dom")
}

// Property is a raw value for a CSS property. For example, with
//
//	color: black
//
// a property value of "black" is set. The main purpose of wrapping
// the raw string value into type Property is to provide a set of
// convenient type conversion functions and other helpers.
type Property string

// NullStyle is an empty property value.
const NullStyle Property = ""

func (p Property) String() string {
	return string(p)
}

// IsInitial denotes if a property is of inheritence-type "initial"
func (p Property) IsInitial() bool {
	return p == "initial"
}

// IsInherit denotes if a property is of inheritence-type "inherit"
func (p Property) IsInherit() bool {
	return p == "inherit"
}

// IsEmpty checks wether a property is empty, i.e. the null-string.
func (p Property) IsEmpty() bool {
	return p == ""
}

// KeyValue is a container for a style property.
type KeyValue struct {
	Key   string
	Value Property
}

// --- CSS Property Groups ----------------------------------------------
//
// Caching is currently not implemented.

// PropertyGroup is a collection of propertes sharing a common topic.
// CSS knows a whole lot of properties. We split them up into organisatorial
// groups.
//
// The mapping of property into groups is documented with
// GroupNameFromPropertyKey[...].
type PropertyGroup struct {
	name      string
	Parent    *PropertyGroup
	propsDict map[string]Property
}

// NewPropertyGroup creates a new empty property group, given its name.
func NewPropertyGroup(groupname string) *PropertyGroup {
	pg := &PropertyGroup{}
	pg.name = groupname
	return pg
}

// Name returns the name of the property group. Once named (during
// construction, property groups may not be renamed.
func (pg *PropertyGroup) Name() string {
	return pg.name
}

// Stringer for property groups; used for debugging.
func (pg *PropertyGroup) String() string {
	s := "[" + pg.name + "] =\n"
	for k, v := range pg.propsDict {
		s += fmt.Sprintf("  %s = %s\n", k, v)
	}
	return s
}

// Properties returns all properties of a group.
func (pg *PropertyGroup) Properties() []KeyValue {
	i := 0
	r := make([]KeyValue, len(pg.propsDict))
	for k, v := range pg.propsDict {
		r[i] = KeyValue{k, v}
		i++
	}
	return r
}

// IsSet is a predicated wether a property is set within this group.
func (pg *PropertyGroup) IsSet(key string) bool {
	key = CanonicalPropertyKey(key)
	if pg.propsDict == nil {
		return false
	}
	v, ok := pg.propsDict[key]
	return ok && !v.IsEmpty()
}

// Get a property's value.
//
// Property values preserve their original case.
func (pg *PropertyGroup) Get(key string) (Property, bool) {
	key = CanonicalPropertyKey(key)
	if pg.propsDict == nil {
		return NullStyle, false
	}
	p, ok := pg.propsDict[key]
	return p, ok
}

// Set a property's value. Overwrites an existing value, if present.
//
// Property values preserve their original case.
func (pg *PropertyGroup) Set(key string, p Property) {
	key = CanonicalPropertyKey(key)
	if pg.propsDict == nil {
		pg.propsDict = make(map[string]Property)
	}
	pg.propsDict[key] = p
}

// Add a property's value. Does not overwrite an existing value, i.e., does nothing
// if a value is already set.
func (pg *PropertyGroup) Add(key string, p Property) {
	key = CanonicalPropertyKey(key)
	if pg.propsDict == nil {
		pg.propsDict = make(map[string]Property)
	}
	_, exists := pg.propsDict[key]
	if !exists {
		pg.propsDict[key] = p
	}
}

// ForkOnProperty creates a new PropertyGroup, pre-filled with a given property.
// If 'cascade' is true, the new PropertyGroup will be
// linking to the ancesting PropertyGroup containing this property.
func (pg *PropertyGroup) ForkOnProperty(key string, p Property, cascade bool) (*PropertyGroup, bool) {
	var ancestor *PropertyGroup
	if cascade {
		ancestor = pg.Cascade(key)
		if ancestor != nil {
			p2, _ := ancestor.Get(key)
			if p2 == p {
				return pg, false
			}
		}
	}
	npg := NewPropertyGroup(pg.name)
	npg.Parent = ancestor
	//npg.signature = pg.signature
	npg.Set(key, p)
	return npg, true
}

// Cascade finds the first group containing a nonempty value for key.
// It returns nil if the key is absent, including on a nil receiver.
func (pg *PropertyGroup) Cascade(key string) *PropertyGroup {
	it := pg
	for it != nil && !it.IsSet(key) { // stopper is default partial
		it = it.Parent
	}
	return it
}

// GroupNameFromPropertyKey returns the style property group name for a
// style property.
// Example:
//
//	GroupNameFromPropertyKey("margin-top") => "Margins"
//
// Unknown style property keys will return a group name of "X".
func GroupNameFromPropertyKey(key string) string {
	key = CanonicalPropertyKey(key)
	if meta, ok := propertyRegistry[key]; ok {
		return meta.group
	}
	return PGX
}

// Symbolic names for string literals, denoting PropertyGroups.
const (
	PGMargins   = "Margins"
	PGPadding   = "Padding"
	PGBorder    = "Border"
	PGDimension = "Dimension"
	PGDisplay   = "Display"
	PGRegion    = "Region"
	PGColor     = "Color"
	PGText      = "Text"
	PGX         = "X"
)

// IsCascading reports whether an absent property inherits by default.
// Unknown extension properties do not inherit implicitly.
func IsCascading(key string) bool { return propertyRegistry[CanonicalPropertyKey(key)].inherited }

// SplitCompoundProperty splits up a shortcut property into its individual
// components. Returns a slice of key-value pairs representing the
// individual (fine grained) style properties.
// Example:
//
//	SplitCompountProperty("padding", "3px")
//
// will return
//
//	"padding-top"    => "3px"
//	"padding-right"  => "3px"
//	"padding-bottom" => "3px"
//	"padding-left  " => "3px"
//
// For the logic behind this, refer to e.g.
// https://www.w3schools.com/css/css_padding.asp .
func SplitCompoundProperty(key string, value Property) ([]KeyValue, error) {
	fields := strings.Fields(value.String())
	switch key {
	case "margin", "margins": // retain the historical alias
		return feazeCompound4("margin", "", fourDirs, fields)
	case "padding":
		return feazeCompound4("padding", "", fourDirs, fields)
	case "border-color":
		return feazeCompound4("border", "color", fourDirs, fields)
	case "border-width":
		return feazeCompound4("border", "width", fourDirs, fields)
	case "border-style":
		return feazeCompound4("border", "style", fourDirs, fields)
	case "border-radius":
		parts := strings.Split(value.String(), "/")
		if len(parts) > 2 {
			return nil, fmt.Errorf("border-radius accepts at most one slash")
		}
		horizontal, err := feazeCompound4("border", "radius", fourCorners, strings.Fields(parts[0]))
		if err != nil {
			return nil, err
		}
		if len(parts) == 2 {
			vertical, err := feazeCompound4("border", "radius", fourCorners, strings.Fields(parts[1]))
			if err != nil {
				return nil, err
			}
			for i := range horizontal {
				horizontal[i].Value += " " + vertical[i].Value
			}
		}
		return horizontal, nil
	}
	return nil, fmt.Errorf("not recognized as compound property: %s", key)
}

// CSS logic to distribute individual values from compound shortcuts is as
// follows: https://www.w3schools.com/css/css_border.asp
func feazeCompound4(pre string, suf string, dirs [4]string, fields []string) ([]KeyValue, error) {
	l := len(fields)
	if l == 0 || l > 4 {
		return nil, fmt.Errorf("expecting 1-4 values for %s-%s", pre, suf)
	}
	r := make([]KeyValue, 4)
	r[0] = KeyValue{p(pre, suf, dirs[0]), Property(fields[0])}
	if l >= 2 {
		r[1] = KeyValue{p(pre, suf, dirs[1]), Property(fields[1])}
		if l >= 3 {
			r[2] = KeyValue{p(pre, suf, dirs[2]), Property(fields[2])}
			if l == 4 {
				r[3] = KeyValue{p(pre, suf, dirs[3]), Property(fields[3])}
			} else {
				r[3] = KeyValue{p(pre, suf, dirs[3]), Property(fields[1])}
			}
		} else {
			r[2] = KeyValue{p(pre, suf, dirs[2]), Property(fields[0])}
			r[3] = KeyValue{p(pre, suf, dirs[3]), Property(fields[1])}
		}
	} else {
		r[1] = KeyValue{p(pre, suf, dirs[1]), Property(fields[0])}
		r[2] = KeyValue{p(pre, suf, dirs[2]), Property(fields[0])}
		r[3] = KeyValue{p(pre, suf, dirs[3]), Property(fields[0])}
	}
	return r, nil
}

var fourDirs = [4]string{"top", "right", "bottom", "left"}
var fourCorners = [4]string{"top-left", "top-right", "bottom-right", "bottom-left"}

func p(prefix string, suffix string, tag string) string {
	if suffix == "" {
		return prefix + "-" + tag
	}
	if prefix == "" {
		return tag + "-" + suffix
	}
	return prefix + "-" + tag + "-" + suffix
}

// --- Property Map -----------------------------------------------------

// PropertyMap holds CSS properties. nil is a legal (empty) property map.
// A property map is the entity styling a DOM node: a DOM node links to a property map,
// which contains zero or more property groups. Property maps may share property groups.
type PropertyMap struct {
	defaults  bool
	overrides map[string]Property
	// As CSS defines a whole lot of properties, we segment them into logical groups.
	m map[string]*PropertyGroup // into struct to make it opaque for clients
}

// NewPropertyMap returns a new empty property map.
func NewPropertyMap() *PropertyMap {
	return &PropertyMap{}
}

func (pmap *PropertyMap) String() string {
	s := "Property Map = {\n"
	for _, v := range pmap.m {
		s += v.String()
	}
	s += "}"
	return s
}

// Size returns the number of property groups.
func (pmap *PropertyMap) Size() int {
	if pmap == nil {
		return 0
	}
	return len(pmap.m)
}

// Group returns the property group for a group name or nil.
func (pmap *PropertyMap) Group(groupname string) *PropertyGroup {
	if pmap == nil {
		return nil
	}
	group := pmap.m[groupname]
	return group
}

// Property returns a style property value, together with an indicator
// wether it has been found in the properties map.
// No cascading is performed
func (pmap *PropertyMap) Property(key string) (Property, bool) {
	groupname := GroupNameFromPropertyKey(key)
	group := pmap.Group(groupname)
	if group == nil {
		return NullStyle, false
	}
	return group.Get(key)
}

// AddAllFromGroup transfers all style properties from a property group
// to a property map. If overwrite is set, existing style property values
// will be overwritten, otherwise only new values are set.
//
// If the property map does not yet contain a group of this kind, it will
// simply set this group (instead of copying values).
func (pmap *PropertyMap) AddAllFromGroup(group *PropertyGroup, overwrite bool) *PropertyMap {
	if pmap == nil {
		pmap = NewPropertyMap()
	}
	if pmap.m == nil {
		pmap.m = make(map[string]*PropertyGroup)
	}
	g := pmap.Group(group.name)
	if g == nil {
		pmap.m[group.name] = group
	} else {
		for k, v := range group.propsDict {
			if overwrite {
				g.Set(k, v)
			} else {
				g.Add(k, v)
			}
		}
	}
	return pmap
}

// Add adds a property to this property map. A zero-value map is initialized;
// a nil receiver is left unchanged. For example,
//
//	pm.Add("funny-margin", "big")
func (pmap *PropertyMap) Add(key string, value Property) {
	if pmap == nil {
		return
	}
	if pmap.m == nil {
		pmap.m = make(map[string]*PropertyGroup)
	}
	groupname := GroupNameFromPropertyKey(key)
	group, found := pmap.m[groupname]
	if !found {
		group = NewPropertyGroup(groupname)
		pmap.m[groupname] = group
	}
	group.Set(key, value)
}

// CanonicalPropertyKey returns the storage and cascade key for a property alias.
// Custom property names are unchanged.
func CanonicalPropertyKey(key string) string {
	if key == "word-wrap" {
		return "overflow-wrap"
	}
	return key
}
