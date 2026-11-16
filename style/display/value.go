// Package display parses and normalizes CSS Display Level 3 values independently
// of the DOM, cascade, and layout engines. Successful parsing does not imply
// layout support. CSS-wide keywords and context-dependent display transformations
// belong to later cascade/layout stages, not this package.
package display

// Kind identifies the variant of a Value. Special variants are not outer/inner
// pairs; inspect InternalRole for layout-internal values.
type Kind uint8

const (
	Invalid  Kind = iota // Invalid is the zero/uninitialized value.
	Box                  // Box is an outer/inner pair, optionally with a list marker.
	None                 // None suppresses the element's subtree.
	Contents             // Contents suppresses the principal box.
	Internal             // Internal describes a table- or ruby-internal role.
)

// Outside is participation in the surrounding flow, for Kind Box only.
type Outside uint8

const (
	NoOutside Outside = iota // NoOutside means not applicable, not an implicit block.
	Block
	Inline
	RunIn
)

// Inside is the contents' layout model, for Kind Box only. Flow does not decide
// whether the eventual contents form block children or an inline group.
type Inside uint8

const (
	NoInside Inside = iota // NoInside means not applicable.
	Flow
	FlowRoot
	Table
	Flex
	Grid
	Ruby
)

// InternalRole identifies a layout-specific value without pretending that its
// role is an ordinary outer/inner pair. Used inner/outer semantics of these roles
// (e.g. the flow-root interior of a table-cell) belong to their layout models.
type InternalRole uint8

const (
	NoInternalRole InternalRole = iota // NoInternalRole means not applicable.
	TableRowGroup
	TableHeaderGroup
	TableFooterGroup
	TableRow
	TableCell
	TableColumnGroup
	TableColumn
	TableCaption
	RubyBase
	RubyText
	RubyBaseContainer
	RubyTextContainer
)

// Value is an immutable, comparable normalized display value. Parse is its only
// constructor. Equivalent spellings compare equal using == and may be map keys.
// The zero Value is invalid; it never means display:block or display:inline.
type Value struct {
	kind     Kind
	outer    Outside
	inner    Inside
	role     InternalRole
	listItem bool
}

// Valid reports whether v is an initialized display value.
func (v Value) Valid() bool { return v.kind != Invalid }

// Kind returns the variant, or Invalid for the zero Value.
func (v Value) Kind() Kind { return v.kind }

// Outer returns the ordinary pair's outer type, or NoOutside for other kinds.
func (v Value) Outer() Outside { return v.outer }

// Inner returns the ordinary pair's inner type, or NoInside for other kinds.
func (v Value) Inner() Inside { return v.inner }

// InternalRole returns the internal role, or NoInternalRole for other kinds.
func (v Value) InternalRole() InternalRole { return v.role }

// IsListItem reports whether the ordinary pair includes the list-item flag.
func (v Value) IsListItem() bool { return v.listItem }

// CSS serializes v using canonical keyword order and the backwards-compatible
// short forms (e.g. inline-block rather than inline flow-root). Invalid values
// serialize to the empty string, which Parse rejects. CSS is not a debug dump.
func (v Value) CSS() string {
	switch v.kind {
	case None:
		return "none"
	case Contents:
		return "contents"
	case Internal:
		return internalNames[v.role]
	case Box:
		outer, inner := outsideNames[v.outer], insideNames[v.inner]
		if v.listItem {
			s := ""
			if v.outer != Block {
				s = outer + " "
			}
			if v.inner != Flow {
				s += inner + " "
			}
			return s + "list-item"
		}
		if v.inner == Flow {
			return outer
		}
		if v.inner == Ruby {
			if v.outer == Inline {
				return "ruby"
			}
			return outer + " ruby"
		}
		if v.outer == Block {
			return inner
		}
		if v.outer == Inline {
			if v.inner == FlowRoot {
				return "inline-block"
			}
			return "inline-" + inner
		}
		return outer + " " + inner
	default:
		return ""
	}
}

// String is a diagnostic representation. Use CSS when serializing a property.
func (v Value) String() string {
	if !v.Valid() {
		return "display.Value(invalid)"
	}
	return "display.Value(" + v.CSS() + ")"
}

var outsideNames = [...]string{"", "block", "inline", "run-in"}
var insideNames = [...]string{"", "flow", "flow-root", "table", "flex", "grid", "ruby"}
var internalNames = [...]string{"", "table-row-group", "table-header-group", "table-footer-group", "table-row", "table-cell", "table-column-group", "table-column", "table-caption", "ruby-base", "ruby-text", "ruby-base-container", "ruby-text-container"}
