package css_test

import (
	"github.com/npillmayer/w3c-dom/style/css"
	"testing"
)

func TestDimensionVariants(t *testing.T) {
	values := []css.DimenT{{}, css.Auto(), css.Inherit(), css.Initial(), css.JustDimen(7), css.Percentage(80), css.DimenOption("fit-content")}
	patterns := css.DimenPatterns[int]{Unset: 0, Auto: 1, Inherit: 2, Initial: 3, Just: 4, Default: 5}
	for i, d := range values {
		want := i
		if i > 5 {
			want = 5
		}
		if got := css.DimenPattern[int](d).OneOf(patterns); got != want {
			t.Errorf("variant %d: pattern %d", i, got)
		}
		if (d.Match().Just(nil) != nil) != (i == 4) {
			t.Errorf("variant %d matched absolute incorrectly", i)
		}
		if (d.Match().Percentage(nil) != nil) != (i == 5) {
			t.Errorf("variant %d matched percent incorrectly", i)
		}
		for j, other := range values {
			if (d.Match().IsKind(other) != nil) != (i == j) {
				t.Errorf("kind %d matched %d incorrectly", i, j)
			}
		}
	}
	for _, unit := range []string{"em", "ex", "ch", "rem", "vw", "vh", "vmin", "vmax"} {
		d, err := css.ParseDimen("12" + unit)
		if err != nil {
			t.Fatal(err)
		}
		if !d.IsRelative() || d.IsPercent() || d.Match().Percentage(nil) != nil || d.Match().IsKind(css.DimenT{}) != nil || d.Match().IsKind(css.Percentage(1)) != nil {
			t.Errorf("incorrect relative classification: %s", unit)
		}
		if css.DimenPattern[int](d).OneOf(patterns) != 5 {
			t.Errorf("relative pattern: %s", unit)
		}
	}
}

func TestPercentageParsing(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  css.Percent
	}{{"0%", 0}, {"80%", 80}, {"100%", 100}, {"-1%", 0}, {"101%", 100}, {"256%", 100}} {
		d, err := css.ParseDimen(tc.input)
		if err != nil {
			t.Fatal(err)
		}
		var got css.Percent
		if d.Match().Percentage(&got) == nil || got != tc.want {
			t.Errorf("%s = %v, want %v", tc.input, got, tc.want)
		}
		if d != css.Percentage(tc.want) {
			t.Errorf("parse/constructor mismatch: %s", tc.input)
		}
		if p, err := css.PercentFromString(tc.input); err != nil || p != tc.want {
			t.Errorf("PercentFromString(%s) = %v, %v", tc.input, p, err)
		}
	}
	for _, bad := range []string{"x%", "999999999999999999999999999%"} {
		if _, err := css.ParseDimen(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
		if _, err := css.PercentFromString(bad); err == nil {
			t.Errorf("accepted percent %q", bad)
		}
	}
}

func TestPositionVariants(t *testing.T) {
	values := []css.PositionT{{}, css.Static(), css.Relative(nil), css.Absolute(nil), css.Fixed(nil)}
	patterns := css.PositionPatterns[int]{Unset: 0, Static: 1, Relative: 2, Absolute: 3, Fixed: 4, Default: -1}
	for i, p := range values {
		if got := css.PositionPattern[int](p).OneOf(patterns); got != i {
			t.Errorf("variant %d pattern = %d", i, got)
		}
		if p.IsUnset() != (i == 0) || p.IsRelative() != (i == 2) || p.IsAbsolute() != (i == 3) || p.IsFixed() != (i == 4) {
			t.Errorf("variant %d predicate mismatch", i)
		}
		for j, other := range values {
			if (p.Match().IsKind(other) != nil) != (i == j) {
				t.Errorf("kind %d matched %d incorrectly", i, j)
			}
		}
	}
}
