package css

import (
	"bytes"
	"fmt"
)

// DisplayMode is a type for CSS property "display".
type DisplayMode uint32

// Flags for box context and display mode (outer and inner).
const (
	NoMode               DisplayMode = iota   // unset or error condition
	DisplayNone          DisplayMode = 0x0001 // CSS outer display = none
	BlockMode            DisplayMode = 0x0002 // CSS block context (inner or outer)
	InlineMode           DisplayMode = 0x0004 // CSS inline context
	FlowRootMode         DisplayMode = 0x0010 // CSS flow-root display property
	ListItemMode         DisplayMode = 0x0020 // CSS list-item display
	FlexMode             DisplayMode = 0x0040 // CSS inner display = flex
	GridMode             DisplayMode = 0x0080 // CSS inner display = grid
	TableMode            DisplayMode = 0x0100 // CSS table display property (inner or outer)
	InnerBlockMode       DisplayMode = 0x0200 // CSS inner block mode (inline-block)
	InnerInlineMode      DisplayMode = 0x0400 // CSS inner inline mode (paragraphs)
	TableCaptionMode     DisplayMode = 0x800
	TableRowMode         DisplayMode = 0x1000
	TableCellMode        DisplayMode = 0x2000
	TableHeaderGroupMode DisplayMode = 0x4000
	TableRowGroupMode    DisplayMode = 0x8000
	TableFooterGroupMode DisplayMode = 0x10000
	TableColumnMode      DisplayMode = 0x20000
	TableColumnGroupMode DisplayMode = 0x40000
)

var allDisplayModes = []DisplayMode{
	DisplayNone, BlockMode, InlineMode, ListItemMode, FlowRootMode, FlexMode,
	GridMode, TableMode, InnerBlockMode, InnerInlineMode,
	TableCaptionMode, TableRowMode, TableCellMode, TableHeaderGroupMode, TableRowGroupMode, TableFooterGroupMode, TableColumnMode, TableColumnGroupMode,
}

// Outer returns outer mode
func (disp DisplayMode) Outer() DisplayMode {
	return disp & 0x000f
}

// Inner returns formatting-mode flags, including table-internal roles.
func (disp DisplayMode) Inner() DisplayMode {
	return disp &^ DisplayMode(0x000f)
}

// IsBlockLevel return true if it has outer display level of BlockMode.
//
// A block-level element is defined as (from the spec):
// Block-level elements are those elements of the source document that are formatted visually
// as blocks (e.g., paragraphs). The following values of the 'display' property make an element
// block-level: 'block', 'list-item', and 'table'.
func (disp DisplayMode) IsBlockLevel() bool {
	return disp&0x000f == BlockMode
}

/*
func (disp DisplayMode) BlockOrInline() DisplayMode {
	if disp.Overlaps(InlineMode) {
		return InlineMode
	}
	return BlockMode
}
*/

// Set sets a given atomic mode within this display mode.
func (disp *DisplayMode) Set(d DisplayMode) {
	*disp = (*disp) | d
}

// Contains checks if a display mode contains a given atomic mode.
// Returns false for d = NoMode.
func (disp DisplayMode) Contains(d DisplayMode) bool {
	return d != NoMode && (disp&d > 0)
}

// Overlaps returns true if a given display mode shares at least one atomic
// mode flag with disp (excluding NoMode).
func (disp DisplayMode) Overlaps(d DisplayMode) bool {
	for _, m := range allDisplayModes {
		if disp.Contains(m) && d.Contains(m) {
			return true
		}
	}
	return false
}

// FullString returns all atomic modes set in a display mode.
func (disp DisplayMode) FullString() string {
	var b bytes.Buffer
	first := true
	for _, m := range allDisplayModes {
		if disp.Contains(m) {
			if !first {
				b.WriteString(" ")
			}
			first = false
			b.WriteString(m.String())
		}
	}
	return b.String()
}

// Symbol returns a Unicode symbol for a mode.
func (disp DisplayMode) Symbol() string {
	//if disp == FlowMode {
	//return "\u25a7"
	//} else
	if disp.Contains(BlockMode) || disp.Contains(InnerBlockMode) {
		return "\u25a9"
	} else if disp.Contains(InlineMode) || disp.Contains(InnerInlineMode) {
		return "\u25ba"
	} else if disp.Contains(FlexMode) {
		return "\u25a4"
	} else if disp.Contains(GridMode) {
		return "\u25f0"
	} else if disp.Contains(ListItemMode) {
		return "\u25a3"
	} else if disp.Contains(TableMode | TableCaptionMode | TableRowMode | TableCellMode | TableHeaderGroupMode | TableRowGroupMode | TableFooterGroupMode | TableColumnMode | TableColumnGroupMode) {
		return "\u25a5"
	} else if disp == NoMode {
		return "–"
	}
	return "?"
}

// ParseDisplay returns mode flags from a display property string. Table parts
// have distinct role flags and no block/inline outer flag; table and inline-table
// retain their existing outer flags. Parsing does not implement table layout.
func ParseDisplay(display string) (DisplayMode, error) {
	if display == "" {
		return NoMode, nil
	}
	switch display {
	case "none":
		return DisplayNone, nil
	case "block":
		return BlockMode | InnerBlockMode, nil
	case "inline":
		return InlineMode | InnerInlineMode, nil
	case "list-item":
		return ListItemMode | BlockMode, nil
	case "block-inline":
		return BlockMode | InnerInlineMode, nil
	case "inline-block":
		return InlineMode | InnerBlockMode, nil
	case "table-caption":
		return TableCaptionMode, nil
	case "table-row":
		return TableRowMode, nil
	case "table-cell":
		return TableCellMode, nil
	case "table-header-group":
		return TableHeaderGroupMode, nil
	case "table-row-group":
		return TableRowGroupMode, nil
	case "table-footer-group":
		return TableFooterGroupMode, nil
	case "table-column":
		return TableColumnMode, nil
	case "table-column-group":
		return TableColumnGroupMode, nil
	case "table":
		return BlockMode | TableMode, nil
	case "inline-table":
		return InlineMode | TableMode, nil
	}
	return BlockMode, fmt.Errorf("Unknown display mode: %s", display)
}
