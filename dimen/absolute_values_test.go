package dimen

import "testing"

func TestAbsoluteCSSUnitsAndExponentRounding(t *testing.T) {
	for _, tc := range []struct {
		number, unit string
		want         DU
	}{
		{"6", "pc", IN}, {"101.6", "q", IN}, {"9.6e1", "px", IN},
		{"7.62939453125e-6", "pt", 1}, {"-7.62939453125e-6", "pt", -1},
		{"1e-10000", "pt", 0}, {"0e10000", "px", 0},
	} {
		got, err := ParseAbsolute(tc.number, tc.unit)
		if err != nil || got != tc.want {
			t.Errorf("%s%s = %d, %v want %d", tc.number, tc.unit, got, err, tc.want)
		}
	}
	for _, number := range []string{"1e10000", "NaN", "Inf", "32768"} {
		if _, err := ParseAbsolute(number, "pt"); err == nil {
			t.Errorf("accepted out-of-range %spt", number)
		}
	}
}
