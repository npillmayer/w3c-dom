package cssom

import (
	"github.com/npillmayer/w3c-dom/style"
	"golang.org/x/net/html"
	"testing"
)

func TestInlinePriorityAndIndependentProperties(t *testing.T) {
	inline := newLocalPseudoRule("color:green")
	list := matchesList{propertiesTable: []propertyPlusSpecifityType{
		{rule: inline, propertyKey: "color", propertyValue: "green", spec: [3]int{}},
		{rule: nil, propertyKey: "color", propertyValue: "blue", spec: [3]int{1, 0, 0}},
		{rule: nil, propertyKey: "font-size", propertyValue: "10px", spec: [3]int{0, 0, 1}},
	}}
	if !byHighestSpecifity(list.propertiesTable).Less(0, 1) {
		t.Fatal("inline lost to selector")
	}
	// Higher priority duplicate must not prevent independent lower rules.
	root := setupStyledNodeTree(&html.Node{Type: html.DocumentNode}, NewCSSOM([]style.KeyValue{{Key: "font-size", Value: "medium"}}).defaultProperties)
	props := list.createStyleGroups(root.Parent())
	if value, ok := props.Property("font-size"); !ok || value != style.Property("10px") {
		t.Fatalf("size lost: %q", value)
	}
}

func TestInlineShorthandOrderAndValueCase(t *testing.T) {
	for _, tc := range []struct{ input, key, want string }{
		{`margin:1px;margin-left:2px`, "margin-left", "2px"},
		{`margin-left:2px;margin:1px`, "margin-left", "1px"},
		{`margin:1px!important;margin-left:2px`, "margin-left", "1px"},
		{`content:"Hello;World"`, "content", `"Hello;World"`},
	} {
		list := matchesList{matchingRules: []matchedRule{{rule: newLocalPseudoRule(tc.input)}}}
		list.SortProperties([]CompoundPropertiesSplitter{style.SplitCompoundProperty})
		root := setupStyledNodeTree(&html.Node{Type: html.DocumentNode}, NewCSSOM(nil).defaultProperties)
		props := list.createStyleGroups(root.Parent())
		if got, _ := props.Property(tc.key); string(got) != tc.want {
			t.Errorf("%s = %q", tc.input, got)
		}
	}
}
