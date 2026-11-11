package css

import "strconv"

// String returns an atomic mode name, or a numeric description of combined flags.
func (mode DisplayMode) String() string {
	switch mode {
	case NoMode:
		return "NoMode"
	case DisplayNone:
		return "DisplayNone"
	case BlockMode:
		return "BlockMode"
	case InlineMode:
		return "InlineMode"
	case FlowRootMode:
		return "FlowRootMode"
	case ListItemMode:
		return "ListItemMode"
	case FlexMode:
		return "FlexMode"
	case GridMode:
		return "GridMode"
	case TableMode:
		return "TableMode"
	case InnerBlockMode:
		return "InnerBlockMode"
	case InnerInlineMode:
		return "InnerInlineMode"
	case TableCaptionMode:
		return "TableCaptionMode"
	case TableRowMode:
		return "TableRowMode"
	case TableCellMode:
		return "TableCellMode"
	case TableHeaderGroupMode:
		return "TableHeaderGroupMode"
	case TableRowGroupMode:
		return "TableRowGroupMode"
	case TableFooterGroupMode:
		return "TableFooterGroupMode"
	case TableColumnMode:
		return "TableColumnMode"
	case TableColumnGroupMode:
		return "TableColumnGroupMode"
	}
	return "DisplayMode(" + strconv.FormatUint(uint64(mode), 10) + ")"
}
