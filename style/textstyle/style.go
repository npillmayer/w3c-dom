// Package textstyle resolves source text styles before layout or DOM detachment.
// It performs no font discovery, shaping, whitespace transformation or sizing of
// boxes. Applications impose their own content and layout profiles on its values.
package textstyle

import (
	"fmt"
	"image/color"

	"github.com/npillmayer/w3c-dom/dimen"
	"golang.org/x/net/html"
)

// Direction is the resolved source direction, independent of a bidi engine.
type Direction uint8

const (
	LeftToRight Direction = iota
	RightToLeft
)

// LineHeightKind distinguishes inherited factors from computed lengths.
type LineHeightKind uint8

const (
	NormalLineHeight LineHeightKind = iota
	FactorLineHeight
	LengthLineHeight
)

// LineHeight retains normal without consulting font metrics. A unitless Factor
// is inherited as a factor; percentages and lengths compute to Length in DOM DU
// on the declaring element and are inherited without recomputation.
type LineHeight struct {
	Kind   LineHeightKind
	Factor float64
	Length dimen.DU
}

// Style is a detached, comparable source snapshot. FontSize is positive DOM DU
// (65536 per big point), not CSS pixels or another library's dimension type.
// Family retains a checked CSS family list; Color retains its checked spelling
// (with identifiers decoded) for string-based adapters. Foreground is the same
// resolved color in non-premultiplied sRGB. Neither color contains currentcolor.
// Language is a canonical BCP 47 tag, defaulting to und. UnicodeBidi belongs to
// the element; text nodes carry their parent's complete text context.
type Style struct {
	Family      string
	FontSize    dimen.DU
	Weight      int
	Slant       string
	Color       string
	Foreground  color.NRGBA
	Language    string
	Direction   Direction
	WhiteSpace  string
	UnicodeBidi string
	LineHeight  LineHeight
	Hyphens     string
}

// Initial returns the print UA's text defaults. This service uses these fixed
// defaults, matching the original tytui adapter; CSSOM custom UA overrides are
// not a separate input to this text-style profile.
func Initial() Style {
	return Style{
		Family: "serif", FontSize: 16 * dimen.PX, Weight: 400, Slant: "normal",
		Color: "black", Foreground: color.NRGBA{A: 255}, Language: "und",
		WhiteSpace: "normal", UnicodeBidi: "normal", Hyphens: "manual",
	}
}

// Context supplies already computed source values for ResolveWithContext.
// Parent must be a resolved Style (Initial for the document/root). RootFontSize
// is the document element's computed size, or Initial().FontSize while resolving
// that element's font-size. It is never a receiving frame's font size.
type Context struct {
	Parent       Style
	RootFontSize dimen.DU
}

// Error identifies an invalid effective value without parsing diagnostic prose.
// Node is the original source node, for the caller to map to its own identity.
// Errors retain that source pointer; successful Style values retain no DOM.
type Error struct {
	Node     *html.Node
	Property string
	Value    string
	Err      error
}

func (e *Error) Error() string {
	return fmt.Sprintf("text style: %s %q: %v", e.Property, e.Value, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }
