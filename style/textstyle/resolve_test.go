package textstyle_test

import (
	"errors"
	"image/color"
	"strings"
	"sync"
	"testing"

	"github.com/npillmayer/tree"
	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/style/textstyle"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func source(t *testing.T, input string) map[string]*styledtree.StyNode {
	t.Helper()
	h, err := html.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	css := cssom.NewCSSOM(nil)
	sheets, err := douceuradapter.ExtractStyleElementsWithError(h)
	if err != nil {
		t.Fatal(err)
	}
	for _, sheet := range sheets {
		if err := css.AddStylesForScope(nil, sheet, cssom.Author); err != nil {
			t.Fatal(err)
		}
	}
	root, err := css.Style(h)
	if err != nil {
		t.Fatal(err)
	}
	nodes := make(map[string]*styledtree.StyNode)
	var visit func(*tree.Node[*styledtree.StyNode])
	visit = func(n *tree.Node[*styledtree.StyNode]) {
		hn := n.Payload.HTMLNode()
		if hn.Type != html.TextNode || nodes[hn.Data] == nil {
			nodes[hn.Data] = n.Payload
		}
		for _, a := range n.Payload.HTMLNode().Attr {
			if a.Key == "id" {
				nodes[a.Val] = n.Payload
			}
		}
		for _, child := range n.Children(true) {
			visit(child)
		}
	}
	visit(root)
	return nodes
}

func resolved(t *testing.T, n *styledtree.StyNode) textstyle.Style {
	t.Helper()
	s, err := textstyle.Resolve(n)
	if err != nil {
		t.Fatal(err)
	}
	return s[0]
}

func TestSourceInheritanceAndCascade(t *testing.T) {
	n := source(t, `<html lang="de-DE" style="font-size:20px"><head><style>
 p {font-size:2em; color:navy} .child {color:red!important}
 </style></head><body><p id="parent" style="font-weight:500">é &amp;
 <span id="child" class="child" style="font-size:150%;font-weight:bolder;color:green">אב</span>
 <i id="reset" style="font-size:1rem;font-style:initial;color:initial">z</i>
 <span id="inherit" style="font-size:inherit;color:currentcolor;font-weight:lighter">inherit</span>
 </p></body></html>`)
	p := resolved(t, n["parent"])
	c := resolved(t, n["child"])
	i := resolved(t, n["reset"])
	h := resolved(t, n["inherit"])
	if p.FontSize != 40*dimen.PX || c.FontSize != 60*dimen.PX || i.FontSize != 20*dimen.PX || h.FontSize != p.FontSize {
		t.Fatalf("source/root sizes: %+v %+v %+v %+v", p, c, i, h)
	}
	if c.Weight != 700 || h.Weight != 100 || c.Language != "de-DE" || c.Color != "red" || h.Color != "navy" || i.Color != "black" || i.Slant != "normal" {
		t.Fatalf("cascade/hints: %+v %+v %+v", c, h, i)
	}
	if c.Foreground != (color.NRGBA{R: 255, A: 255}) || h.Foreground != p.Foreground {
		t.Fatal("foreground is not a resolved color")
	}
	if got := resolved(t, n["אב"]); got != c {
		t.Fatalf("text did not inherit complete context: %+v", got)
	}
}

func TestHTMLDefaultsAndCSSOverrides(t *testing.T) {
	n := source(t, `<div lang="he" dir="rtl"><strong id="bold">b</strong><em id="italic">i</em>
 <pre id="pre"> p </pre><bdi id="bdi">abc</bdi>
 <strong id="reset" style="font-weight:initial">r</strong>
 <em id="unset" style="font-style:unset">u</em>
 <span id="override" dir="auto" style="direction:ltr;unicode-bidi:initial">אב</span>
 <span id="empty-lang" lang="">unknown</span></div>`)
	if resolved(t, n["bold"]).Weight != 700 || resolved(t, n["italic"]).Slant != "italic" || resolved(t, n["pre"]).WhiteSpace != "pre" {
		t.Fatal("HTML defaults missing")
	}
	if s := resolved(t, n["bdi"]); s.Direction != textstyle.LeftToRight || s.UnicodeBidi != "isolate" || s.Language != "he" {
		t.Fatalf("bdi: %+v", s)
	}
	if resolved(t, n["reset"]).Weight != 400 || resolved(t, n["unset"]).Slant != "normal" || resolved(t, n["empty-lang"]).Language != "und" {
		t.Fatal("resets missing")
	}
	if s := resolved(t, n["override"]); s.Direction != textstyle.LeftToRight || s.UnicodeBidi != "normal" {
		t.Fatalf("author CSS did not override HTML hints: %+v", s)
	}
}

func TestAutoDirectionUsesOriginalSource(t *testing.T) {
	for _, tc := range []struct {
		fragment string
		want     textstyle.Direction
	}{
		{`<span dir="rtl">אב</span>abc`, textstyle.LeftToRight},
		{`<span dir="LTR">abc</span>אב`, textstyle.RightToLeft},
		{`<span dir="auto">אב</span>abc`, textstyle.LeftToRight},
		{`<bdi>אב</bdi>abc`, textstyle.LeftToRight},
		{`<em><span dir="rtl">אב</span></em>abc`, textstyle.LeftToRight},
		{`123<em>אב</em>abc`, textstyle.RightToLeft},
		{`<script>abc</script><style>abc</style><template>abc</template>אב`, textstyle.RightToLeft},
		{`<span dir="ltr">abc</span>123`, textstyle.RightToLeft},
	} {
		t.Run(tc.fragment, func(t *testing.T) {
			// Direction comes from text outside the selected span as well.
			n := source(t, `<div dir="rtl"><p id="auto" dir="auto">`+tc.fragment+`<span id="selection">123</span></p></div>`)
			s := resolved(t, n["selection"])
			if s.Direction != tc.want || s.UnicodeBidi != "normal" {
				t.Fatalf("selection: %+v", s)
			}
			if resolved(t, n["auto"]).UnicodeBidi != "isolate" {
				t.Fatal("auto element lacks isolate")
			}
		})
	}
}

func TestLineHeightAndHyphenation(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  textstyle.LineHeight
	}{
		{"normal", textstyle.LineHeight{}},
		{"1.5", textstyle.LineHeight{Kind: textstyle.FactorLineHeight, Factor: 1.5}},
		{"0", textstyle.LineHeight{Kind: textstyle.FactorLineHeight}},
		{"150%", textstyle.LineHeight{Kind: textstyle.LengthLineHeight, Length: 30 * dimen.PX}},
		{"1.5em", textstyle.LineHeight{Kind: textstyle.LengthLineHeight, Length: 30 * dimen.PX}},
		{"2rem", textstyle.LineHeight{Kind: textstyle.LengthLineHeight, Length: 32 * dimen.PX}},
		{"24pt", textstyle.LineHeight{Kind: textstyle.LengthLineHeight, Length: 24 * dimen.BP}},
	} {
		t.Run(tc.value, func(t *testing.T) {
			n := source(t, `<div style="font-size:20px;line-height:`+tc.value+`;hyphens:auto"><span id="child" style="font-size:2em">soft­hyphen</span><span id="reset" style="line-height:initial;hyphens:initial"></span></div>`)
			s := resolved(t, n["child"])
			if s.LineHeight != tc.want || s.Hyphens != "auto" || s.FontSize != 40*dimen.PX {
				t.Fatalf("policy inheritance: %+v", s)
			}
			if n["soft­hyphen"].HTMLNode().Data != "soft­hyphen" {
				t.Fatal("source text changed")
			}
			if s := resolved(t, n["reset"]); s.LineHeight != (textstyle.LineHeight{}) || s.Hyphens != "manual" {
				t.Fatalf("policy reset: %+v", s)
			}
		})
	}
	n := source(t, `<html style="font-size:2rem;line-height:2rem"><body><p>x</p></body></html>`)
	if s := resolved(t, n["p"]); s.FontSize != 32*dimen.PX || s.LineHeight.Length != 64*dimen.PX {
		t.Fatalf("root rem computation: %+v", s)
	}
}

func TestCheckedMetadataAndKeywords(t *testing.T) {
	n := source(t, `<div style="font-family:'A Font', serif;font-size:24px;font-weight:700;color:navy;hyphens:none">
 <span id="escaped" style="font-size:i\6e herit;font-weight:UNSET;color:c\75 rrentcolor;white-space:PRE-WRAP;unicode-bidi:ISOLATE;hyphens:inherit">x</span>
 <span id="initial" style="font-family:INITIAL;font-size:initial;font-weight:initial;white-space:initial;unicode-bidi:unset">y</span></div>`)
	s := resolved(t, n["escaped"])
	if s.FontSize != 24*dimen.PX || s.Weight != 700 || s.Color != "navy" || s.WhiteSpace != "pre-wrap" || s.UnicodeBidi != "isolate" || s.Hyphens != "none" || s.Family != "'A Font', serif" {
		t.Fatalf("checked metadata: %+v", s)
	}
	r := resolved(t, n["initial"])
	if r.FontSize != 16*dimen.PX || r.Weight != 400 || r.Family != "serif" || r.UnicodeBidi != "normal" || r.WhiteSpace != "normal" {
		t.Fatalf("initial: %+v", r)
	}
}

func TestStructuredFailureNoPartialResults(t *testing.T) {
	for _, decl := range []string{
		"font-size:1 em", "font-size:0px", "font-size:2vw", "font-size:calc(1em + 1px)",
		"font-weight:0", "font-weight:1001", "font-weight:450.5", "font-style:oblique 10deg",
		"white-space:unknown", "unicode-bidi:unknown", "hyphens:unknown", "line-height:-1",
		"line-height:-1e-999", "line-height:1e999", "line-height:calc(2em)", "font-family:serif,",
		"font-size:revert", "line-height:revert-layer",
	} {
		t.Run(decl, func(t *testing.T) {
			n := source(t, `<p id="good">good</p><p id="bad" style="`+decl+`">bad</p>`)
			if resolved(t, n["good"]).FontSize != 16*dimen.PX {
				t.Fatal("unselected invalid sibling interfered")
			}
			s, err := textstyle.Resolve(n["good"], n["bad"])
			var e *textstyle.Error
			if s != nil || !errors.As(err, &e) || !errors.Is(err, style.ErrInvalidValue) || e.Node != n["bad"].HTMLNode() || e.Property != strings.SplitN(decl, ":", 2)[0] {
				t.Fatalf("partial or unlocated failure: %v %v", s, err)
			}
		})
	}
	for _, attr := range []string{`lang="not_a_language_!"`, `dir="diagonal"`} {
		n := source(t, `<p id="bad" `+attr+`>x</p>`)
		var e *textstyle.Error
		if _, err := textstyle.Resolve(n["bad"]); !errors.As(err, &e) || e.Node != n["bad"].HTMLNode() {
			t.Fatalf("attribute error: %v", err)
		}
	}
	if s, err := textstyle.Resolve(nil); err == nil || s != nil {
		t.Fatal("nil accepted")
	}
}

func TestDetachedSnapshotsAndExplicitContext(t *testing.T) {
	n := source(t, `<div style="font-size:20px"><p id="p" style="font-size:150%">x</p></div>`)
	want := resolved(t, n["p"])
	parent := resolved(t, n["div"])
	got, err := textstyle.ResolveWithContext(n["p"], textstyle.Context{Parent: parent, RootFontSize: 16 * dimen.PX})
	if err != nil || got != want {
		t.Fatalf("explicit context: %+v %v", got, err)
	}
	if _, err := textstyle.ResolveWithContext(n["p"], textstyle.Context{}); err == nil {
		t.Fatal("zero font context accepted")
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				s, err := textstyle.Resolve(n["p"], n["p"])
				if err != nil || s[0] != want || s[1] != want {
					t.Errorf("concurrent snapshot: %v %v", s, err)
					return
				}
				s[0].Color = "changed"
			}
		})
	}
	wg.Wait()
	n["p"].Styles().Add("font-size", "12px")
	if want.FontSize != 30*dimen.PX || resolved(t, n["p"]).FontSize != 12*dimen.PX {
		t.Fatal("snapshot aliases source/cache")
	}
}
