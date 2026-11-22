package style

import (
	"errors"
	"image/color"
	"math"
	"testing"

	"github.com/npillmayer/w3c-dom/dimen"
)

func TestCheckedBoxLengths(t *testing.T) {
	for _, input := range []string{"1in", "25.4mm", "2.54cm", "72pt", "96px", "6pc", "101.6Q", "9.6e1px"} {
		v, err := ParseBoxLength("width", input)
		if err != nil || v.Kind != AbsoluteLength || v.Absolute != dimen.IN {
			t.Errorf("%s = %+v, %v", input, v, err)
		}
	}
	for _, tc := range []struct {
		key, input string
		kind       LengthKind
		number     float64
		keyword    string
	}{
		{"width", "150.25%", PercentageLength, 150.25, ""},
		{"left", "-12.5%", PercentageLength, -12.5, ""},
		{"width", ".25rem", RelativeLength, .25, ""},
		{"border-left-width", "2em", RelativeLength, 2, ""},
		{"width", ` a\75 to `, KeywordLength, 0, "auto"},
		{"max-width", "none", KeywordLength, 0, "none"},
		{"min-width", "min-content", KeywordLength, 0, "min-content"},
	} {
		v, err := ParseBoxLength(tc.key, tc.input)
		if err != nil || v.Kind != tc.kind || v.Magnitude != tc.number || v.Keyword != tc.keyword {
			t.Errorf("%s:%s = %+v, %v", tc.key, tc.input, v, err)
		}
	}
	for _, input := range []string{"0", "-0", "+0.0", "0e100", "/*a*/ .5px /*b*/", "32767pt"} {
		if _, err := ParseBoxLength("padding-top", input); err != nil {
			t.Errorf("%s: %v", input, err)
		}
	}
	if v, err := ParseBoxLength("left", "-32768pt"); err != nil || v.Absolute != dimen.DU(math.MinInt32) {
		t.Fatalf("minimum DU: %+v %v", v, err)
	}
	for _, tc := range []struct{ key, value string }{
		{"width", "1"}, {"width", "1e-999"}, {"width", "1e309px"}, {"width", "32768pt"},
		{"width", "NaNpx"}, {"width", "Infpx"}, {"width", "10sp"}, {"width", "10bp"}, {"width", "10px junk"},
		{"width", "-.000000001px"}, {"width", "-1e-999px"}, {"width", "-1%"},
		{"padding-top", "auto"}, {"padding-top", "-1px"}, {"border-top-width", "10%"}, {"width", "thin"},
		{"width", "none"}, {"width", "inherit"}, {"width", "var(--x)"}, {"color", "10px"},
	} {
		v, err := ParseBoxLength(tc.key, tc.value)
		if !errors.Is(err, ErrInvalidValue) || v != (BoxLength{}) {
			t.Errorf("accepted %s:%s: %+v %v", tc.key, tc.value, v, err)
		}
	}
}

func TestCheckedFlowNamesAndColors(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Main-Flow", "Main-Flow"}, {`\4d ain`, "Main"}, {`\31 Main`, "1Main"}, {"章", "章"}, {"--flow", "--flow"},
		{"none", ""}, {`\6e one`, ""}, {"NONE", ""},
	} {
		got, err := ParseFlowName(tc.input)
		if err != nil || got != tc.want {
			t.Errorf("%q = %q %v", tc.input, got, err)
		}
	}
	for _, s := range []string{"", "a b", "a/**/b", `"main"`, "1bad", "default", `\64 efault`, "inherit", "revert-layer"} {
		if _, err := ParseFlowName(s); err == nil {
			t.Errorf("accepted name %q", s)
		}
	}
	c, err := ParseColorValue(`c\75 rrentColor`)
	if err != nil || !c.Current {
		t.Fatalf("currentcolor: %+v %v", c, err)
	}
	for _, tc := range []struct {
		input string
		want  color.NRGBA
	}{
		{"transparent", color.NRGBA{}}, {"#1234", color.NRGBA{17, 34, 51, 68}}, {"rgb(255, 0, 0)", color.NRGBA{255, 0, 0, 255}},
		{`r\65 d`, color.NRGBA{255, 0, 0, 255}},
	} {
		v, err := ParseColorValue(tc.input)
		if err != nil || v.Current || v.Color != tc.want {
			t.Errorf("%s: %+v %v", tc.input, v, err)
		}
	}
	for _, s := range []string{"default", "blacK", "rgb(NaN,0,0)", "rgb(1e999,0,0)"} {
		if _, err := ParseColorValue(s); err == nil {
			t.Errorf("accepted color %q", s)
		}
	}
}

func TestBoxMetadataAndDeclarationValidation(t *testing.T) {
	for key, want := range map[string]Property{"box-sizing": "content-box", "writing-mode": "horizontal-tb", "color": "black", "background-color": "transparent", "border-top-color": "currentcolor", "min-width": "auto"} {
		if got := InitialProperty(key); got != want {
			t.Errorf("%s = %s", key, got)
		}
	}
	if !IsCascading("writing-mode") || IsCascading("box-sizing") || IsCascading("flow-from") || IsCascading("background-color") {
		t.Fatal("incorrect inheritance metadata")
	}
	for _, tc := range []struct{ key, value string }{
		{"width", "bogus"}, {"padding-left", "-1px"}, {"border-top-color", "default"}, {"box-sizing", "padding-box"},
		{"writing-mode", "vertical-up"}, {"flow-from", "two names"}, {"padding", "1px -2px"}, {"position", "calc(1px)"},
	} {
		if _, err := NormalizeDeclaration(tc.key, Property(tc.value)); err == nil {
			t.Errorf("accepted %s:%s", tc.key, tc.value)
		}
	}
	for _, tc := range []struct{ key, value, want string }{
		{"width", `\69 nherit`, "inherit"}, {"box-sizing", "BORDER-BOX", "border-box"}, {"writing-mode", `vertical-\72 l`, "vertical-rl"},
		{"color", "revert-layer", "revert-layer"}, {"custom-extension", "opaque:value", "opaque:value"},
		{"width", "calc(10px + 2px)", "calc(10px + 2px)"}, {"color", "var(--color)", "var(--color)"},
		{"border-color", "rgb(1, 2, 3) red", "rgb(1, 2, 3) red"},
	} {
		got, err := NormalizeDeclaration(tc.key, Property(tc.value))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s:%s = %q %v", tc.key, tc.value, got, err)
		}
	}
}
