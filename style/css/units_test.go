package css_test

import (
	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style/css"
	"testing"
)

func TestCSSAbsoluteUnits(t *testing.T) {
	if 96*dimen.PX != dimen.IN || 72*dimen.PT != dimen.IN || dimen.PT != dimen.BP || dimen.BP != 65536 {
		t.Fatal("inconsistent CSS unit constants")
	}
	for _, tc := range []struct {
		input string
		want  dimen.DU
	}{
		{"96px", dimen.IN}, {"72pt", dimen.IN}, {"1in", dimen.IN},
		{"25.4mm", dimen.IN}, {"2.54cm", dimen.IN},
		{"4px", 3 * dimen.PT}, {"3pt", 3 * dimen.PT}, {"1.5PX", 73728},
		{"-.5pT", -32768}, {"1bp", 65536}, {"1sp", 1}, {"1", 1},
		{"32768px", 1610612736}, {"-32768pt", -2147483648},
	} {
		direct, percent, err := dimen.ParsePercent(tc.input)
		if err != nil || percent || direct != tc.want {
			t.Errorf("ParsePercent(%s)=%d,%v,%v", tc.input, direct, percent, err)
		}
		parsed, err := css.ParseDimen(tc.input)
		var got dimen.DU
		if err != nil || parsed.Match().Just(&got) == nil || got != tc.want {
			t.Errorf("ParseDimen(%s)=%d,%v", tc.input, got, err)
		}
	}
	for _, input := range []string{"43691px", "-43691px", "32768pt", "-32768.00001pt"} {
		if _, _, err := dimen.ParsePercent(input); err == nil {
			t.Errorf("ParsePercent accepted overflow %s", input)
		}
		if _, err := css.ParseDimen(input); err == nil {
			t.Errorf("ParseDimen accepted overflow %s", input)
		}
	}
}
