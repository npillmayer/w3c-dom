package textstyle_test

import (
	"errors"
	"testing"

	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/textstyle"
)

func TestAuthoredTextDeclarationChecks(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"font-size", ".0000001em"}, {"font-size", "1e20%"}, {"font-size", "2rem"}, {"font-size", `m\65 dium`},
		{"font-size", "inherit"}, {"font-weight", "bolder"}, {"font-family", "'A Font', serif"}, {"font-style", "ITALIC"},
		{"white-space", "pre-wrap"}, {"unicode-bidi", "isolate"}, {"hyphens", "auto"}, {"color", "currentcolor"},
		{"line-height", "1e20em"}, {"line-height", "150%"}, {"line-height", "1.2"}, {"word-spacing", "normal"},
	} {
		if err := textstyle.ValidateDeclaration(tc.key, tc.value); err != nil {
			t.Errorf("valid declaration %v: %v", tc, err)
		}
	}
	for _, tc := range []struct{ key, value string }{
		{"font-size", "0"}, {"font-size", "-1em"}, {"font-size", "2vw"}, {"font-size", "var(--size)"}, {"font-size", "revert"},
		{"font-family", "serif,"}, {"font-weight", "450.5"}, {"font-style", "oblique 10deg"}, {"hyphens", "other"},
		{"white-space", "unknown"}, {"unicode-bidi", "unknown"}, {"line-height", "-1em"}, {"unknown", "initial"},
	} {
		if err := textstyle.ValidateDeclaration(tc.key, tc.value); !errors.Is(err, style.ErrInvalidValue) {
			t.Errorf("unchecked declaration %v: %v", tc, err)
		}
	}
	if value, err := textstyle.ParseKeyword("white-space", `/**/N\4f RMAL/**/`); err != nil || value != "normal" {
		t.Fatal("lost CSS keyword syntax", value, err)
	}
	if _, err := textstyle.ParseKeyword("line-height", "1.2"); err == nil {
		t.Fatal("non-keyword parsed as keyword")
	}
}
