package css_test

import (
	"errors"
	"fmt"
	"image/color"
	"sync"
	"testing"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/textstyle"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

// Supply snapshots computed before border resolution. The border getter must
// request the original element/ancestor, without recomputing any text styles.
func borderFonts(t *testing.T, nodes map[string]*styledtree.StyNode) css.FontContext {
	t.Helper()
	var ordered []*styledtree.StyNode
	for _, n := range nodes {
		ordered = append(ordered, n)
	}
	styles, err := textstyle.Resolve(ordered...)
	if err != nil {
		t.Fatal(err)
	}
	sizes := make(map[*styledtree.StyNode]dimen.DU)
	fonts := css.FontContext{}
	for i, n := range ordered {
		sizes[n] = styles[i].FontSize
		if n.HTMLNode().Data == "html" {
			fonts.RootFontSize = styles[i].FontSize
		}
	}
	fonts.FontSize = func(n *styledtree.StyNode) (dimen.DU, error) {
		size, ok := sizes[n]
		if !ok {
			return 0, fmt.Errorf("missing source font for %s", n)
		}
		return size, nil
	}
	return fonts
}

func TestBorderWithSourceFontSizes(t *testing.T) {
	n := displayNodes(t, `<html id="root" style="font-size:2rem;border-top:.5rem solid"><body>
<div id="parent" style="font-size:20px"><div id="child" style="font-size:150%;color:#224466;
border-top:.1em solid currentcolor;border-right:.5rem dashed red;
border-bottom:0.00001em dotted;border-left:1.5pt double"></div></div></body></html>`, nil)
	fonts := borderFonts(t, n)
	for _, tc := range []struct {
		id, side, lineStyle string
		width               dimen.DU
		color               color.NRGBA
	}{
		{"root", "top", "solid", 16 * dimen.PX, color.NRGBA{A: 255}},
		{"child", "top", "solid", 3 * dimen.PX, color.NRGBA{R: 0x22, G: 0x44, B: 0x66, A: 255}},
		{"child", "right", "dashed", 16 * dimen.PX, color.NRGBA{R: 255, A: 255}},
		{"child", "bottom", "dotted", 15, color.NRGBA{R: 0x22, G: 0x44, B: 0x66, A: 255}},
		{"child", "left", "double", 98304, color.NRGBA{R: 0x22, G: 0x44, B: 0x66, A: 255}},
	} {
		t.Run(tc.id+"/"+tc.side, func(t *testing.T) {
			b, err := css.GetBorderWithFontContext(n[tc.id], tc.side, fonts)
			want := css.Border{Width: tc.width, Style: tc.lineStyle, Color: tc.color}
			if err != nil || b != want {
				t.Fatalf("border = %+v, %v; want %+v", b, err, want)
			}
		})
	}
	// The legacy API still cannot resolve a nonzero em/rem width.
	for _, side := range []string{"top", "right", "bottom"} {
		if b, err := css.GetBorder(n["child"], side); err == nil || b != (css.Border{}) {
			t.Fatalf("legacy API sized %s: %+v %v", side, b, err)
		}
	}
}

func TestBorderInheritsComputedWidthAtAncestor(t *testing.T) {
	n := displayNodes(t, `<html id="root" style="font-size:16px"><body>
<div id="parent" style="font-size:20px;color:red;border-top:.25em solid currentcolor;border-left:.5rem dashed green">
 <div id="child" style="font-size:40px;color:blue;border-top:inherit;border-left:inherit">
  <div id="grandchild" style="font-size:80px;color:currentcolor;border-top:inherit;border-left:inherit"></div>
 </div>
 <div id="initial" style="font-size:40px;border-top-style:solid;border-top-width:initial"></div>
 <div id="unset" style="font-size:40px;border-top-style:solid;border-top-width:unset"></div>
 <div id="plain" style="border-top-style:solid"></div>
</div></body></html>`, nil)
	fonts := borderFonts(t, n)
	lookup := fonts.FontSize
	var requested []*styledtree.StyNode
	fonts.FontSize = func(node *styledtree.StyNode) (dimen.DU, error) {
		requested = append(requested, node)
		return lookup(node)
	}
	for _, id := range []string{"child", "grandchild"} {
		requested = nil
		b, err := css.GetBorderWithFontContext(n[id], "top", fonts)
		if err != nil || b.Width != 5*dimen.PX || b.Style != "solid" || b.Color != (color.NRGBA{B: 255, A: 255}) {
			t.Fatalf("%s did not inherit computed width/currentcolor: %+v %v", id, b, err)
		}
		if len(requested) != 1 || requested[0] != n["parent"] {
			t.Fatalf("wrong font context for inherited width: %v", requested)
		}
		requested = nil
		b, err = css.GetBorderWithFontContext(n[id], "left", fonts)
		if err != nil || b.Width != 8*dimen.PX || b.Style != "dashed" || b.Color != (color.NRGBA{G: 128, A: 255}) || len(requested) != 0 {
			t.Fatalf("inherited rem/color: %+v %v, font requests %v", b, err, requested)
		}
	}
	for _, id := range []string{"initial", "unset", "plain"} {
		b, err := css.GetBorderWithFontContext(n[id], "top", css.FontContext{})
		if err != nil || b.Width != dimen.PX {
			t.Fatalf("%s: %+v %v", id, b, err)
		}
	}
}

func TestBorderFontContextDemandAndCompatibility(t *testing.T) {
	n := displayNodes(t, `<html id="root" style="border-top-style:solid;border-top-width:inherit"><body>
<div id="none" style="border-top:2em none;color:red">
 <div id="from-none" style="border-top-style:solid;border-top-width:inherit"></div></div>
<div id="hidden" style="border-top:2rem hidden">
 <div id="from-hidden" style="border-top-style:solid;border-top-width:inherit"></div></div>
<div id="child-none" style="border-top:2em solid"><div id="suppressed" style="border-top-width:inherit"></div></div>
<div id="zero" style="border-top:0em solid"></div><div id="zero-metric" style="border-top:0ex solid"></div>
<div id="absolute" style="border-top:2px solid red"></div><div id="keyword" style="border-top:thick solid"></div>
<div id="rem" style="border-top:.5rem solid"></div><div id="em" style="border-top:.5em solid"></div>
</body></html>`, nil)
	called := false
	unused := css.FontContext{RootFontSize: -1, FontSize: func(*styledtree.StyNode) (dimen.DU, error) {
		called = true
		return 0, errors.New("unexpected font lookup")
	}}
	for _, id := range []string{"root", "none", "from-none", "hidden", "from-hidden", "suppressed", "zero", "zero-metric", "absolute", "keyword"} {
		old, err := css.GetBorder(n[id], "top")
		b, newErr := css.GetBorderWithFontContext(n[id], "top", unused)
		if err != nil || newErr != nil || b != old || called {
			t.Fatalf("%s changed absolute/zero behavior: %+v/%+v %v %v", id, old, b, err, newErr)
		}
	}
	if b, err := css.GetBorderWithFontContext(n["rem"], "top", css.FontContext{RootFontSize: 16 * dimen.PX}); err != nil || b.Width != 8*dimen.PX {
		t.Fatalf("rem required element lookup: %+v %v", b, err)
	}
	if b, err := css.GetBorderWithFontContext(n["em"], "top", css.FontContext{FontSize: func(*styledtree.StyNode) (dimen.DU, error) { return 20 * dimen.PX, nil }}); err != nil || b.Width != 10*dimen.PX {
		t.Fatalf("em required root lookup: %+v %v", b, err)
	}
}

func TestBorderFontErrorsAreChecked(t *testing.T) {
	n := displayNodes(t, `<div id="parent" style="border-top:1em solid"><div id="child" style="border-top:inherit"></div></div>`, nil)
	missing := errors.New("source snapshot unavailable")
	fonts := css.FontContext{FontSize: func(node *styledtree.StyNode) (dimen.DU, error) {
		if node != n["parent"] {
			t.Errorf("requested wrong source: %s", node)
		}
		return 0, missing
	}}
	if b, err := css.GetBorderWithFontContext(n["child"], "top", fonts); b != (css.Border{}) || !errors.Is(err, missing) {
		t.Fatalf("lost source lookup failure: %+v %v", b, err)
	}
	for _, size := range []dimen.DU{0, -1, dimen.Fil, dimen.Fill, dimen.Infinity} {
		for _, unit := range []string{"em", "rem"} {
			n["parent"].Styles().Add("border-top-width", style.Property("1"+unit))
			fonts := css.FontContext{RootFontSize: size, FontSize: func(*styledtree.StyNode) (dimen.DU, error) { return size, nil }}
			if b, err := css.GetBorderWithFontContext(n["child"], "top", fonts); b != (css.Border{}) || !errors.Is(err, style.ErrInvalidValue) {
				t.Errorf("accepted font size %v %s: %+v %v", size, unit, b, err)
			}
		}
	}
	for _, unit := range []string{"em", "rem"} {
		n["parent"].Styles().Add("border-top-width", style.Property("1"+unit))
		if b, err := css.GetBorderWithFontContext(n["parent"], "top", css.FontContext{}); b != (css.Border{}) || !errors.Is(err, style.ErrInvalidValue) {
			t.Errorf("accepted missing context %s: %+v %v", unit, b, err)
		}
	}
	for _, value := range []string{"1ex", "1ch", "1vw", "1vh", "1vmin", "1vmax", "10%", "-1em", "-1e-999em", "NaNem", "1e999em", "1e100em", "calc(1em + 1px)", "var(--width)", "revert", "revert-layer"} {
		// Test checked access directly; permissive styling may drop invalid input.
		n["parent"].Styles().Add("border-top-width", style.Property(value))
		fonts := css.FontContext{RootFontSize: 16 * dimen.PX, FontSize: func(*styledtree.StyNode) (dimen.DU, error) { return 16 * dimen.PX, nil }}
		if b, err := css.GetBorderWithFontContext(n["parent"], "top", fonts); b != (css.Border{}) || !errors.Is(err, style.ErrInvalidValue) {
			t.Errorf("unchecked width %s: %+v %v", value, b, err)
		}
	}
	for _, node := range []*styledtree.StyNode{nil, {}, styledtree.NewNodeForHTMLNode(&html.Node{Type: html.TextNode}).Payload} {
		if b, err := css.GetBorderWithFontContext(node, "top", css.FontContext{}); err == nil || b != (css.Border{}) {
			t.Fatal("accepted non-element", node, b, err)
		}
	}
	if b, err := css.GetBorderWithFontContext(n["parent"], "invalid", css.FontContext{}); err == nil || b != (css.Border{}) {
		t.Fatal("accepted invalid side", b, err)
	}
}

func TestBorderRelativeDimensionRangeAndRounding(t *testing.T) {
	n := displayNodes(t, `<div id="target" style="border-top:1em solid"></div>`, nil)["target"]
	for _, tc := range []struct {
		value      string
		size, want dimen.DU
		invalid    bool
	}{
		{".25em", 2, 1, false}, // Half a DU rounds upward.
		{".1em", 1, 0, false},
		{"1e-999em", dimen.PX, 0, false},
		{"1em", dimen.Fil - 1, dimen.Fil - 1, false},
		{"2em", dimen.Fil / 2, 0, true},
		{"2rem", dimen.Fil / 2, 0, true},
		{"1e308em", dimen.PX, 0, true},
	} {
		n.Styles().Add("border-top-width", style.Property(tc.value))
		fonts := css.FontContext{RootFontSize: tc.size, FontSize: func(*styledtree.StyNode) (dimen.DU, error) { return tc.size, nil }}
		b, err := css.GetBorderWithFontContext(n, "top", fonts)
		if tc.invalid {
			if b != (css.Border{}) || !errors.Is(err, style.ErrInvalidValue) {
				t.Errorf("accepted overflow: %+v %+v %v", tc, b, err)
			}
		} else if err != nil || b.Width != tc.want {
			t.Errorf("rounding: %+v got %+v %v", tc, b, err)
		}
	}
}

func TestBorderWithFontsPreservesCascadeAndSource(t *testing.T) {
	n := displayNodes(t, `<html id="root" style="font-size:16px"><body><div id="target" style="font-size:20px;border-left:.1em solid;color:blue"></div></body></html>`, nil,
		displaySheet(t, `#target { border-left-width:.25em!important; border-left-color:currentcolor }`))
	fonts := borderFonts(t, n)
	want := css.Border{Width: 5 * dimen.PX, Style: "solid", Color: color.NRGBA{B: 255, A: 255}}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			b, err := css.GetBorderWithFontContext(n["target"], "left", fonts)
			if err != nil || b != want {
				t.Errorf("concurrent border: %+v %v", b, err)
			}
		})
	}
	wg.Wait()
	if p, err := css.GetProperty(n["target"], "border-left-width"); err != nil || p != ".25em" {
		t.Fatal("source was mutated", p, err)
	}
}
