package css

import (
	"fmt"
	"image/color"

	"github.com/npillmayer/w3c-dom/dimen"
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

func elementProperty(node *styledtree.StyNode, key string) (style.Property, error) {
	if node == nil || node.Payload == nil || node.HTMLNode() == nil || node.HTMLNode().Type != html.ElementNode {
		return style.NullStyle, fmt.Errorf("%s requires an element node", key)
	}
	return GetProperty(node, key)
}

// GetBoxLength resolves defaults/inheritance and returns a checked size or edge.
// Percentages/relative lengths remain unsized. Use GetBorder or
// GetBorderWithFontContext for effective border widths (including none/hidden),
// rather than this raw longhand accessor.
func GetBoxLength(node *styledtree.StyNode, key string) (style.BoxLength, error) {
	p, err := elementProperty(node, key)
	if err != nil {
		return style.BoxLength{}, err
	}
	return style.ParseBoxLength(key, string(p))
}

// GetBoxKeyword resolves a supported box keyword longhand.
func GetBoxKeyword(node *styledtree.StyNode, key string) (string, error) {
	p, err := elementProperty(node, key)
	if err != nil {
		return "", err
	}
	return style.ParseBoxKeyword(key, string(p))
}

// GetFlowName resolves flow-from or flow-into. Empty means none. Names are decoded
// and case-sensitive; the two properties do not inherit unless explicitly requested.
func GetFlowName(node *styledtree.StyNode, key string) (string, error) {
	if key != "flow-from" && key != "flow-into" {
		return "", fmt.Errorf("not a flow property: %s", key)
	}
	p, err := elementProperty(node, key)
	if err != nil {
		return "", err
	}
	return style.ParseFlowName(string(p))
}

// GetColor resolves a foreground/background/border color to non-premultiplied
// sRGB. Inherited currentcolor remains symbolic until resolved on this element.
// Unsupported values and deferred substitutions return errors, never a sentinel.
func GetColor(node *styledtree.StyNode, key string) (color.NRGBA, error) {
	valid := key == "color" || key == "background-color"
	for _, side := range []string{"top", "right", "bottom", "left"} {
		valid = valid || key == "border-"+side+"-color"
	}
	if !valid {
		return color.NRGBA{}, fmt.Errorf("not a supported color property: %s", key)
	}
	p, err := elementProperty(node, key)
	if err != nil {
		return color.NRGBA{}, err
	}
	c, err := style.ParseColorValue(string(p))
	if err != nil {
		return color.NRGBA{}, err
	}
	if !c.Current {
		return c.Color, nil
	}
	if key == "color" {
		return color.NRGBA{}, fmt.Errorf("unresolved foreground currentcolor")
	}
	return GetColor(node, "color")
}

// Border is an effective physical edge; Width is in dimen.DU. The keyword
// widths thin/medium/thick follow the module's existing UA choices (.5/1/2px).
type Border struct {
	Width dimen.DU
	Style string
	Color color.NRGBA
}

// GetBorder resolves one physical side. none/hidden have zero effective width.
// Inherited widths use the parent's effective width, not its specified width.
// Nonzero relative widths need GetBorderWithFontContext and return an error here.
func GetBorder(node *styledtree.StyNode, side string) (Border, error) {
	return getBorder(node, side, nil)
}

func getBorder(node *styledtree.StyNode, side string, fonts *FontContext) (Border, error) {
	var b Border
	var err error
	b.Style, err = GetBoxKeyword(node, "border-"+side+"-style")
	if err != nil {
		return Border{}, err
	}
	b.Color, err = GetColor(node, "border-"+side+"-color")
	if err != nil {
		return Border{}, err
	}
	b.Width, err = borderWidth(node, side, fonts)
	if err != nil {
		return Border{}, err
	}
	return b, nil
}

func borderWidth(node *styledtree.StyNode, side string, fonts *FontContext) (dimen.DU, error) {
	lineStyle, err := GetBoxKeyword(node, "border-"+side+"-style")
	if err != nil {
		return 0, err
	}
	if lineStyle == "none" || lineStyle == "hidden" {
		return 0, nil
	}

	key := "border-" + side + "-width"
	local := GetLocalProperty(node.Styles(), key)
	if style.DeclarationKeyword(string(local)) == "inherit" {
		parent := styledtree.Node(node.Parent())
		if parent != nil && parent.HTMLNode().Type == html.ElementNode && !parent.Styles().IsDefaults() {
			inherited, e := borderWidth(parent, side, fonts)
			if e != nil {
				return 0, e
			}
			return inherited, nil
		}
	}
	width, err := GetBoxLength(node, key)
	if err != nil {
		return 0, err
	}
	if width.Kind != style.AbsoluteLength {
		if width.Kind == style.RelativeLength && width.Magnitude == 0 {
			return 0, nil
		}
		if fonts != nil {
			return fonts.borderWidth(node, key, width)
		}
		return 0, fmt.Errorf("border width requires absolute resolution: %s", key)
	}
	return width.Absolute, nil
}
