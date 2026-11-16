package display_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/w3c-dom/style/display"
)

func mustParse(t *testing.T, s string) display.Value {
	t.Helper()
	v, err := display.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return v
}

func checkPair(t *testing.T, v display.Value, outer display.Outside, inner display.Inside, list bool) {
	t.Helper()
	if !v.Valid() || v.Kind() != display.Box || v.Outer() != outer || v.Inner() != inner || v.IsListItem() != list || v.InternalRole() != display.NoInternalRole {
		t.Fatalf("wrong pair: %+v outer=%d inner=%d list=%v", v, v.Outer(), v.Inner(), v.IsListItem())
	}
}

func TestAllOuterInnerPairs(t *testing.T) {
	outers := []struct {
		name  string
		value display.Outside
	}{{"block", display.Block}, {"inline", display.Inline}, {"run-in", display.RunIn}}
	inners := []struct {
		name  string
		value display.Inside
	}{{"flow", display.Flow}, {"flow-root", display.FlowRoot}, {"table", display.Table}, {"flex", display.Flex}, {"grid", display.Grid}, {"ruby", display.Ruby}}
	canonical := [][]string{
		{"block", "flow-root", "table", "flex", "grid", "block ruby"},
		{"inline", "inline-block", "inline-table", "inline-flex", "inline-grid", "ruby"},
		{"run-in", "run-in flow-root", "run-in table", "run-in flex", "run-in grid", "run-in ruby"},
	}
	distinct := map[display.Value]bool{}
	for oi, o := range outers {
		for ii, i := range inners {
			t.Run(o.name+"/"+i.name, func(t *testing.T) {
				value := mustParse(t, o.name+" "+i.name)
				checkPair(t, value, o.value, i.value, false)
				if distinct[value] {
					t.Fatal("distinct pairs compare equal")
				}
				distinct[value] = true
				if got := value.CSS(); got != canonical[oi][ii] {
					t.Fatalf("CSS=%q want %q", got, canonical[oi][ii])
				}
				for _, spelling := range []string{i.name + " " + o.name, canonical[oi][ii]} {
					if other := mustParse(t, spelling); other != value {
						t.Fatalf("not equivalent: %s", spelling)
					}
				}
			})
		}
	}
	if len(distinct) != 18 {
		t.Fatal("incomplete pairs")
	}
}

func TestDefaultComponents(t *testing.T) {
	for _, tc := range []struct{ input, full string }{
		{"block", "block flow"}, {"inline", "inline flow"}, {"run-in", "run-in flow"},
		{"flow", "block flow"}, {"flow-root", "block flow-root"}, {"table", "block table"},
		{"flex", "block flex"}, {"grid", "block grid"}, {"ruby", "inline ruby"},
		{"list-item", "block flow list-item"}, {"flow list-item", "block flow list-item"},
		{"flow-root list-item", "block flow-root list-item"}, {"inline list-item", "inline flow list-item"},
		{"run-in list-item", "run-in flow list-item"},
	} {
		if a, b := mustParse(t, tc.input), mustParse(t, tc.full); a != b {
			t.Fatalf("%s != %s", tc.input, tc.full)
		}
	}
}

func TestListItemPermutations(t *testing.T) {
	for _, o := range []struct {
		name  string
		value display.Outside
	}{{"block", display.Block}, {"inline", display.Inline}, {"run-in", display.RunIn}} {
		for _, i := range []struct {
			name  string
			value display.Inside
		}{{"flow", display.Flow}, {"flow-root", display.FlowRoot}} {
			words := []string{o.name, i.name, "list-item"}
			canonical := ""
			if o.value != display.Block {
				canonical = o.name + " "
			}
			if i.value != display.Flow {
				canonical += i.name + " "
			}
			canonical += "list-item"
			want := mustParse(t, canonical)
			checkPair(t, want, o.value, i.value, true)
			for a := 0; a < 3; a++ {
				for b := 0; b < 3; b++ {
					if a == b {
						continue
					}
					c := 3 - a - b
					input := words[a] + " " + words[b] + " " + words[c]
					v := mustParse(t, input)
					if v != want || v.CSS() != canonical {
						t.Fatalf("list permutation %q: %v", input, v)
					}
				}
			}
		}
	}
}

func TestSpecialValuesAndRoles(t *testing.T) {
	for _, tc := range []struct {
		input string
		kind  display.Kind
		role  display.InternalRole
	}{
		{"none", display.None, display.NoInternalRole}, {"contents", display.Contents, display.NoInternalRole},
		{"table-row-group", display.Internal, display.TableRowGroup}, {"table-header-group", display.Internal, display.TableHeaderGroup},
		{"table-footer-group", display.Internal, display.TableFooterGroup}, {"table-row", display.Internal, display.TableRow},
		{"table-cell", display.Internal, display.TableCell}, {"table-column-group", display.Internal, display.TableColumnGroup},
		{"table-column", display.Internal, display.TableColumn}, {"table-caption", display.Internal, display.TableCaption},
		{"ruby-base", display.Internal, display.RubyBase}, {"ruby-text", display.Internal, display.RubyText},
		{"ruby-base-container", display.Internal, display.RubyBaseContainer}, {"ruby-text-container", display.Internal, display.RubyTextContainer},
	} {
		t.Run(tc.input, func(t *testing.T) {
			v := mustParse(t, tc.input)
			if !v.Valid() || v.Kind() != tc.kind || v.InternalRole() != tc.role || v.Outer() != display.NoOutside || v.Inner() != display.NoInside || v.IsListItem() || v.CSS() != tc.input {
				t.Fatalf("wrong special value: %v", v)
			}
			for _, extra := range []string{"block", "flow", "list-item", tc.input} {
				assertInvalid(t, tc.input+" "+extra)
				assertInvalid(t, extra+" "+tc.input)
			}
		})
	}
}

func TestIdentifierTokenization(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{" \t\r\nBlOcK\f FLOW ", "block"},
		{"/*leading*/inline/**/flow-root/*trailing*/", "inline-block"},
		{"list-item/**/flow-root/**/inline", "inline flow-root list-item"},
		{`\62 lock`, "block"}, {`\000062lock`, "block"}, {`\000062 lock`, "block"},
		{`b\6c ock`, "block"}, {`bl\ock`, "block"}, {`bloc\k`, "block"},
		{`inline\2d block`, "inline-block"}, {`\69 nline \66 low-root`, "inline-block"},
		{`\42 LOCK`, "block"}, {"\\62\r\nlock", "block"}, {"\\62\rlock", "block"}, {"\\62\flock", "block"},
		{`\6e one`, "none"}, {`table-\63 ell`, "table-cell"},
		{"block/* non-ASCII comment: 世界 */flow", "block"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if v, w := mustParse(t, tc.input), mustParse(t, tc.want); v != w {
				t.Fatalf("%q: %v != %v", tc.input, v, w)
			}
		})
	}
}

func assertInvalid(t *testing.T, input string) {
	t.Helper()
	v, err := display.Parse(input)
	if !errors.Is(err, display.ErrInvalidValue) || v != (display.Value{}) {
		t.Fatalf("Parse(%q) = %v, %v; want zero and ErrInvalidValue", input, v, err)
	}
}

func TestInvalidValues(t *testing.T) {
	for _, input := range []string{
		"", " \t\n", "/**/", "block-inline", "inline-root", "inline-ruby", "bogus", "math",
		"block inline", "flow grid", "block block", "flow flow", "list-item list-item",
		"block flow flow-root", "inline flow list-item block", "none contents", "none block",
		"inline-block flow", "inline-block list-item", "block inline-flex", "flex list-item",
		"ruby list-item", "table list-item", "list-item inline grid", "table-cell list-item",
		"initial", "inherit", "unset", "revert", "revert-layer", `\69 nherit`, "inherit block",
		"var(--display)", "env(display)", "display:block", "block;", "block !important", "block,inline",
		`"block"`, "'inline'", "123", "10px", "#block", "@block", "{block}", "block()",
		"block/*", "/*unterminated", "bl/**/ock", `block\`, `\`, "blo\\\nck", `\62  lock`,
		`\0000620lock`, `\0 block`, `\D800 block`, `\110000 block`,
		"blocK", "block\u00a0flow", "block\vflow", "\ufeffblock", "bl\x00ock", "block\xff",
	} {
		t.Run(input, func(t *testing.T) { assertInvalid(t, input) })
	}
	// Every inner model outside flow/flow-root is disallowed with list-item.
	for _, inner := range []string{"table", "flex", "grid", "ruby"} {
		for _, outer := range []string{"block", "inline", "run-in"} {
			assertInvalid(t, outer+" "+inner+" list-item")
		}
	}
}

func TestZeroAndValueSemantics(t *testing.T) {
	var zero display.Value
	if zero.Valid() || zero.Kind() != display.Invalid || zero.Outer() != display.NoOutside || zero.Inner() != display.NoInside || zero.InternalRole() != display.NoInternalRole || zero.IsListItem() || zero.CSS() != "" {
		t.Fatalf("bad zero value: %v", zero)
	}
	if zero.String() == "" {
		t.Fatal("missing invalid diagnostic")
	}
	assertInvalid(t, zero.CSS())
	value := mustParse(t, "inline-block")
	copy := value
	// A new Parse does not mutate previously returned values.
	other := mustParse(t, "block flow-root")
	if value != copy || value == other || copy.CSS() != "inline-block" {
		t.Fatal("value semantics lost")
	}
	if value.String() == value.CSS() || !strings.Contains(value.String(), "inline-block") {
		t.Fatal("CSS and diagnostics must be distinct")
	}
}

func ExampleParse() {
	a, _ := display.Parse("inline flow-root")
	b, _ := display.Parse("inline-block")
	fmt.Println(a == b, a.CSS())
	fmt.Println(a.Kind() == display.Box, a.Outer() == display.Inline, a.Inner() == display.FlowRoot)
	// Output:
	// true inline-block
	// true true true
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"", "block", "inline flow-root", "list-item inline flow-root", "table-cell", "none", "contents", "ruby", "run-in grid", `\62 lock`, "block/**/flow", "block/*", "blocK", "inherit", "\xff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := display.Parse(s)
		if err != nil {
			if v != (display.Value{}) || !errors.Is(err, display.ErrInvalidValue) {
				t.Fatalf("invalid error result: %v %v", v, err)
			}
			return
		}
		if !v.Valid() || v.CSS() == "" {
			t.Fatal("success returned invalid value")
		}
		again, err := display.Parse(v.CSS())
		if err != nil || again != v {
			t.Fatalf("round trip: %q -> %v -> %v (%v)", s, v, again, err)
		}
		if repeated, err := display.Parse(s); err != nil || repeated != v {
			t.Fatal("non-deterministic Parse")
		}
	})
}
