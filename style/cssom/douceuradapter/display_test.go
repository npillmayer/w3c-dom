package douceuradapter

import (
	"github.com/aymerick/douceur/parser"
	"reflect"
	"testing"
)

func TestDisplayDuplicatesFilterBeforeWinning(t *testing.T) {
	for _, tc := range []struct {
		body, value string
		important   bool
		keys        []string
	}{
		{"display:bogus;color:red", "", false, []string{"color"}},
		{"display:inline;display:bogus!important;color:red", "inline", false, []string{"display", "color"}},
		{"display:bogus!important;color:red;display:grid", "grid", false, []string{"color", "display"}},
		{"display:inline!important;color:red;display:grid", "inline", true, []string{"display", "color"}},
		{"display:inline;color:red;display:revert-layer", "revert-layer", false, []string{"color", "display"}},
	} {
		s, err := parser.Parse("p{" + tc.body + "}")
		if err != nil {
			t.Fatal(err)
		}
		r := Wrap(s).Rules()[0]
		if got := r.Properties(); !reflect.DeepEqual(got, tc.keys) {
			t.Fatalf("%s keys=%v want %v", tc.body, got, tc.keys)
		}
		if v := r.Value("display"); string(v) != tc.value || r.IsImportant("display") != tc.important {
			t.Fatalf("%s value=%s important=%v", tc.body, v, r.IsImportant("display"))
		}
	}
}
