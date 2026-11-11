package style

import (
	"strings"
	"testing"
)

func TestShorthandExpansion(t *testing.T) {
	for _, shape := range []struct {
		input string
		want  []string
	}{
		{"1px", []string{"1px", "1px", "1px", "1px"}},
		{"1px 2px", []string{"1px", "2px", "1px", "2px"}},
		{"1px 2px 3px", []string{"1px", "2px", "3px", "2px"}},
		{"1px 2px 3px 4px", []string{"1px", "2px", "3px", "4px"}},
	} {
		for _, key := range []string{"margin", "margins", "padding", "border-width", "border-radius"} {
			t.Run(key+"/"+shape.input, func(t *testing.T) {
				got, err := SplitCompoundProperty(key, Property(shape.input))
				if err != nil {
					t.Fatal(err)
				}
				keys := []string{"margin-top", "margin-right", "margin-bottom", "margin-left"}
				if key == "padding" {
					for i := range keys {
						keys[i] = strings.Replace(keys[i], "margin", "padding", 1)
					}
				}
				if key == "border-width" {
					keys = []string{"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"}
				}
				if key == "border-radius" {
					keys = []string{"border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius"}
				}
				if len(got) != 4 {
					t.Fatalf("got %d values", len(got))
				}
				for i := range got {
					if got[i].Key != keys[i] || string(got[i].Value) != shape.want[i] {
						t.Errorf("entry %d = %v", i, got[i])
					}
				}
			})
		}
	}
}

func TestEllipticalBorderRadius(t *testing.T) {
	got, err := SplitCompoundProperty("border-radius", "1px 2px 3px/4px 5px")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []Property{"1px 4px", "2px 5px", "3px 4px", "2px 5px"} {
		if got[i].Value != want {
			t.Errorf("corner %d = %q", i, got[i].Value)
		}
	}
}

func TestInvalidShorthandArity(t *testing.T) {
	for _, key := range []string{"margin", "border-radius"} {
		for _, value := range []Property{"", "1 2 3 4 5"} {
			if got, err := SplitCompoundProperty(key, value); err == nil || got != nil {
				t.Errorf("accepted %s: %q", key, value)
			}
		}
	}
	for _, value := range []Property{"/1px", "1px/", "1px/2px/3px", "1px/1 2 3 4 5"} {
		if got, err := SplitCompoundProperty("border-radius", value); err == nil || got != nil {
			t.Errorf("accepted radius %q", value)
		}
	}
}
