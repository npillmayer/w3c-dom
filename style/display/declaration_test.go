package display_test

import (
	"errors"
	"testing"

	"github.com/npillmayer/w3c-dom/style/display"
)

func TestParseDeclaration(t *testing.T) {
	for _, keyword := range []display.WideKeyword{display.Initial, display.Inherit, display.Unset, display.Revert, display.RevertLayer} {
		for _, input := range []string{keyword.String(), "/**/" + keyword.String() + "/**/"} {
			v, k, err := display.ParseDeclaration(input)
			if err != nil || v != (display.Value{}) || k != keyword {
				t.Fatalf("%s: %v %v %v", input, v, k, err)
			}
			if _, err := display.Parse(input); !errors.Is(err, display.ErrInvalidValue) {
				t.Fatal("Parse must still reject CSS-wide values")
			}
		}
	}
	for _, input := range []string{`\69 nherit`, "INHERIT", `/**/\000069nherit/**/`} {
		_, k, err := display.ParseDeclaration(input)
		if err != nil || k != display.Inherit {
			t.Fatalf("%s: %v %v", input, k, err)
		}
	}
	for _, input := range []string{"grid", "inline flow-root", "none", "contents", "table-cell", "ruby", "inline list-item"} {
		v, k, err := display.ParseDeclaration(input)
		want, _ := display.Parse(input)
		if err != nil || v != want || k != display.NoKeyword {
			t.Fatalf("%s: %v %v %v", input, v, k, err)
		}
	}
	for _, input := range []string{"", "bogus", "block-inline", "inherit block", "initial initial", "revert grid", "none !important", "var(--display)", "inherit/*", "\xff"} {
		v, k, err := display.ParseDeclaration(input)
		if v != (display.Value{}) || k != display.NoKeyword || !errors.Is(err, display.ErrInvalidValue) {
			t.Fatalf("%q: %v %v %v", input, v, k, err)
		}
	}
	if display.NoKeyword.String() != "" || display.WideKeyword(255).String() != "" {
		t.Fatal("invalid keyword serialized")
	}
}

func FuzzParseDeclaration(f *testing.F) {
	for _, input := range []string{"grid", "initial", "inherit", "unset", "revert", "revert-layer", `\69 nherit`, "inherit block", "", "/*", "inline flow-root", "\xff"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		v, k, err := display.ParseDeclaration(input)
		if err != nil {
			if v != (display.Value{}) || k != display.NoKeyword || !errors.Is(err, display.ErrInvalidValue) {
				t.Fatalf("bad failure: %v %v %v", v, k, err)
			}
			return
		}
		if v.Valid() == (k != display.NoKeyword) {
			t.Fatal("expected exactly one value variant")
		}
		canonical := v.CSS()
		if k != display.NoKeyword {
			canonical = k.String()
		}
		round, wide, err := display.ParseDeclaration(canonical)
		if err != nil || round != v || wide != k {
			t.Fatalf("round trip %q: %v %v %v", canonical, round, wide, err)
		}
		ordinary, err := display.Parse(input)
		if k == display.NoKeyword {
			if err != nil || ordinary != v {
				t.Fatal("value parsers disagree")
			}
		} else if !errors.Is(err, display.ErrInvalidValue) {
			t.Fatal("Parse accepted CSS-wide keyword")
		}
	})
}
