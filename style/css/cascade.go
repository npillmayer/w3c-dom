package css

import (
	"fmt"

	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/styledtree"
)

// GetCascadedProperty gets the value of a property. The search cascades to
// parent property maps, if available.
//
// Clients will usually call GetProperty(…) instead as this will respect
// CSS semantics for inherited properties.
//
// GetCascadedProperty returns NullStyle and an error if node is nil or no
// nonempty value is found in the node, its property groups, or its ancestors.
func GetCascadedProperty(node *styledtree.StyNode, key string) (style.Property, error) {
	groupname := style.GroupNameFromPropertyKey(key)
	for node != nil {
		if group := node.Styles().Group(groupname).Cascade(key); group != nil {
			p, _ := group.Get(key)
			return p, nil
		}
		node = styledtree.Node(node.Parent())
	}
	return style.NullStyle, fmt.Errorf("no cascaded value for property %q", key)
}

// GetProperty resolves inheritance, explicit inherit/initial/unset, and defaults.
// Unknown absent properties return NullStyle; nil nodes return an error.
// Relative lengths and other layout-dependent values are not computed here.
func GetProperty(node *styledtree.StyNode, key string) (style.Property, error) {
	if node == nil {
		return style.NullStyle, fmt.Errorf("cannot get property %q from a nil node", key)
	}
	return node.GetPropertyValue(key, node.Styles()), nil
}

// GetLocalProperty returns a style property value, if it is set locally
// for a styled node's property map. No cascading is performed.
func GetLocalProperty(pmap *style.PropertyMap, key string) style.Property {
	groupname := style.GroupNameFromPropertyKey(key)
	var group *style.PropertyGroup
	group = pmap.Group(groupname)
	if group == nil {
		return style.NullStyle
	}
	p, _ := group.Get(key)
	return p
}
