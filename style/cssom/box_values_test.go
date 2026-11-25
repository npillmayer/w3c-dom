package cssom_test

import (
	"strings"
	"testing"

	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestBoxInvalidValuesDoNotWin(t *testing.T) {
	for _, tc := range []struct{ key, good, bad string }{
		{"break-before", "recto", "sideways"},
		{"width", "12px", "-1px"}, {"height", "13px", "garbage"}, {"padding-left", "2px", "-1px"},
		{"color", "red", "default"}, {"background-color", "transparent", "bogus"},
		{"border-top-width", "2px", "10%"}, {"border-top-style", "solid", "bogus"},
		{"flow-from", "Main", "two names"}, {"flow-into", "Source", "default"},
		{"box-sizing", "border-box", "padding-box"}, {"writing-mode", "vertical-rl", "nonsense"},
	} {
		for _, route := range []string{"inline", "inspected", "legacy"} {
			t.Run(tc.key+"/"+route, func(t *testing.T) {
				decls := tc.key + ":" + tc.good + ";" + tc.key + ":" + tc.bad + "!important"
				n := boxNode(t, route, decls, "")
				got, err := css.GetProperty(n, tc.key)
				if err != nil || string(got) != tc.good {
					t.Fatalf("%s = %q %v, want %s", tc.key, got, err, tc.good)
				}
			})
		}
	}
	for _, route := range []string{"inspected", "legacy"} {
		n := boxNode(t, route, "width:12px", `#x {width:-1px!important}`)
		got, _ := css.GetProperty(n, "width")
		if got != "12px" {
			t.Fatalf("higher specificity invalid value won: %s", got)
		}
	}
}

func TestEdgeShorthandValidationIsAtomic(t *testing.T) {
	for _, route := range []string{"inline", "inspected", "legacy"} {
		n := boxNode(t, route, `padding:1px 2px;padding:3px -4px!important;border-color:rgb(255, 0, 0) blue`, "")
		for key, want := range map[string]string{"padding-top": "1px", "padding-right": "2px", "padding-bottom": "1px", "padding-left": "2px", "border-top-color": "rgb(255, 0, 0)", "border-right-color": "blue"} {
			got, err := css.GetProperty(n, key)
			if err != nil || string(got) != want {
				t.Errorf("%s %s = %s %v, want %s", route, key, got, err, want)
			}
		}
	}
}

func TestBoxInspectionRetainsInvalidSource(t *testing.T) {
	r := cssom.InspectStylesheet(`.unmatched {width:12px;width:-1px!important;color:default;flow-from:two names}`)
	if len(r.Diagnostics) != 3 || len(r.Rules[0].Declarations) != 4 {
		t.Fatalf("lost diagnostics/source: %+v", r)
	}
	for i, d := range r.Diagnostics {
		if d.Code != "css.invalid-value" || d.Rule != 1 || d.Declaration != i+2 || d.Property == "" || d.Err == nil {
			t.Fatalf("bad diagnostic: %+v", d)
		}
	}
	if r.Rules[0].Declarations[1].Value != "-1px" || !r.Rules[0].Declarations[1].Important || r.Rules[0].Declarations[1].Valid {
		t.Fatal("invalid original declaration lost")
	}
	if got := r.Sheet.Rules()[0].Value("width"); got != "12px" {
		t.Fatalf("permissive sheet lost valid width: %s", got)
	}
}

func boxNode(t *testing.T, route, decls, extraRules string) *styledtree.StyNode {
	t.Helper()
	markup := `<body><div id="x" class="box"`
	if route == "inline" {
		markup += ` style="` + decls + `"`
	}
	markup += `></div></body>`
	h, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	om := cssom.NewCSSOM(nil)
	if route != "inline" {
		text := `.box {` + decls + `}` + extraRules
		var sheet cssom.StyleSheet
		if route == "inspected" {
			sheet = cssom.InspectStylesheet(text).Sheet
		} else {
			parsed, err := parser.Parse(text)
			if err != nil {
				t.Fatal(err)
			}
			sheet = douceuradapter.Wrap(parsed)
		}
		if err := om.AddStylesForScope(nil, sheet, cssom.Author); err != nil {
			t.Fatal(err)
		}
	}
	root, err := om.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	var result *styledtree.StyNode
	var visit func(*styledtree.StyNode)
	visit = func(n *styledtree.StyNode) {
		for _, a := range n.HTMLNode().Attr {
			if a.Key == "id" && a.Val == "x" {
				result = n
			}
		}
		for _, ch := range n.Children(true) {
			visit(ch.Payload)
		}
	}
	visit(root.Payload)
	if result == nil {
		t.Fatal("missing box")
	}
	return result
}

func TestBreakBeforeInspectionAndCascade(t *testing.T) {
	r := cssom.InspectStylesheet(`.unmatched {break-before:verso;break-before:page;break-before:bogus!important}`)
	if len(r.Diagnostics) != 1 || len(r.Rules[0].Declarations) != 3 {
		t.Fatalf("lost source: %+v", r)
	}
	ds := r.Rules[0].Declarations
	if ds[0].Value != "verso" || ds[1].Value != "page" || ds[2].Valid || !ds[2].Important {
		t.Fatalf("lost declarations: %+v", ds)
	}
	for _, route := range []string{"inspected", "legacy"} {
		n := boxNode(t, route, `break-before:recto!important`, `#x {break-before:verso}`)
		got, err := css.GetBoxKeyword(n, "break-before")
		if err != nil || got != "recto" {
			t.Fatalf("importance: %s %v", got, err)
		}
		n = boxNode(t, route, `break-before:recto`, `#x {break-before:verso}`)
		got, err = css.GetBoxKeyword(n, "break-before")
		if err != nil || got != "verso" {
			t.Fatalf("specificity: %s %v", got, err)
		}
	}
}
