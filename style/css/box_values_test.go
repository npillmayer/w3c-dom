package css_test

import (
	"image/color"
	"testing"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/styledtree"
)

func TestEffectivePageBoxValues(t *testing.T) {
	n := displayNodes(t, `<html style="color:currentcolor"><body>
<div id="parent" style="color:red;background-color:currentcolor;flow-from:Main;flow-into:Source;writing-mode:vertical-rl;box-sizing:border-box;border-top-width:9px">
<div id="plain"></div>
<div id="child" style="color:blue;background-color:inherit;border-top-style:solid;border-top-width:inherit;flow-from:inherit;flow-into:inherit"></div>
<div id="reset" style="color:initial;writing-mode:unset;box-sizing:unset;flow-from:unset"></div>
<div id="self" style="color:currentcolor;border-top-style:solid"></div>
</div><div id="hidden" style="border-top-style:hidden;border-top-width:99px"></div>
<div id="solid" style="border-top-style:solid;border-top-width:2px;width:210mm"></div>
</body></html>`, nil)
	for _, tc := range []struct{ id, key, want string }{
		{"plain", "writing-mode", "vertical-rl"}, {"plain", "box-sizing", "content-box"},
		{"reset", "writing-mode", "vertical-rl"}, {"reset", "box-sizing", "content-box"},
	} {
		got, err := css.GetBoxKeyword(n[tc.id], tc.key)
		if err != nil || got != tc.want {
			t.Errorf("%s %s = %q %v", tc.id, tc.key, got, err)
		}
	}
	for _, tc := range []struct{ id, key, want string }{
		{"plain", "flow-from", ""}, {"plain", "flow-into", ""}, {"child", "flow-from", "Main"}, {"child", "flow-into", "Source"}, {"reset", "flow-from", ""},
	} {
		got, err := css.GetFlowName(n[tc.id], tc.key)
		if err != nil || got != tc.want {
			t.Errorf("%s %s = %q %v", tc.id, tc.key, got, err)
		}
	}
	red, blue, black := color.NRGBA{255, 0, 0, 255}, color.NRGBA{0, 0, 255, 255}, color.NRGBA{0, 0, 0, 255}
	for _, tc := range []struct {
		id, key string
		want    color.NRGBA
	}{
		{"parent", "background-color", red}, {"plain", "background-color", color.NRGBA{}},
		{"child", "background-color", blue}, {"child", "border-top-color", blue},
		{"reset", "color", black}, {"self", "color", red}, {"solid", "color", black},
	} {
		got, err := css.GetColor(n[tc.id], tc.key)
		if err != nil || got != tc.want {
			t.Errorf("%s %s = %v %v", tc.id, tc.key, got, err)
		}
	}
	for _, tc := range []struct {
		id    string
		width dimen.DU
	}{
		{"parent", 0}, {"plain", 0}, {"child", 0}, {"hidden", 0}, {"solid", 2 * dimen.PX}, {"self", dimen.PX},
	} {
		got, err := css.GetBorder(n[tc.id], "top")
		if err != nil || got.Width != tc.width {
			t.Errorf("border %s = %+v %v", tc.id, got, err)
		}
	}
	length, err := css.GetBoxLength(n["solid"], "width")
	want, _ := dimen.ParseAbsolute("210", "mm")
	if err != nil || length.Kind != style.AbsoluteLength || length.Absolute != want {
		t.Fatalf("page width = %+v %v", length, err)
	}
}

func TestCheckedAccessorErrorsAndDeferredValues(t *testing.T) {
	n := displayNodes(t, `<div id="deferred" style="width:calc(1px + 2px);color:var(--color)"></div><div id="rollback" style="width:revert"></div><div id="relative" style="border-top-style:solid;border-top-width:2em"></div>`, nil)
	for _, node := range []*styledtree.StyNode{nil, {}, n["deferred"], n["rollback"]} {
		if _, err := css.GetBoxLength(node, "width"); err == nil {
			t.Fatal("unchecked dimension")
		}
	}
	if _, err := css.GetColor(n["deferred"], "color"); err == nil {
		t.Fatal("unresolved color accepted")
	}
	if _, err := css.GetBorder(n["relative"], "top"); err == nil {
		t.Fatal("relative border silently sized")
	}
	if _, err := css.GetBorder(n["relative"], "invalid-side"); err == nil {
		t.Fatal("invalid side accepted")
	}
	if _, err := css.GetFlowName(n["relative"], "width"); err == nil {
		t.Fatal("wrong property accepted")
	}
	if _, err := css.GetColor(n["relative"], "width"); err == nil {
		t.Fatal("wrong color property accepted")
	}
}
