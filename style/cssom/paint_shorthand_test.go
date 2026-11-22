package cssom_test

import (
	"errors"
	"image/color"
	"strings"
	"testing"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestPaintShorthandCascade(t *testing.T) {
	for _, tc := range []struct {
		name, decls string
		want        map[string]string
	}{
		{"later longhands", `border:2px solid red;border-left-color:blue;background:red;background-color:blue`, map[string]string{"border-left-color": "blue", "border-right-color": "red", "background-color": "blue"}},
		{"later shorthands", `border-left-width:9px;border-left-style:solid;border-left-color:blue;border:0;background-color:blue;background:red`, map[string]string{"border-left-width": "0", "border-left-style": "none", "border-left-color": "currentcolor", "background-color": "red"}},
		{"duplicates retain order", `border:solid red;border-left-color:green;border:dashed blue;background:red;background-color:green;background:blue`, map[string]string{"border-left-color": "blue", "border-top-style": "dashed", "background-color": "blue"}},
		{"earlier important shorthand", `border:solid red!important;border-left-color:blue;border:0;background:red!important;background-color:blue;background:none`, map[string]string{"border-left-color": "red", "border-left-width": "medium", "background-color": "red"}},
		{"later important longhand", `border:solid red!important;border-left-color:blue!important;background:red!important;background-color:blue!important`, map[string]string{"border-left-color": "blue", "border-right-color": "red", "background-color": "blue"}},
		{"earlier important longhand", `border-left-color:blue!important;border:solid red;background-color:blue!important;background:red`, map[string]string{"border-left-color": "blue", "border-right-color": "red", "background-color": "blue"}},
		{"later important shorthand", `border-left-color:blue!important;border:solid red!important;background-color:blue!important;background:red!important`, map[string]string{"border-left-color": "red", "background-color": "red"}},
		{"side boundaries", `border:2px solid red;border-image-source:url(a);border-left:blue;border-radius:5px;border-top:0`, map[string]string{"border-right-width": "2px", "border-right-color": "red", "border-left-width": "medium", "border-left-style": "none", "border-left-color": "blue", "border-image-source": "url(a)", "border-top-left-radius": "5px"}},
		{"reset-only properties", `border-image-source:url(a);border-image-slice:20%;border-image-width:3;border-image-outset:2;border-image-repeat:round;border-radius:5px;border:solid;background-image:url(a);background-position:center;background-size:cover;background-repeat:no-repeat;background-attachment:fixed;background-origin:content-box;background-clip:content-box;background-blend-mode:multiply;background:red`, map[string]string{"border-image-source": "none", "border-image-slice": "100%", "border-image-width": "1", "border-image-outset": "0", "border-image-repeat": "stretch", "border-top-left-radius": "5px", "background-image": "none", "background-position": "0% 0%", "background-size": "auto auto", "background-repeat": "repeat", "background-attachment": "scroll", "background-origin": "padding-box", "background-clip": "border-box", "background-blend-mode": "multiply"}},
		{"important resets", `border:0!important;border-image-source:url(a);background:none!important;background-image:url(a)`, map[string]string{"border-image-source": "none", "background-image": "none", "background-color": "transparent"}},
		{"important longhands survive resets", `border-image-source:url(a)!important;border:0;background-image:url(a)!important;background:red`, map[string]string{"border-image-source": "url(a)", "background-image": "url(a)"}},
		{"invalid later declarations", `border:2px solid red;border:3px dashed bogus!important;border-left:1px 2px!important;background:red;background:blue no-repeat!important`, map[string]string{"border-top-width": "2px", "border-top-style": "solid", "border-left-width": "2px", "border-left-color": "red", "background-color": "red"}},
	} {
		for _, route := range []string{"inline", "inspected", "legacy"} {
			t.Run(tc.name+"/"+route, func(t *testing.T) {
				n := boxNode(t, route, tc.decls, "")
				for key, want := range tc.want {
					wantPaintProperty(t, n, key, want)
				}
			})
		}
	}
	for _, route := range []string{"inspected", "legacy"} {
		n := boxNode(t, route, `border:solid red;background:red`, `#x {border-left-color:green;background-color:green} .box {border:0;background:blue} #x {border:bogus!important;background:url(a)!important}`)
		wantPaintProperty(t, n, "border-left-color", "green")
		wantPaintProperty(t, n, "border-right-width", "0")
		wantPaintProperty(t, n, "background-color", "green")
	}
}

func TestPaintShorthandInspection(t *testing.T) {
	input := `border:0;border:solid bogus!important;background:red;background:linear-gradient(red,blue);background:none,none;background:var(--paint);border:calc(1px + 2px) solid`
	r := cssom.InspectStylesheet(".unmatched {" + input + "}")
	decls, diagnostics := cssom.InspectInlineStyle(input)
	for i, ds := range [][]cssom.Diagnostic{r.Diagnostics, diagnostics} {
		if len(ds) != 5 {
			t.Fatalf("route %d lost diagnostics: %+v", i, ds)
		}
		for j, index := range []int{2, 4, 5, 6, 7} {
			d := ds[j]
			if d.Code != "css.invalid-value" || d.Declaration != index || d.Rule != 1-i || d.Line < 1 || d.Column < 1 || !errors.Is(d.Err, style.ErrInvalidValue) {
				t.Fatalf("wrong provenance: %+v", d)
			}
		}
	}
	for _, source := range [][]cssom.SourceDeclaration{r.Rules[0].Declarations, decls} {
		if len(source) != 7 || source[1].Value != "solid bogus" || !source[1].Important || source[1].Valid || !source[0].Valid || source[3].Value != "linear-gradient(red,blue)" {
			t.Fatalf("source lost: %+v", source)
		}
	}
	if r.Sheet.Rules()[0].Value("border") != "0" || r.Sheet.Rules()[0].Value("background") != "red" {
		t.Fatal("invalid declarations displaced valid values")
	}
}

func TestPaintShorthandInheritanceAndInlinePriority(t *testing.T) {
	markup := `<body><div style="border:2px solid red;border-image-source:url(a);background:blue;background-image:url(b)">
<div id="inherit" style="border:inherit;background:inherit"></div>
<div id="initial" style="border:initial;background:initial"></div>
<div id="unset" style="border:unset;background:unset"></div>
<div id="default"></div>
<div id="current" style="color:green;border:solid;background:currentcolor"></div>
<div id="inline" style="border:solid green;background:green"></div>
<div id="important" style="border:solid green!important;background:green!important"></div>
<div id="revert" style="border:revert;background:revert-layer"></div>
</div></body>`
	h, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	om := cssom.NewCSSOM(nil)
	for _, text := range []string{`#inline, #important {border:solid blue;background:blue}`, `#inline, #important {border:solid red!important;background:red!important}`} {
		r := cssom.InspectStylesheet(text)
		if len(r.Diagnostics) > 0 {
			t.Fatal(r.Diagnostics)
		}
		if err := om.AddStylesForScope(nil, r.Sheet, cssom.Author); err != nil {
			t.Fatal(err)
		}
	}
	root, err := om.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	var visit func(*styledtree.StyNode)
	seen := 0
	visit = func(n *styledtree.StyNode) {
		for _, a := range n.HTMLNode().Attr {
			if a.Key != "id" {
				continue
			}
			seen++
			switch a.Val {
			case "inherit":
				b, err := css.GetBorder(n, "left")
				if err != nil || b.Width != 2*dimen.PX || b.Style != "solid" || b.Color != (color.NRGBA{R: 255, A: 255}) {
					t.Errorf("inherited border: %+v, %v", b, err)
				}
				wantPaintProperty(t, n, "background-color", "blue")
				wantPaintProperty(t, n, "background-image", "url(b)")
				wantPaintProperty(t, n, "border-image-source", "url(a)")
			case "initial", "unset", "default":
				b, err := css.GetBorder(n, "left")
				if err != nil || b.Width != 0 || b.Style != "none" {
					t.Errorf("%s border: %+v, %v", a.Val, b, err)
				}
				wantPaintProperty(t, n, "background-color", "transparent")
				wantPaintProperty(t, n, "background-image", "none")
				wantPaintProperty(t, n, "border-image-source", "none")
			case "current":
				for _, key := range []string{"background-color", "border-left-color"} {
					c, err := css.GetColor(n, key)
					if err != nil || c != (color.NRGBA{G: 128, A: 255}) {
						t.Errorf("%s = %v %v", key, c, err)
					}
				}
			case "inline", "important":
				want := "red"
				if a.Val == "important" {
					want = "green"
				}
				wantPaintProperty(t, n, "background-color", want)
				wantPaintProperty(t, n, "border-left-color", want)
			case "revert":
				wantPaintProperty(t, n, "border-image-source", "revert")
				wantPaintProperty(t, n, "background-image", "revert-layer")
				if _, err := css.GetBorder(n, "left"); err == nil {
					t.Error("fabricated rollback border")
				}
				if _, err := css.GetColor(n, "background-color"); err == nil {
					t.Error("fabricated rollback color")
				}
			}
		}
		for _, ch := range n.Children(true) {
			visit(ch.Payload)
		}
	}
	visit(root.Payload)
	if seen != 8 {
		t.Fatalf("visited %d test elements", seen)
	}
}

func wantPaintProperty(t *testing.T, n *styledtree.StyNode, key, want string) {
	t.Helper()
	got, err := css.GetProperty(n, key)
	if err != nil || string(got) != want {
		t.Errorf("%s = %q %v, want %s", key, got, err, want)
	}
}
