// Package csssyntax contains lexical helpers shared by CSS value parsers.
package csssyntax

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gorilla/css/scanner"
)

const Space = " \t\n\r\f"

func Preprocess(input string) (string, error) {
	if !utf8.ValidString(input) {
		return "", fmt.Errorf("CSS value is not UTF-8")
	}
	input = strings.ReplaceAll(input, "\r\n", "\n")
	return strings.NewReplacer("\r", "\n", "\f", "\n", "\x00", "\uFFFD").Replace(input), nil
}

// Clean removes comments without joining surrounding tokens.
func Clean(input string) (string, error) {
	input, err := Preprocess(input)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	l := scanner.New(input)
	for {
		t := l.Next()
		switch t.Type {
		case scanner.TokenEOF:
			return strings.Trim(b.String(), Space), nil
		case scanner.TokenError:
			return "", fmt.Errorf("invalid CSS token: %s", t)
		case scanner.TokenComment:
			b.WriteByte(' ')
		default:
			b.WriteString(t.Value)
		}
	}
}

// Ident accepts one identifier, decodes escapes and preserves name case.
func Ident(input string) (string, error) {
	s, err := Clean(input)
	if err != nil {
		return "", err
	}
	l := scanner.New(s)
	t := l.Next()
	// Gorilla's CSS2 scanner splits the leading hyphen of a CSS3 --identifier.
	if t.Type == scanner.TokenChar && t.Value == "-" && strings.HasPrefix(s, "--") {
		t = l.Next()
		if t.Type == scanner.TokenIdent && strings.HasPrefix(t.Value, "-") && l.Next().Type == scanner.TokenEOF {
			return "-" + DecodeIdent(t.Value), nil
		}
	}
	if t.Type != scanner.TokenIdent || l.Next().Type != scanner.TokenEOF {
		return "", fmt.Errorf("expected one CSS identifier: %q", input)
	}
	return DecodeIdent(t.Value), nil
}

func Lower(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, s)
}

// Components splits whitespace-separated values while keeping functions intact.
func Components(input string) ([]string, error) {
	s, err := Clean(input)
	if err != nil {
		return nil, err
	}
	l := scanner.New(s)
	var out []string
	var b strings.Builder
	depth := 0
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for {
		t := l.Next()
		if t.Type == scanner.TokenEOF {
			flush()
			return out, nil
		}
		if t.Type == scanner.TokenError {
			return nil, fmt.Errorf("invalid CSS component")
		}
		if t.Type == scanner.TokenS && depth == 0 {
			flush()
			continue
		}
		if t.Type == scanner.TokenFunction || t.Type == scanner.TokenChar && (t.Value == "(" || t.Value == "[") {
			depth++
		}
		if t.Type == scanner.TokenChar && (t.Value == ")" || t.Value == "]") {
			depth--
		}
		b.WriteString(t.Value)
	}
}

// DecodeIdent decodes a scanner-validated identifier without case folding.
func DecodeIdent(raw string) string {
	var b strings.Builder
	for i := 0; i < len(raw); {
		r, size := utf8.DecodeRuneInString(raw[i:])
		i += size
		if r == '\\' {
			var n rune
			digits := 0
			for i < len(raw) && digits < 6 {
				h, ok := hex(raw[i])
				if !ok {
					break
				}
				n = n*16 + rune(h)
				i++
				digits++
			}
			if digits > 0 {
				r = n
				if r == 0 || r > utf8.MaxRune || r >= 0xD800 && r <= 0xDFFF {
					r = utf8.RuneError
				}
				if i < len(raw) && strings.ContainsRune(Space, rune(raw[i])) {
					i++
				}
			} else if i < len(raw) {
				r, size = utf8.DecodeRuneInString(raw[i:])
				i += size
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

func hex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
