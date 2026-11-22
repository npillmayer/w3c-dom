package style

import (
	"errors"
	"reflect"
	"testing"
)

func TestBorderShorthandComponentsAndResets(t *testing.T) {
	for _, tc := range []struct{ input, width, line, color string }{
		{"0", "0", "none", "currentcolor"},
		{"none", "medium", "none", "currentcolor"},
		{"solid", "medium", "solid", "currentcolor"},
		{"red", "medium", "none", "red"},
		{"2px dashed blue", "2px", "dashed", "blue"},
		{"blue 2px dashed", "2px", "dashed", "blue"},
		{"dashed blue 2px", "2px", "dashed", "blue"},
		{"dashed 2px blue", "2px", "dashed", "blue"},
		{"2px blue dashed", "2px", "dashed", "blue"},
		{"blue dashed 2px", "2px", "dashed", "blue"},
		{"thin/*gap*/solid rgb(255, 0, 0)", "thin", "solid", "rgb(255, 0, 0)"},
		{`s\6f lid .5em c\75 rrentColor`, ".5em", `s\6f lid`, `c\75 rrentColor`},
	} {
		for _, key := range []string{"border", "border-top", "border-right", "border-bottom", "border-left"} {
			t.Run(key+"/"+tc.input, func(t *testing.T) {
				want := map[string]Property{}
				for _, side := range []string{"top", "right", "bottom", "left"} {
					if key != "border" && key != "border-"+side {
						continue
					}
					want["border-"+side+"-width"] = Property(tc.width)
					want["border-"+side+"-style"] = Property(tc.line)
					want["border-"+side+"-color"] = Property(tc.color)
				}
				if key == "border" {
					for k, v := range map[string]Property{"border-image-source": "none", "border-image-slice": "100%", "border-image-width": "1", "border-image-outset": "0", "border-image-repeat": "stretch"} {
						want[k] = v
					}
				}
				if got := expandedPaint(t, key, tc.input); !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			})
		}
	}
}

func TestSolidBackgroundShorthand(t *testing.T) {
	for _, tc := range []struct{ input, color string }{
		{"#ffffff", "#ffffff"}, {"none", "transparent"}, {"transparent", "transparent"},
		{"none red", "red"}, {"red none", "red"}, {`\6e one /*gap*/ currentColor`, "currentColor"},
		{"rgba(1, 2, 3, .5)", "rgba(1, 2, 3, .5)"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			want := map[string]Property{
				"background-color": Property(tc.color), "background-image": "none", "background-position": "0% 0%",
				"background-size": "auto auto", "background-repeat": "repeat", "background-attachment": "scroll",
				"background-origin": "padding-box", "background-clip": "border-box",
			}
			if got := expandedPaint(t, "background", tc.input); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestPaintShorthandCSSWideValues(t *testing.T) {
	for _, key := range []string{"border", "border-top", "border-right", "border-bottom", "border-left", "background"} {
		for _, tc := range []struct{ input, want string }{
			{"initial", "initial"}, {"INHERIT", "inherit"}, {`\75 nset`, "unset"},
			{"revert", "revert"}, {"revert-layer", "revert-layer"},
		} {
			baseline := expandedPaint(t, key, "none")
			got := expandedPaint(t, key, tc.input)
			if len(got) != len(baseline) {
				t.Fatalf("%s lost subproperties", key)
			}
			for k := range baseline {
				if got[k] != Property(tc.want) {
					t.Errorf("%s:%s: %s=%s", key, tc.input, k, got[k])
				}
				if InitialProperty(k) == NullStyle || IsCascading(k) {
					t.Errorf("incorrect metadata for %s", k)
				}
			}
			if v, err := NormalizeDeclaration(key, Property(tc.input)); err != nil || v != Property(tc.want) {
				t.Errorf("normalization = %s, %v", v, err)
			}
		}
	}
}

func TestPaintShorthandRejectsWholeDeclaration(t *testing.T) {
	for _, tc := range []struct {
		key     string
		invalid []string
	}{
		{"border", []string{"", "1px 2px", "solid dashed", "red blue", "-1px solid", "10% solid", "1 solid", "solid bogus", "solid initial", "inherit red", "red, blue", "calc(1px + 2px) solid", "var(--border)", "solid rgb(1,2,3"}},
		{"border-left", []string{"1px 2px", "initial solid", "url(a)", "var(--edge)"}},
		{"background", []string{"", "red blue", "none none", "red initial", "inherit red", "red, blue", "none, none", "url(a)", "linear-gradient(red, blue)", "red no-repeat", "red center", "red / cover", "var(--paint)", "rgb(1,2,3"}},
	} {
		for _, input := range tc.invalid {
			t.Run(tc.key+"/"+input, func(t *testing.T) {
				got, err := SplitCompoundProperty(tc.key, Property(input))
				if got != nil || !errors.Is(err, ErrInvalidValue) {
					t.Fatalf("partial/accepted expansion %v, %v", got, err)
				}
				if v, err := NormalizeDeclaration(tc.key, Property(input)); v != NullStyle || !errors.Is(err, ErrInvalidValue) {
					t.Fatalf("accepted declaration %s, %v", v, err)
				}
			})
		}
	}
}

func expandedPaint(t *testing.T, key, input string) map[string]Property {
	t.Helper()
	values, err := SplitCompoundProperty(key, Property(input))
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]Property, len(values))
	for _, kv := range values {
		if _, exists := got[kv.Key]; exists {
			t.Fatalf("duplicate subproperty %s", kv.Key)
		}
		got[kv.Key] = kv.Value
	}
	return got
}
