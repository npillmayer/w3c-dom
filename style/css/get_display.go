package css

import (
	"errors"
	"fmt"

	"github.com/npillmayer/w3c-dom/style/display"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

// ErrUnsupportedDisplayCascade means the winning display value needs cascade
// rollback which this engine does not implement. It is not an invalid declaration
// and must not silently select a lower-priority display value.
var ErrUnsupportedDisplayCascade = errors.New("unsupported display cascade")

// GetDisplay resolves an element's defaults and inherit/initial/unset semantics,
// then returns a normalized display value. It preserves valid values regardless
// of downstream layout support. Winning revert/revert-layer declarations return
// ErrUnsupportedDisplayCascade; invalid directly supplied values wrap
// display.ErrInvalidValue. Errors return the invalid zero display.Value.
//
// Nil, uninitialized, and non-element nodes are rejected. This is not a full
// browser computed-style operation: blockification/inlinification, variables,
// rollback and layout-dependent transformations are not implemented here.
// Use GetProperty for raw property inspection and display.Parse for standalone values.
func GetDisplay(node *styledtree.StyNode) (display.Value, error) {
	if node == nil || node.Payload == nil || node.HTMLNode() == nil || node.HTMLNode().Type != html.ElementNode {
		return display.Value{}, fmt.Errorf("GetDisplay requires an element node")
	}
	raw, err := GetProperty(node, "display")
	if err != nil {
		return display.Value{}, err
	}
	value, keyword, err := display.ParseDeclaration(string(raw))
	if err != nil {
		return display.Value{}, fmt.Errorf("display on <%s>: %w", node.HTMLNode().Data, err)
	}
	if keyword != display.NoKeyword {
		return display.Value{}, fmt.Errorf("%w: display:%s on <%s>", ErrUnsupportedDisplayCascade, keyword, node.HTMLNode().Data)
	}
	return value, nil
}
