package dimen

import "testing"

func TestFractionalDimensions(t *testing.T) {
	for _, tc := range []struct {
		s    string
		want DU
	}{{"1.5px", 98304}, {"-.5PX", -32768}, {"+.5sp", 1}, {"-.5sp", -1}, {"2147483647sp", 2147483647}, {"-2147483648sp", -2147483648}, {"1.25bp", 81920}} {
		got, _, err := ParsePercent(tc.s)
		if err != nil || got != tc.want {
			t.Errorf("%s = %d, %v", tc.s, got, err)
		}
	}
	for _, s := range []string{"2147483648sp", "2147483647.00000001sp", "-2147483648.00000001sp", "-2147483649sp", "32768px", "999999999999999999999999mm", "1..2px", ".px", "NaNpx", "1.px"} {
		if _, _, err := ParsePercent(s); err == nil {
			t.Errorf("accepted %s", s)
		}
	}
}
