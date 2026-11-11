package css_test

import (
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestTableDisplayModes(t *testing.T) {
	cases := []struct {
		tag, value string
		mode       css.DisplayMode
	}{
		{"caption", "table-caption", css.TableCaptionMode}, {"tr", "table-row", css.TableRowMode},
		{"td", "table-cell", css.TableCellMode}, {"th", "table-cell", css.TableCellMode},
		{"thead", "table-header-group", css.TableHeaderGroupMode}, {"tbody", "table-row-group", css.TableRowGroupMode},
		{"tfoot", "table-footer-group", css.TableFooterGroupMode}, {"col", "table-column", css.TableColumnMode},
		{"colgroup", "table-column-group", css.TableColumnGroupMode},
	}
	for _, tc := range cases {
		raw := style.DisplayPropertyForHTMLNode(&html.Node{Type: html.ElementNode, Data: tc.tag})
		if string(raw) != tc.value {
			t.Fatalf("%s default=%s", tc.tag, raw)
		}
		got, err := css.ParseDisplay(string(raw))
		if err != nil || got != tc.mode {
			t.Fatalf("%s=%v %v", raw, got, err)
		}
		if got.Outer() != css.NoMode || got.Inner() != tc.mode || got.IsBlockLevel() {
			t.Errorf("wrong role flags for %s", raw)
		}
		if !got.Contains(tc.mode) || !got.Overlaps(tc.mode) || got.Overlaps(css.TableMode) {
			t.Errorf("wrong membership for %s", raw)
		}
		if got.String() != got.FullString() || strings.HasPrefix(got.String(), "DisplayMode(") || got.Symbol() == "?" {
			t.Errorf("missing diagnostics for %s", raw)
		}
		var set css.DisplayMode
		set.Set(tc.mode)
		if set != got {
			t.Errorf("Set lost %s", raw)
		}
		for _, other := range cases {
			if other.mode != tc.mode && got.Overlaps(other.mode) {
				t.Errorf("%s overlaps %s", raw, other.value)
			}
		}
	}
}
func TestExistingDisplayModes(t *testing.T) {
	for text, want := range map[string]css.DisplayMode{"": css.NoMode, "none": css.DisplayNone, "block": css.BlockMode | css.InnerBlockMode, "inline": css.InlineMode | css.InnerInlineMode, "block-inline": css.BlockMode | css.InnerInlineMode, "inline-block": css.InlineMode | css.InnerBlockMode, "list-item": css.BlockMode | css.ListItemMode, "table": css.BlockMode | css.TableMode, "inline-table": css.InlineMode | css.TableMode} {
		got, err := css.ParseDisplay(text)
		if err != nil || got != want {
			t.Errorf("%s=%v %v", text, got, err)
		}
	}
	if _, err := css.ParseDisplay("table-invalid"); err == nil {
		t.Fatal("accepted invalid display")
	}
	if css.BlockMode.String() != "BlockMode" || css.InnerInlineMode.String() != "InnerInlineMode" {
		t.Fatal("stale atomic names")
	}
}
