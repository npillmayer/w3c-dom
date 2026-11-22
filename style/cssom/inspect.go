package cssom

import (
	"fmt"
	"sort"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/gorilla/css/scanner"
	"github.com/npillmayer/w3c-dom/style"
)

// Diagnostic reports a CSS inspection failure. Rule and Declaration are
// one-based source ordinals; zero means not applicable. Line and Column refer
// to the supplied CSS text, not the enclosing HTML document.
type Diagnostic struct {
	Code, Message     string
	Rule, Declaration int
	Line, Column      int
	Property          string
	Err               error
}

func (d Diagnostic) Error() string { return d.Message }
func (d Diagnostic) Unwrap() error { return d.Err }

// SourceDeclaration retains every declaration, including duplicates and invalid
// declarations, before cascade reduction. Valid denotes syntax and currently
// implemented box/display value checks, not support for every property's grammar.
type SourceDeclaration struct {
	Index            int
	Line, Column     int
	Text, Property   string
	Value            style.Property
	Important, Valid bool
}

// SourceRule retains a rule before matching. AtKeyword is empty for qualified
// rules. At-rules are retained for inspection but are unsupported by this engine.
type SourceRule struct {
	Index                    int
	Line, Column             int
	AtKeyword, Prelude, Text string
	Declarations             []SourceDeclaration
}

// StylesheetInspection separates source information from a stylesheet usable by
// the cascade. Sheet contains supported rules and valid declarations only.
// Strict consumers must reject Diagnostics instead of styling a successful subset.
// Keep the result read-only when sharing Sheet between independent CSSOMs.
type StylesheetInspection struct {
	Rules       []SourceRule
	Diagnostics []Diagnostic
	Sheet       StyleSheet
}

// InspectStylesheet inspects ordinary selector/declaration rules with the same
// declaration tokenizer used for inline styling. It reports malformed selectors,
// syntax failures and unsupported at-rules without logging or fetching resources.
// The legacy douceur adapter and Style entry point retain permissive behavior.
func InspectStylesheet(text string) StylesheetInspection {
	tokens, diagnostics := cssTokens(text)
	result := StylesheetInspection{Diagnostics: diagnostics}
	sheet := &inspectedSheet{}
	result.Sheet = sheet
	for pos := 0; pos < len(tokens); {
		if trivia(tokens[pos]) || tokens[pos].Type == scanner.TokenCDO || tokens[pos].Type == scanner.TokenCDC {
			pos++
			continue
		}
		start := pos
		rule := SourceRule{Index: len(result.Rules) + 1, Line: tokens[pos].Line, Column: tokens[pos].Column}
		if tokens[pos].Type == scanner.TokenAtKeyword {
			rule.AtKeyword = tokens[pos].Value
		}
		// Braces delimit rules; strings and URLs are single scanner tokens.
		for pos < len(tokens) && !char(tokens[pos], "{") && !char(tokens[pos], ";") && !char(tokens[pos], "}") {
			pos++
		}
		rule.Prelude = strings.TrimSpace(tokenText(tokens[start:pos]))
		if pos == len(tokens) || char(tokens[pos], "}") || char(tokens[pos], ";") && rule.AtKeyword == "" {
			result.Diagnostics = append(result.Diagnostics, ruleDiagnostic(rule, "css.rule-syntax", "expected a declaration block", nil))
			if pos < len(tokens) {
				pos++
			}
			rule.Text = tokenText(tokens[start:pos])
			result.Rules = append(result.Rules, rule)
			continue
		}
		if char(tokens[pos], ";") { // An at-rule without a block.
			pos++
		} else {
			pos++
			body := pos
			depth := 1
			for pos < len(tokens) && depth != 0 {
				if char(tokens[pos], "{") {
					depth++
				}
				if char(tokens[pos], "}") {
					depth--
				}
				if depth != 0 {
					pos++
				}
			}
			if depth != 0 {
				result.Diagnostics = append(result.Diagnostics, ruleDiagnostic(rule, "css.rule-syntax", "unclosed declaration block", nil))
			}
			if rule.AtKeyword == "" {
				var ds []Diagnostic
				rule.Declarations, ds = inspectDeclarations(tokens[body:pos], rule.Index)
				result.Diagnostics = append(result.Diagnostics, ds...)
			}
			if pos < len(tokens) {
				pos++
			}
		}
		rule.Text = tokenText(tokens[start:pos])
		result.Rules = append(result.Rules, rule)
		if rule.AtKeyword != "" {
			result.Diagnostics = append(result.Diagnostics, ruleDiagnostic(rule, "css.unsupported-rule", "unsupported at-rule "+rule.AtKeyword, nil))
			continue
		}
		if _, err := cascadia.ParseGroup(rule.Prelude); err != nil {
			result.Diagnostics = append(result.Diagnostics, ruleDiagnostic(rule, "css.selector-syntax", "invalid selector: "+err.Error(), err))
			continue
		}
		sheet.rules = append(sheet.rules, inspectedRule{rule})
	}
	orderDiagnostics(result.Diagnostics)
	return result
}

// InspectInlineStyle retains source-ordered declarations and reports errors
// that permissive inline styling would discard. Rule ordinals are zero.
func InspectInlineStyle(text string) ([]SourceDeclaration, []Diagnostic) {
	tokens, diagnostics := cssTokens(text)
	declarations, ds := inspectDeclarations(tokens, 0)
	diagnostics = append(diagnostics, ds...)
	orderDiagnostics(diagnostics)
	return declarations, diagnostics
}

func orderDiagnostics(ds []Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].Line != ds[j].Line {
			return ds[i].Line < ds[j].Line
		}
		return ds[i].Column < ds[j].Column
	})
}

func cssTokens(text string) ([]*scanner.Token, []Diagnostic) {
	var tokens []*scanner.Token
	lexer := scanner.New(text)
	for {
		t := lexer.Next()
		if t.Type == scanner.TokenEOF {
			return tokens, nil
		}
		if t.Type == scanner.TokenError {
			err := fmt.Errorf("CSS tokenizer: %s", t.String())
			return tokens, []Diagnostic{{Code: "css.token-syntax", Message: err.Error(), Line: t.Line, Column: t.Column, Err: err}}
		}
		tokens = append(tokens, t)
	}
}

func inspectDeclarations(tokens []*scanner.Token, rule int) ([]SourceDeclaration, []Diagnostic) {
	var declarations []SourceDeclaration
	var diagnostics []Diagnostic
	for start := 0; start < len(tokens); {
		if trivia(tokens[start]) || char(tokens[start], ";") {
			start++
			continue
		}
		end := start
		var stack []string
		balanced := true
		for end < len(tokens) {
			t := tokens[end]
			if char(t, ";") && len(stack) == 0 {
				break
			}
			switch {
			case t.Type == scanner.TokenFunction || char(t, "("):
				stack = append(stack, ")")
			case char(t, "["):
				stack = append(stack, "]")
			case char(t, "{"):
				stack = append(stack, "}")
			case char(t, ")") || char(t, "]") || char(t, "}"):
				if len(stack) == 0 || stack[len(stack)-1] != t.Value {
					balanced = false
				} else {
					stack = stack[:len(stack)-1]
				}
			}
			end++
		}
		part := tokens[start:end]
		d := SourceDeclaration{Index: len(declarations) + 1, Line: part[0].Line, Column: part[0].Column, Text: tokenText(part)}
		var significant []*scanner.Token
		for _, t := range part {
			if !trivia(t) {
				significant = append(significant, t)
			}
		}
		d.Valid = balanced && len(stack) == 0 && len(significant) >= 3 && significant[0].Type == scanner.TokenIdent && char(significant[1], ":")
		if len(significant) > 0 && significant[0].Type == scanner.TokenIdent {
			d.Property = significant[0].Value
		}
		if d.Valid {
			valueStart, valueEnd := 0, len(part)
			for i, t := range part {
				if t == significant[1] {
					valueStart = i + 1
					break
				}
			}
			last := len(significant) - 1
			if last >= 3 && significant[last].Type == scanner.TokenIdent && strings.EqualFold(significant[last].Value, "important") && char(significant[last-1], "!") {
				d.Important = true
				for i, t := range part {
					if t == significant[last-1] {
						valueEnd = i
						break
					}
				}
			}
			d.Value = style.Property(strings.TrimSpace(valueText(part[valueStart:valueEnd])))
			d.Valid = d.Value != ""
		}
		if !d.Valid {
			diagnostics = append(diagnostics, Diagnostic{Code: "css.declaration-syntax", Message: "malformed CSS declaration", Rule: rule, Declaration: d.Index, Line: d.Line, Column: d.Column, Property: d.Property})
		} else {
			if _, err := style.NormalizeDeclaration(d.Property, d.Value); err != nil {
				d.Valid = false
				diagnostics = append(diagnostics, Diagnostic{Code: "css.invalid-value", Message: err.Error(), Rule: rule, Declaration: d.Index, Line: d.Line, Column: d.Column, Property: d.Property, Err: err})
			}
		}
		declarations = append(declarations, d)
		start = end + 1
	}
	return declarations, diagnostics
}

func char(t *scanner.Token, value string) bool {
	return t.Type == scanner.TokenChar && t.Value == value
}
func trivia(t *scanner.Token) bool {
	return t.Type == scanner.TokenS || t.Type == scanner.TokenComment || t.Type == scanner.TokenBOM
}
func tokenText(tokens []*scanner.Token) string {
	var s strings.Builder
	for _, t := range tokens {
		s.WriteString(t.Value)
	}
	return s.String()
}
func valueText(tokens []*scanner.Token) string {
	var s strings.Builder
	for _, t := range tokens {
		if t.Type == scanner.TokenComment {
			s.WriteByte(' ')
		} else {
			s.WriteString(t.Value)
		}
	}
	return s.String()
}
func ruleDiagnostic(r SourceRule, code, message string, err error) Diagnostic {
	return Diagnostic{Code: code, Message: message, Rule: r.Index, Line: r.Line, Column: r.Column, Err: err}
}

// Reduce only after inspection. This is shared with inline styling so importance,
// duplicate order and known invalid values behave identically in both paths.
func localDeclarations(source []SourceDeclaration) localPseudoRuleType {
	var result localPseudoRuleType
	for _, d := range source {
		if !d.Valid {
			continue
		}
		key := d.Property
		if !strings.HasPrefix(key, "--") {
			key = style.CanonicalPropertyKey(strings.ToLower(key))
		}
		skip := false
		for i, old := range result {
			if old.Key == key {
				if old.important && !d.Important {
					skip = true
				} else {
					result = append(result[:i], result[i+1:]...)
				}
				break
			}
		}
		if !skip {
			result = append(result, localDeclaration{style.KeyValue{Key: key, Value: d.Value}, d.Important})
		}
	}
	return result
}

type inspectedRule struct{ SourceRule }

func (r inspectedRule) Selector() string     { return r.Prelude }
func (r inspectedRule) Properties() []string { return localDeclarations(r.Declarations).Properties() }
func (r inspectedRule) Value(key string) style.Property {
	return localDeclarations(r.Declarations).Value(key)
}
func (r inspectedRule) IsImportant(key string) bool {
	return localDeclarations(r.Declarations).IsImportant(key)
}

type inspectedSheet struct{ rules []Rule }

func (s *inspectedSheet) AppendRules(other StyleSheet) { s.rules = append(s.rules, other.Rules()...) }
func (s *inspectedSheet) Empty() bool                  { return len(s.rules) == 0 }
func (s *inspectedSheet) Rules() []Rule                { return append([]Rule(nil), s.rules...) }
