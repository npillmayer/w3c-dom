package css_test

import (
	"testing"

	"github.com/npillmayer/w3c-dom/style/css"
	"github.com/npillmayer/w3c-dom/style/display"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

// HTML defaults must resolve to distinct normalized roles through the public
// accessor, without imposing a downstream table layout model.
func TestTableDisplayDefaults(t *testing.T) {
	for _, tc := range []struct {
		tag, spelling string
		role          display.InternalRole
	}{
		{"caption", "table-caption", display.TableCaption}, {"tr", "table-row", display.TableRow},
		{"td", "table-cell", display.TableCell}, {"th", "table-cell", display.TableCell},
		{"thead", "table-header-group", display.TableHeaderGroup}, {"tbody", "table-row-group", display.TableRowGroup},
		{"tfoot", "table-footer-group", display.TableFooterGroup}, {"col", "table-column", display.TableColumn},
		{"colgroup", "table-column-group", display.TableColumnGroup},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			node := styledtree.NewNodeForHTMLNode(&html.Node{Type: html.ElementNode, Data: tc.tag}).Payload
			got, err := css.GetDisplay(node)
			if err != nil || got.Kind() != display.Internal || got.InternalRole() != tc.role || got.CSS() != tc.spelling || got.Outer() != display.NoOutside || got.Inner() != display.NoInside {
				t.Fatalf("GetDisplay(%s) = %v, %v", tc.tag, got, err)
			}
		})
	}
}
