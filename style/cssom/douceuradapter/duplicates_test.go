package douceuradapter

import (
	"github.com/aymerick/douceur/parser"
	"reflect"
	"testing"
)

func TestDuplicateDeclarations(t *testing.T) {
	for _, tc := range []struct {
		body, want string
		important  bool
		keys       []string
	}{
		{"color:red; width:1px; color:blue", "blue", false, []string{"width", "color"}},
		{"color:red!important; width:1px; color:blue", "red", true, []string{"color", "width"}},
		{"color:red; width:1px; color:blue!important", "blue", true, []string{"width", "color"}},
		{"color:red!important; width:1px; color:blue!important", "blue", true, []string{"width", "color"}},
		{"COLOR:red; width:1px; color:blue", "blue", false, []string{"width", "color"}},
	} {
		t.Run(tc.body, func(t *testing.T) {
			sheet, err := parser.Parse("p{" + tc.body + "}")
			if err != nil {
				t.Fatal(err)
			}
			r := Wrap(sheet).Rules()[0]
			if got := r.Properties(); !reflect.DeepEqual(got, tc.keys) {
				t.Errorf("keys = %v, want %v", got, tc.keys)
			}
			if got := string(r.Value("color")); got != tc.want || r.IsImportant("color") != tc.important {
				t.Errorf("value=%q important=%v", got, r.IsImportant("color"))
			}
			if r.Value("missing") != "" || r.IsImportant("missing") {
				t.Error("missing declaration found")
			}
		})
	}
}
