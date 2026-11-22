package css

import (
	"fmt"
	"math"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/styledtree"
)

// FontContext supplies computed source font sizes in DOM dimen.DU, typically
// from style/textstyle snapshots. RootFontSize is the document element's computed
// font size, including when resolving a border on that element itself.
// FontSize returns the requested element's computed size in its original source
// ancestry. It may be called for an ancestor when a border width is inherited.
// It must not substitute the selected element's or a receiving frame's font size.
//
// Only the context needed by a nonzero relative width is consulted: em uses
// FontSize, rem uses RootFontSize. Absolute/zero widths and none/hidden borders
// need neither. A used font size must be positive and below dimen.Fil. Callers
// may supply a lookup into their already resolved source snapshots; no fonts or
// containing geometry are required. The context is not retained after the call.
type FontContext struct {
	FontSize     func(*styledtree.StyNode) (dimen.DU, error)
	RootFontSize dimen.DU
}

// GetBorderWithFontContext resolves one physical border, adding em/rem support
// to GetBorder. An inherited width is computed at its source ancestor before
// inheritance, including that ancestor's effective zero for none/hidden.
// Border style, color/currentcolor, defaults and CSS-wide keywords retain the
// existing GetBorder semantics. Nonzero relative units other than em/rem,
// invalid font context and dimensions outside the finite DU range are errors.
// Relative results round to nearest DU, with half units rounded upward.
// Failures return a zero Border and preserve font-lookup error causes.
// Keep the source styled tree stable throughout the call.
func GetBorderWithFontContext(node *styledtree.StyNode, side string, fonts FontContext) (Border, error) {
	return getBorder(node, side, &fonts)
}

func (fonts *FontContext) borderWidth(node *styledtree.StyNode, key string, width style.BoxLength) (dimen.DU, error) {
	if width.Kind != style.RelativeLength || (width.Unit != "em" && width.Unit != "rem") {
		return 0, fmt.Errorf("%w: %s requires absolute, em or rem units", style.ErrInvalidValue, key)
	}
	fontSize := fonts.RootFontSize
	if width.Unit == "em" {
		if fonts.FontSize == nil {
			return 0, fmt.Errorf("%w: %s requires a source font-size lookup", style.ErrInvalidValue, key)
		}
		var err error
		fontSize, err = fonts.FontSize(node)
		if err != nil {
			return 0, fmt.Errorf("%s: source font-size: %w", key, err)
		}
	}
	if fontSize <= 0 || fontSize >= dimen.Fil {
		return 0, fmt.Errorf("%w: %s requires a positive finite %s font size", style.ErrInvalidValue, key, width.Unit)
	}
	value := math.Round(width.Magnitude * float64(fontSize))
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= float64(dimen.Fil) {
		return 0, fmt.Errorf("%w: %s exceeds the finite dimension range", style.ErrInvalidValue, key)
	}
	return dimen.DU(value), nil
}
