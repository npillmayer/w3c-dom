package style

import (
	"errors"
	"fmt"
	"github.com/gorilla/css/scanner"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/internal/csssyntax"
	"github.com/npillmayer/w3c-dom/style/display"
)

// ErrInvalidValue covers invalid or unsupported values in the checked box subset.
var ErrInvalidValue = errors.New("invalid or unsupported CSS value")

type LengthKind uint8

const (
	InvalidLength LengthKind = iota
	AbsoluteLength
	PercentageLength
	RelativeLength
	KeywordLength
)

// BoxLength is a checked, unsized value. Absolute is in dimen.DU, not CSS pixels.
// Percentages retain their fractional percentage points without clamping.
// Relative lengths retain Magnitude and Unit; Keyword is auto/none or intrinsic.
// The zero value is invalid. No used-value calculation is performed here.
type BoxLength struct {
	Kind          LengthKind
	Absolute      dimen.DU
	Magnitude     float64
	Unit, Keyword string
}

var boxNumber = regexp.MustCompile(`^([+\-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)(?:[eE][+\-]?[0-9]+)?)(%|[a-zA-Z]+)?$`)

func invalidValue(key, input string) error {
	return fmt.Errorf("%w: %s: %q", ErrInvalidValue, key, input)
}

func borderSideProperty(key, suffix string) bool {
	for _, side := range []string{"top", "right", "bottom", "left"} {
		if key == "border-"+side+"-"+suffix {
			return true
		}
	}
	return false
}

func lengthProperty(key string) bool {
	switch key {
	case "width", "height", "min-width", "min-height", "max-width", "max-height", "top", "right", "bottom", "left":
		return true
	}
	for _, side := range []string{"top", "right", "bottom", "left"} {
		if key == "padding-"+side || key == "margin-"+side {
			return true
		}
	}
	return borderSideProperty(key, "width")
}

// ParseBoxLength checks a supported physical size, inset, margin, padding or
// border-width longhand. CSS-wide keywords must be resolved by the caller.
// Absolute CSS units, relative units and percentages are preserved distinctly.
// Nonzero unitless lengths and legacy bp/sp units are rejected in this API.
func ParseBoxLength(key, input string) (BoxLength, error) {
	key = CanonicalPropertyKey(csssyntax.Lower(key))
	bad := func() (BoxLength, error) { return BoxLength{}, invalidValue(key, input) }
	if !lengthProperty(key) {
		return bad()
	}
	s, err := csssyntax.Clean(input)
	if err != nil {
		return bad()
	}
	word, _ := csssyntax.Ident(s)
	word = csssyntax.Lower(word)
	isBorder := borderSideProperty(key, "width")
	isMargin := strings.HasPrefix(key, "margin-")
	isInset := key == "top" || key == "right" || key == "bottom" || key == "left"
	isSize := key == "width" || key == "height" || strings.HasPrefix(key, "min-") || strings.HasPrefix(key, "max-")
	if word == "auto" && (isMargin || isInset || isSize && !strings.HasPrefix(key, "max-")) || word == "none" && strings.HasPrefix(key, "max-") || isSize && (word == "min-content" || word == "max-content" || word == "fit-content" || word == "stretch") {
		return BoxLength{Kind: KeywordLength, Keyword: word}, nil
	}
	if isBorder {
		// Retain this module's UA choices for border keyword widths.
		for i, k := range []string{"thin", "medium", "thick"} {
			if word == k {
				return BoxLength{Kind: AbsoluteLength, Absolute: []dimen.DU{dimen.PX / 2, dimen.PX, 2 * dimen.PX}[i]}, nil
			}
		}
	}
	m := boxNumber.FindStringSubmatch(s)
	if m == nil {
		return bad()
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
		return bad()
	}
	// Check the source sign before DU rounding or float underflow can erase it.
	mantissa := strings.Split(strings.ToLower(m[1]), "e")[0]
	nonzero := strings.Trim(mantissa, "+-.0") != ""
	if strings.HasPrefix(m[1], "-") && nonzero && !isMargin && !isInset {
		return bad()
	}
	unit := csssyntax.Lower(m[2])
	if unit == "" {
		if nonzero {
			return bad()
		}
		return BoxLength{Kind: AbsoluteLength}, nil
	}
	if unit == "%" {
		if isBorder {
			return bad()
		}
		return BoxLength{Kind: PercentageLength, Magnitude: n, Unit: unit}, nil
	}
	switch unit {
	case "em", "ex", "ch", "rem", "vw", "vh", "vmin", "vmax":
		return BoxLength{Kind: RelativeLength, Magnitude: n, Unit: unit}, nil
	case "px", "pt", "pc", "in", "cm", "mm", "q":
		d, err := dimen.ParseAbsolute(m[1], unit)
		if err != nil {
			return BoxLength{}, fmt.Errorf("%w: %s: %v", ErrInvalidValue, key, err)
		}
		return BoxLength{Kind: AbsoluteLength, Absolute: d}, nil
	}
	return bad()
}

// ColorValue keeps currentcolor symbolic until its element's color is known.
type ColorValue struct {
	Color   color.NRGBA
	Current bool
}

func ParseColorValue(input string) (ColorValue, error) {
	s, err := csssyntax.Clean(input)
	if err != nil {
		return ColorValue{}, invalidValue("color", input)
	}
	if name, e := csssyntax.Ident(s); e == nil {
		s = csssyntax.Lower(name)
		if s == "currentcolor" {
			return ColorValue{Current: true}, nil
		}
	}
	c, err := ParseColor(s)
	if err != nil {
		return ColorValue{}, invalidValue("color", input)
	}
	return ColorValue{Color: c}, nil
}

// ParseFlowName returns a decoded, case-sensitive custom identifier. Empty
// means none. CSS-wide keywords and the reserved name default are not flow names.
// This checks the simple named-flow form, not flow-into's optional modifiers.
func ParseFlowName(input string) (string, error) {
	name, err := csssyntax.Ident(input)
	if err != nil {
		return "", invalidValue("flow name", input)
	}
	switch csssyntax.Lower(name) {
	case "none":
		return "", nil
	case "initial", "inherit", "unset", "revert", "revert-layer", "default":
		return "", invalidValue("flow name", input)
	}
	return name, nil
}

var boxKeywords = map[string]string{
	"box-sizing":   "content-box border-box",
	"writing-mode": "horizontal-tb vertical-rl vertical-lr sideways-rl sideways-lr",
	"direction":    "ltr rtl",
	"position":     "static relative absolute fixed sticky",
	"float":        "none left right inline-start inline-end",
	"visibility":   "visible hidden collapse",
}

// ParseBoxKeyword checks a supported box keyword longhand, including border style.
func ParseBoxKeyword(key, input string) (string, error) {
	key = csssyntax.Lower(key)
	allowed := boxKeywords[key]
	if borderSideProperty(key, "style") {
		allowed = "none hidden dotted dashed solid double groove ridge inset outset"
	}
	word, err := csssyntax.Ident(input)
	word = csssyntax.Lower(word)
	if err == nil && allowed != "" {
		for _, v := range strings.Fields(allowed) {
			if v == word {
				return word, nil
			}
		}
	}
	return "", invalidValue(key, input)
}

// DeclarationKeyword decodes a CSS-wide keyword; other values return empty.
func DeclarationKeyword(input string) string {
	word, err := csssyntax.Ident(input)
	if err != nil {
		return ""
	}
	switch word = csssyntax.Lower(word); word {
	case "initial", "inherit", "unset", "revert", "revert-layer":
		return word
	}
	return ""
}

// NormalizeDeclaration validates the supported longhands before cascade reduction.
// Unknown properties and unimplemented shorthand grammars remain permissive.
// CSS-wide keywords are normalized; revert/revert-layer still require a cascade
// capability this engine lacks and checked accessors will report them explicitly.
func NormalizeDeclaration(key string, value Property) (Property, error) {
	key = CanonicalPropertyKey(csssyntax.Lower(key))
	if key == "display" {
		_, wide, err := display.ParseDeclaration(string(value))
		if err != nil {
			return NullStyle, err
		}
		if wide != display.NoKeyword {
			return Property(wide.String()), nil
		}
		return value, nil
	}
	quad := key == "margin" || key == "margins" || key == "padding" || key == "border-width" || key == "border-style" || key == "border-color"
	known := quad || lengthProperty(key) || key == "color" || key == "background-color" || borderSideProperty(key, "color") || key == "flow-from" || key == "flow-into" || boxKeywords[key] != "" || borderSideProperty(key, "style")
	if !known {
		return value, nil
	}
	if word := DeclarationKeyword(string(value)); word != "" {
		return Property(word), nil
	}
	// Keep deferred math/substitutions opaque in the permissive cascade. Checked
	// accessors reject them until a caller supplies the missing evaluation stage.
	if deferredValue(string(value), lengthProperty(key) || quad) {
		return value, nil
	}
	var err error
	switch {
	case quad:
		var expanded []KeyValue
		expanded, err = SplitCompoundProperty(key, value)
		if err == nil {
			for _, kv := range expanded {
				if _, err = NormalizeDeclaration(kv.Key, kv.Value); err != nil {
					break
				}
			}
		}
	case lengthProperty(key):
		_, err = ParseBoxLength(key, string(value))
	case key == "color" || key == "background-color" || borderSideProperty(key, "color"):
		_, err = ParseColorValue(string(value))
	case key == "flow-from" || key == "flow-into":
		_, err = ParseFlowName(string(value))
	default:
		var word string
		word, err = ParseBoxKeyword(key, string(value))
		if err == nil {
			return Property(word), nil
		}
	}
	if err != nil {
		return NullStyle, err
	}
	return value, nil
}

func deferredValue(input string, mathAllowed bool) bool {
	s, err := csssyntax.Clean(input)
	if err != nil {
		return false
	}
	t := scanner.New(s).Next()
	if t.Type != scanner.TokenFunction {
		return false
	}
	name := csssyntax.Lower(csssyntax.DecodeIdent(strings.TrimSuffix(t.Value, "(")))
	switch name {
	case "var", "env":
		return true
	case "calc", "min", "max", "clamp":
		return mathAllowed
	}
	return false
}
