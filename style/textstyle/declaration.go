package textstyle

import "github.com/npillmayer/w3c-dom/style"

var textKeywords = map[string]string{
	"font-style":   "normal italic oblique",
	"white-space":  "normal nowrap pre pre-wrap pre-line break-spaces",
	"direction":    "ltr rtl",
	"unicode-bidi": "normal embed isolate bidi-override isolate-override plaintext",
	"hyphens":      "none manual auto",
	"line-height":  "normal", "word-spacing": "normal", "letter-spacing": "normal",
	"word-break": "normal break-all keep-all", "overflow-wrap": "normal break-word anywhere",
}

// ParseKeyword checks the keyword form of a text property, decoding CSS escapes
// and ASCII case. It does not resolve CSS-wide keywords. For line-height and
// spacing it accepts normal only; their length/number forms are not keywords.
// Applications can use this parser when imposing a narrower preparation profile.
func ParseKeyword(key, value string) (string, error) {
	return keyword(key, value, textKeywords[key])
}

// ValidateDeclaration checks an authored text-style longhand without requiring
// source font sizes. Use it before cascade reduction to audit losing and
// nonmatching declarations. Effective values still require Resolve: relative
// font sizes can underflow or overflow for a particular source context.
// Supported properties are the longhands retained by Style, plus the keyword
// forms accepted by ParseKeyword. CSS-wide inherit/initial/unset are accepted;
// rollback and unknown properties fail. No font services or DOM are consulted.
func ValidateDeclaration(key, value string) error {
	switch key {
	case "font-family", "font-size", "font-weight", "font-style", "color", "white-space", "direction", "unicode-bidi", "line-height", "hyphens", "word-spacing", "letter-spacing", "word-break", "overflow-wrap":
	default:
		return invalid(key, value)
	}
	if wide := style.DeclarationKeyword(value); wide != "" {
		if wide == "inherit" || wide == "initial" || wide == "unset" {
			return nil
		}
		return invalid(key, value)
	}
	var err error
	switch key {
	case "font-family":
		_, err = family(value)
	case "font-weight":
		_, err = weight(value, Initial().Weight)
	case "font-size":
		// Relative values must be checked before computing with a made-up font.
		// A tiny positive em value may be valid with a large source font size.
		v, e := style.ParseBoxLength("width", value)
		if e == nil && (v.Kind == style.PercentageLength || v.Kind == style.RelativeLength && (v.Unit == "em" || v.Unit == "rem")) {
			if v.Magnitude <= 0 {
				return invalid(key, value)
			}
			return nil
		}
		_, err = ResolveFontSize(value, Initial().FontSize, Initial().FontSize)
	case "color":
		_, err = style.ParseColorValue(value)
	case "line-height":
		v, e := style.ParseBoxLength("width", value)
		if e == nil && (v.Kind == style.PercentageLength || v.Kind == style.RelativeLength && (v.Unit == "em" || v.Unit == "rem")) {
			return nil
		}
		_, err = lineHeight(value, Initial().FontSize, Initial().FontSize)
	default:
		_, err = ParseKeyword(key, value)
	}
	return err
}
