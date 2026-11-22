package cssom

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func TestInspectStylesheetRetainsSource(t *testing.T) {
	input := "\n.frame { color:red!important; color:blue; display:bogus; width:10px }\n.frame { color:green }"
	r := InspectStylesheet(input)
	if len(r.Rules) != 2 || len(r.Rules[0].Declarations) != 4 {
		t.Fatalf("lost declarations: %+v", r)
	}
	if r.Rules[0].Line != 2 || r.Rules[1].Index != 2 || r.Rules[0].Declarations[1].Index != 2 {
		t.Fatal("lost source order/locations")
	}
	if d := r.Rules[0].Declarations[0]; d.Property != "color" || d.Value != "red" || !d.Important || !d.Valid {
		t.Fatalf("declaration = %+v", d)
	}
	if len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "css.invalid-value" || r.Diagnostics[0].Rule != 1 || r.Diagnostics[0].Declaration != 3 {
		t.Fatalf("diagnostics = %+v", r.Diagnostics)
	}
	if got := r.Sheet.Rules()[0].Value("color"); got != "red" {
		t.Fatalf("importance lost: %s", got)
	}
	if got := r.Sheet.Rules()[0].Properties(); !reflect.DeepEqual(got, []string{"color", "width"}) {
		t.Fatalf("winners = %v", got)
	}
	if !reflect.DeepEqual(r, InspectStylesheet(input)) {
		t.Fatal("inspection is nondeterministic")
	}
}

func TestInspectionFailures(t *testing.T) {
	for _, tc := range []struct{ input, code string }{
		{`.frame[ { width:10px }`, "css.selector-syntax"},
		{`@media print { .frame {width:10px} }`, "css.unsupported-rule"},
		{`@import "never-fetch.css";`, "css.unsupported-rule"},
		{`.frame;`, "css.rule-syntax"},
		{`.frame`, "css.rule-syntax"},
		{`.frame { width:10px`, "css.rule-syntax"},
		{`.frame { width }`, "css.declaration-syntax"},
		{`.frame { width: }`, "css.declaration-syntax"},
		{`.frame { width: calc(10px }`, "css.declaration-syntax"},
		{`.frame { color:"unterminated }`, "css.token-syntax"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			r := InspectStylesheet(tc.input)
			found := false
			for _, d := range r.Diagnostics {
				if d.Code == tc.code {
					found = true
					if d.Line == 0 || d.Column == 0 {
						t.Fatal("missing location")
					}
				}
			}
			if !found {
				t.Fatalf("missing %s: %+v", tc.code, r.Diagnostics)
			}
		})
	}
	r := InspectStylesheet(`.x[ {color:red} .valid {color:green}`)
	if len(r.Rules) != 2 || len(r.Sheet.Rules()) != 1 || r.Diagnostics[0].Rule != 1 || errors.Unwrap(r.Diagnostics[0]) == nil {
		t.Fatalf("bad recovery: %+v", r)
	}
}

func TestInlineInspectionAndPermissiveStyling(t *testing.T) {
	input := `color:red!important; broken; color:blue; width:calc(10px + 2px); content:"a;b:c"; display:bogus`
	declarations, diagnostics := InspectInlineStyle(input)
	if len(declarations) != 6 || len(diagnostics) != 2 {
		t.Fatalf("inspection = %+v, %+v", declarations, diagnostics)
	}
	if diagnostics[0].Declaration != 2 || diagnostics[0].Rule != 0 || diagnostics[1].Declaration != 6 {
		t.Fatal("bad diagnostic association")
	}
	if declarations[4].Value != `"a;b:c"` {
		t.Fatal("split quoted content")
	}
	r := newLocalPseudoRule(input)
	if r.Value("color") != "red" || r.Value("width") != "calc(10px + 2px)" || r.Value("display") != "" {
		t.Fatalf("permissive reduction changed: %v", r)
	}
	for _, bad := range []string{`width:calc(1px]`, `color:`, `color red`, `color:"unterminated`} {
		if _, ds := InspectInlineStyle(bad); len(ds) == 0 {
			t.Errorf("no diagnostic for %q", bad)
		}
	}
}

func TestInspectedSheetCascadeAndInlinePriority(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<body><div class="frame" id="x" style="color:green!important"></div><div class="frame" id="y"></div></body>`))
	if err != nil {
		t.Fatal(err)
	}
	sheet := InspectStylesheet(`#x {color:red!important} .frame {color:blue!important;color:yellow}`)
	if len(sheet.Diagnostics) != 0 {
		t.Fatal(sheet.Diagnostics)
	}
	om := NewCSSOM(nil)
	if err := om.AddStylesForScope(nil, sheet.Sheet, Author); err != nil {
		t.Fatal(err)
	}
	root, err := om.Style(doc)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	var visit func(*styledtree.StyNode)
	visit = func(n *styledtree.StyNode) {
		for _, a := range n.HTMLNode().Attr {
			if a.Key == "id" {
				p, e := css.GetProperty(n, "color")
				if e != nil {
					t.Fatal(e)
				}
				got[a.Val] = string(p)
			}
		}
		for _, child := range n.Children(true) {
			visit(child.Payload)
		}
	}
	visit(root.Payload)
	if got["x"] != "green" || got["y"] != "blue" {
		t.Fatal(got)
	}
}
