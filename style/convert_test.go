package style

import (
	"image/color"
	"testing"
)

func TestCSSColors(t *testing.T) {
	for _, tc := range []struct {
		s    string
		want color.NRGBA
	}{
		{"transparent", color.NRGBA{}}, {"green", color.NRGBA{0, 128, 0, 255}}, {"lime", color.NRGBA{0, 255, 0, 255}},
		{" ReBeccaPurple ", color.NRGBA{102, 51, 153, 255}}, {"aliceblue", color.NRGBA{240, 248, 255, 255}},
		{"#f08", color.NRGBA{255, 0, 136, 255}}, {"#f008", color.NRGBA{255, 0, 0, 136}},
		{"#12345678", color.NRGBA{18, 52, 86, 120}}, {"#123456", color.NRGBA{18, 52, 86, 255}},
		{"rgba(255,0,0,.5)", color.NRGBA{255, 0, 0, 128}}, {"rgb(100% 0% 0% / 50%)", color.NRGBA{255, 0, 0, 128}},
		{"rgb(300,-1,0)", color.NRGBA{255, 0, 0, 255}},
		{"hsl(120,100%,50%)", color.NRGBA{0, 255, 0, 255}},
		{"hsla(.5turn 100% 50% / .5)", color.NRGBA{0, 255, 255, 128}},
		{"hsl(-240deg 100% 50%)", color.NRGBA{0, 255, 0, 255}},
		{"hwb(0 0% 0% / 50%)", color.NRGBA{255, 0, 0, 128}},
		{"hwb(0 100% 100%)", color.NRGBA{128, 128, 128, 255}},
	} {
		got, err := ParseColor(tc.s)
		if err != nil || got != tc.want {
			t.Errorf("%s = %v, %v want %v", tc.s, got, err, tc.want)
		}
		if v := color.NRGBAModel.Convert(Property(tc.s).Color()).(color.NRGBA); v != tc.want {
			t.Errorf("Property.Color %s = %v", tc.s, v)
		}
	}
	for _, s := range []string{"", "#12", "#gggggg", "rgb(1,2)", "rgb(1,2,3/4)", "rgb(1%,2,3)", "rgba(1 2 3 / NaN)", "hsl(0 1 2)", "currentcolor", "lab(50% 1 2)", "rgb(1 2 3)junk"} {
		if _, err := ParseColor(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}
func TestColorString(t *testing.T) {
	for _, c := range []color.Color{color.NRGBA{18, 52, 86, 255}, color.NRGBA{200, 100, 50, 128}, color.RGBA{128, 0, 0, 128}, color.White, color.Black} {
		got, err := ParseColor(ColorString(c))
		want := color.NRGBAModel.Convert(c).(color.NRGBA)
		if err != nil || got != want {
			t.Errorf("round trip %v = %v, %v", c, got, err)
		}
	}
	if ColorString(color.Transparent) != "transparent" || ColorString(nil) != "default" || Property("default").Color() != nil {
		t.Fatal("sentinel/transparency mismatch")
	}
	if Property("invalid").Color() != color.Black {
		t.Fatal("legacy fallback changed")
	}
}
