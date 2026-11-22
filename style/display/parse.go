package display

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gorilla/css/scanner"
	"github.com/npillmayer/w3c-dom/internal/csssyntax"
)

// ErrInvalidValue identifies invalid syntax and inputs that need another stage
// (CSS-wide keywords, unresolved variables, or complete declarations).
var ErrInvalidValue = errors.New("invalid display value")

// Parse accepts one UTF-8 CSS Display Level 3 property value, including comments
// and escaped, ASCII-case-insensitive identifiers. It accepts all Level 3 display
// roles, without promising layout support. It rejects empty input, block-inline,
// CSS-wide keywords, functions, declaration syntax, and !important. The caller
// must resolve cascade keywords and substitutions before calling Parse.
// Lexical errors, including unterminated comments, are rejected. On every error,
// Parse returns the invalid zero Value and an error wrapping ErrInvalidValue.
func Parse(input string) (Value, error) {
	words, err := declarationWords(input)
	if err != nil {
		return Value{}, err
	}
	return parseWords(words)
}

// declarationWords is shared by value parsing and CSS-wide declaration parsing.
func declarationWords(input string) ([]string, error) {
	fail := func(reason string) ([]string, error) { _, err := invalid(reason); return nil, err }
	if !utf8.ValidString(input) {
		return fail("input is not UTF-8")
	}
	// CSS input preprocessing. In particular CRLF is one newline, including
	// when it terminates a hexadecimal escape. The scanner otherwise only
	// normalizes CRLF, not bare CR or form feed.
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.NewReplacer("\r", "\n", "\f", "\n", "\x00", "\uFFFD").Replace(input)
	lexer := scanner.New(input)
	var words [3]string // grammar allows at most outside + inside + list-item
	count := 0
	for {
		token := lexer.Next()
		switch token.Type {
		case scanner.TokenEOF:
			return words[:count], nil
		case scanner.TokenS, scanner.TokenComment:
			continue
		case scanner.TokenIdent:
			if count == len(words) {
				return fail("too many keywords")
			}
			words[count] = identifier(token.Value)
			count++
		default:
			return fail(fmt.Sprintf("unexpected %s at %d:%d", token.Type, token.Line, token.Column))
		}
	}
}

func invalid(reason string) (Value, error) {
	return Value{}, fmt.Errorf("%w: %s", ErrInvalidValue, reason)
}

func parseWords(words []string) (Value, error) {
	if len(words) == 0 {
		return invalid("expected a keyword")
	}
	if len(words) == 1 {
		switch words[0] {
		case "none":
			return Value{kind: None}, nil
		case "contents":
			return Value{kind: Contents}, nil
		case "inline-block":
			return Value{kind: Box, outer: Inline, inner: FlowRoot}, nil
		case "inline-table":
			return Value{kind: Box, outer: Inline, inner: Table}, nil
		case "inline-flex":
			return Value{kind: Box, outer: Inline, inner: Flex}, nil
		case "inline-grid":
			return Value{kind: Box, outer: Inline, inner: Grid}, nil
		case "initial", "inherit", "unset", "revert", "revert-layer":
			return invalid("CSS-wide keyword requires cascade resolution")
		}
		for role := TableRowGroup; role <= RubyTextContainer; role++ {
			if words[0] == internalNames[role] {
				return Value{kind: Internal, role: role}, nil
			}
		}
	}
	v := Value{kind: Box}
	for _, word := range words {
		switch word {
		case "block", "inline", "run-in":
			if v.outer != NoOutside {
				return invalid("duplicate or conflicting outer types")
			}
			switch word {
			case "block":
				v.outer = Block
			case "inline":
				v.outer = Inline
			case "run-in":
				v.outer = RunIn
			}
		case "flow", "flow-root", "table", "flex", "grid", "ruby":
			if v.inner != NoInside {
				return invalid("duplicate or conflicting inner types")
			}
			switch word {
			case "flow":
				v.inner = Flow
			case "flow-root":
				v.inner = FlowRoot
			case "table":
				v.inner = Table
			case "flex":
				v.inner = Flex
			case "grid":
				v.inner = Grid
			case "ruby":
				v.inner = Ruby
			}
		case "list-item":
			if v.listItem {
				return invalid("duplicate list-item")
			}
			v.listItem = true
		default:
			return invalid(fmt.Sprintf("unexpected keyword %q", word))
		}
	}
	if v.inner == NoInside {
		v.inner = Flow
	}
	if v.listItem && v.inner != Flow && v.inner != FlowRoot {
		return invalid("list-item requires flow or flow-root")
	}
	if v.outer == NoOutside {
		v.outer = Block
		if v.inner == Ruby {
			v.outer = Inline
		}
	}
	return v, nil
}

// identifier decodes a scanner-validated IDENT and folds ASCII only.
func identifier(raw string) string { return csssyntax.Lower(csssyntax.DecodeIdent(raw)) }
