package textstyle_test

import (
	"testing"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style/textstyle"
)

func TestFontSizeUnits(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  dimen.DU
	}{
		{"96px", 72 * dimen.BP}, {"72pt", 72 * dimen.BP}, {"1in", 72 * dimen.BP},
		{"25.4mm", 72 * dimen.BP}, {"2.54cm", 72 * dimen.BP}, {"6pc", 72 * dimen.BP}, {"101.6q", 72 * dimen.BP},
		{"1.5PX", 73728}, {".5pt", 32768}, {"1e1px", 10 * dimen.PX},
		{"150%", 30 * dimen.PX}, {"2em", 40 * dimen.PX}, {"2rem", 32 * dimen.PX},
		{"medium", 16 * dimen.PX}, {"larger", 24 * dimen.PX}, {"smaller", 819200},
		{`m\65 dium`, 16 * dimen.PX}, {" 2em /*comment*/ ", 40 * dimen.PX},
	} {
		t.Run(tc.value, func(t *testing.T) {
			s, err := textstyle.ResolveFontSize(tc.value, 20*dimen.PX, 16*dimen.PX)
			if err != nil || s != tc.want {
				t.Fatalf("size=%v want=%v error=%v", s, tc.want, err)
			}
		})
	}
	for _, v := range []string{"0pt", "-1px", "32768pt", "1e100px", "NaNpt", "1bp", "12", "2vw", "1ex", "1ch", "1 em", "1/**/em", "0.0000001px", "inherit"} {
		if _, err := textstyle.ResolveFontSize(v, 12*dimen.BP, 12*dimen.BP); err == nil {
			t.Errorf("accepted %s", v)
		}
	}
}

func TestRelativeWeights(t *testing.T) {
	for _, tc := range []struct{ parent, bolder, lighter int }{{50, 400, 50}, {100, 400, 100}, {349, 400, 100}, {350, 700, 100}, {549, 700, 100}, {550, 900, 400}, {749, 900, 400}, {750, 900, 700}, {900, 900, 700}, {950, 950, 700}} {
		n := source(t, `<p id="b" style="font-weight:bolder"></p><p id="l" style="font-weight:lighter"></p>`)
		p := textstyle.Initial()
		p.Weight = tc.parent
		ctx := textstyle.Context{Parent: p, RootFontSize: p.FontSize}
		b, e1 := textstyle.ResolveWithContext(n["b"], ctx)
		l, e2 := textstyle.ResolveWithContext(n["l"], ctx)
		if e1 != nil || e2 != nil || b.Weight != tc.bolder || l.Weight != tc.lighter {
			t.Fatalf("weight %d: %v %v %v %v", tc.parent, b, l, e1, e2)
		}
	}
}
