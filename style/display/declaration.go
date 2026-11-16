package display

// WideKeyword is a CSS-wide declaration value. These require cascade resolution
// and are not normalized display Values. Parsing does not implement rollback.
type WideKeyword uint8

const (
	NoKeyword WideKeyword = iota // NoKeyword means an ordinary display Value.
	Initial
	Inherit
	Unset
	Revert
	RevertLayer
)

// String returns the canonical keyword, or empty for NoKeyword/unknown values.
func (k WideKeyword) String() string {
	switch k {
	case Initial:
		return "initial"
	case Inherit:
		return "inherit"
	case Unset:
		return "unset"
	case Revert:
		return "revert"
	case RevertLayer:
		return "revert-layer"
	default:
		return ""
	}
}

// ParseDeclaration validates a display declaration's value, after removal of
// !important. Success returns either a valid Value and NoKeyword, or the invalid
// zero Value and a CSS-wide keyword. Errors return both zero values and wrap
// ErrInvalidValue. Unlike Parse, this accepts CSS-wide keywords; it does not
// resolve them. Functions/substitutions remain outside the supported grammar.
func ParseDeclaration(input string) (Value, WideKeyword, error) {
	words, err := declarationWords(input)
	if err != nil {
		return Value{}, NoKeyword, err
	}
	if len(words) == 1 {
		for keyword := Initial; keyword <= RevertLayer; keyword++ {
			if words[0] == keyword.String() {
				return Value{}, keyword, nil
			}
		}
	}
	value, err := parseWords(words)
	return value, NoKeyword, err
}
