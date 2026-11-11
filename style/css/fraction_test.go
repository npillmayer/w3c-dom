package css_test

import (
	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style/css"
	"testing"
)

func TestFractionalDimensionParsing(t *testing.T) {
	for _, s := range []string{"1.5px", "-.5mm", "+.25BP", "2147483647sp", "-2147483648sp"} {
		want, _, err := dimen.ParsePercent(s)
		if err != nil {
			t.Fatal(err)
		}
		got, err := css.ParseDimen(s)
		var value dimen.DU
		if err != nil || got.Match().Just(&value) == nil || value != want {
			t.Errorf("%s = %v, %v", s, got, err)
		}
	}
	for _, unit := range []string{"em", "ex", "ch", "rem", "vw", "vh", "vmin", "vmax"} {
		d, err := css.ParseDimen("-.25" + unit)
		v, ok := d.RelativeValue()
		if err != nil || !ok || v != -.25 || d.UnitString() != unit {
			t.Errorf("relative %s: %v %v", unit, d, err)
		}
	}
	d, err := css.ParseDimen("80.5%")
	var percent css.Percent
	if err != nil || d.Match().Percentage(&percent) == nil || percent != 81 {
		t.Fatalf("percentage: %v %v", percent, err)
	}
	for _, s := range []string{"32768px", "-32768.01px", "2147483648sp", "1..2px", ".px", "NaNpx"} {
		if d, err := css.ParseDimen(s); err == nil || !d.IsNone() {
			t.Errorf("accepted %s", s)
		}
	}
}
