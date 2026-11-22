package textstyle

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/css/scanner"
	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/internal/csssyntax"
	"github.com/npillmayer/w3c-dom/style"
)

func invalid(key, value string) error {
	return fmt.Errorf("%w: %s %q", style.ErrInvalidValue, key, value)
}

func keyword(key, value, allowed string) (string, error) {
	word, err := csssyntax.Ident(value)
	word = csssyntax.Lower(word)
	if err == nil {
		for _, v := range strings.Fields(allowed) {
			if v == word {
				return word, nil
			}
		}
	}
	return "", invalid(key, value)
}

func validSize(d dimen.DU) bool { return d > 0 && d < dimen.Fil }

func rounded(value float64) (dimen.DU, error) {
	value = math.Round(value)
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= float64(dimen.Fil) {
		return 0, fmt.Errorf("%w: dimension outside finite nonnegative range", style.ErrInvalidValue)
	}
	return dimen.DU(value), nil
}

// ResolveFontSize checks an effective font-size with explicit parent/root font
// sizes. CSS-wide keywords belong to the cascade and must be resolved first.
// Supports absolute CSS units, em/rem/percent, size keywords, larger and smaller.
// Zero, metric-dependent units, viewport units and deferred math are unsupported.
func ResolveFontSize(value string, parent, root dimen.DU) (dimen.DU, error) {
	if !validSize(parent) || !validSize(root) {
		return 0, invalid("font context", value)
	}
	word, _ := csssyntax.Ident(value)
	word = csssyntax.Lower(word)
	var size dimen.DU
	var err error
	switch word {
	case "xx-small":
		size = 9 * dimen.PX
	case "x-small":
		size = 10 * dimen.PX
	case "small":
		size = 13 * dimen.PX
	case "medium":
		size = 16 * dimen.PX
	case "large":
		size = 18 * dimen.PX
	case "x-large":
		size = 24 * dimen.PX
	case "xx-large":
		size = 32 * dimen.PX
	case "larger":
		size, err = rounded(float64(parent) * 1.2)
	case "smaller":
		size, err = rounded(float64(parent) / 1.2)
	default:
		size, err = resolveLength(value, parent, root)
	}
	if err != nil || !validSize(size) {
		return 0, invalid("font-size", value)
	}
	return size, nil
}

func resolveLength(value string, em, rem dimen.DU) (dimen.DU, error) {
	// Reuse the DOM's checked nonnegative length grammar and absolute conversion.
	v, err := style.ParseBoxLength("width", value)
	if err != nil {
		return 0, err
	}
	switch v.Kind {
	case style.AbsoluteLength:
		return rounded(float64(v.Absolute))
	case style.PercentageLength:
		return rounded(v.Magnitude * float64(em) / 100)
	case style.RelativeLength:
		switch v.Unit {
		case "em":
			return rounded(v.Magnitude * float64(em))
		case "rem":
			return rounded(v.Magnitude * float64(rem))
		}
	}
	return 0, invalid("length", value)
}

var number = regexp.MustCompile(`^[+\-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)(?:[eE][+\-]?[0-9]+)?$`)

func lineHeight(value string, em, rem dimen.DU) (LineHeight, error) {
	if word, _ := keyword("line-height", value, "normal"); word == "normal" {
		return LineHeight{}, nil
	}
	s, err := csssyntax.Clean(value)
	if err != nil {
		return LineHeight{}, invalid("line-height", value)
	}
	if number.MatchString(s) {
		n, err := strconv.ParseFloat(s, 64)
		// Reject negative source magnitudes even if they underflow to negative zero.
		mantissa := strings.Split(strings.ToLower(s), "e")[0]
		negative := strings.HasPrefix(s, "-") && strings.Trim(mantissa, "-.0") != ""
		if err != nil || negative || math.IsInf(n, 0) || math.IsNaN(n) {
			return LineHeight{}, invalid("line-height", value)
		}
		return LineHeight{Kind: FactorLineHeight, Factor: n}, nil
	}
	d, err := resolveLength(s, em, rem)
	if err != nil {
		return LineHeight{}, invalid("line-height", value)
	}
	return LineHeight{Kind: LengthLineHeight, Length: d}, nil
}

func weight(value string, parent int) (int, error) {
	word, _ := csssyntax.Ident(value)
	switch csssyntax.Lower(word) {
	case "normal":
		return 400, nil
	case "bold":
		return 700, nil
	case "bolder":
		switch {
		case parent < 350:
			return 400, nil
		case parent < 550:
			return 700, nil
		case parent > 900:
			return parent, nil
		default:
			return 900, nil
		}
	case "lighter":
		switch {
		case parent < 100:
			return parent, nil
		case parent < 550:
			return 100, nil
		case parent < 750:
			return 400, nil
		default:
			return 700, nil
		}
	}
	s, err := csssyntax.Clean(value)
	if err == nil && number.MatchString(s) {
		n, err := strconv.ParseFloat(s, 64)
		if err == nil && n >= 1 && n <= 1000 && n == math.Trunc(n) {
			return int(n), nil
		}
	}
	return 0, invalid("font-weight", value)
}

func family(value string) (string, error) {
	s, err := csssyntax.Clean(value)
	if err != nil {
		return "", invalid("font-family", value)
	}
	l := scanner.New(s)
	count, quoted := 0, false
	for {
		t := l.Next()
		switch t.Type {
		case scanner.TokenS:
			continue
		case scanner.TokenIdent:
			if quoted || style.DeclarationKeyword(t.Value) != "" || csssyntax.Lower(csssyntax.DecodeIdent(t.Value)) == "default" {
				return "", invalid("font-family", value)
			}
			count++
		case scanner.TokenString:
			if count != 0 || len(t.Value) <= 2 {
				return "", invalid("font-family", value)
			}
			count, quoted = 1, true
		case scanner.TokenEOF:
			if count > 0 {
				return s, nil
			}
			return "", invalid("font-family", value)
		case scanner.TokenChar:
			if t.Value != "," || count == 0 {
				return "", invalid("font-family", value)
			}
			count, quoted = 0, false
		default:
			return "", invalid("font-family", value)
		}
	}
}
