/*
Package dimen implements dimensions and units.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>
*/
package dimen

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
)

// Online dimension conversion for print:
// http://www.unitconversion.org/unit_converter/typography-ex.html

// DU is a 'design unit' typ.
// Values are in scaled big points (different from TeX).
type DU int32

// Some pre-defined dimensions
const (
	Zero DU = 0
	SP   DU = 1       // scaled point = BP / 65536
	BP   DU = 65536   // big point (PDF) = 1/72 inch
	PX   DU = IN / 96 // CSS pixel
	PT   DU = BP      // CSS point = 1/72 inch
	MM   DU = 185771  // nearest DU to one millimeter
	CM   DU = 1857710 // nearest DU to one centimeter
	IN   DU = 4718592 // inch
)

// Infinity is the largest possible dimension
const Infinity = math.MaxInt32

// Some very stretchable dimensions
const Fil DU = Infinity - 3
const Fill DU = Infinity - 2
const Filll DU = Infinity - 1

// Some common paper sizes
var DINA4 = Point{210 * MM, 297 * MM}
var DINA5 = Point{148 * MM, 210 * MM}
var USLetter = Point{216 * MM, 279 * MM}
var USLegal = Point{216 * MM, 357 * MM}

// Stringer implementation.
func (d DU) String() string {
	return fmt.Sprintf("%dsp", int32(d))
}

// Points returns a dimension in big (PDF) points.
func (d DU) Points() float64 {
	return float64(d) / float64(BP)
}

// Point is a point on a page.
//
// TODO see methods in https://golang.org/pkg/image/#Point
type Point struct {
	X, Y DU
}

// Origin is origin
var Origin = Point{0, 0}

// Shift a point along a vector.
func (p *Point) Shift(vector Point) *Point {
	p.X += vector.X
	p.Y += vector.Y
	return p
}

// Rect is a rectangle (on a page).
type Rect struct {
	TopL, BotR Point
}

// Width returns the width of a rectangle, i.e. the difference between x-coordinates
// of bottom-right and top-left corner.
func (r Rect) Width() DU {
	return r.BotR.X - r.TopL.X
}

// Height returns the height of a rectangle, i.e. the difference between y-coordinates
// of bottom-right and top-left corner.
func (r Rect) Height() DU {
	return r.BotR.Y - r.TopL.Y
}

// ---------------------------------------------------------------------------

var dimenPattern = regexp.MustCompile(`^([+\-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+))(%|[a-zA-Z]{2})?$`)

// ParsePercent parses a string to return a dimension. Syntax is CSS Unit.
// If a percentage value is given (`80%`), the second return value will be true.
// Decimal results round half away from zero to DU (whole percent for percentages).
// Values outside int32 before rounding return an error.
func ParsePercent(s string) (DU, bool, error) {
	d := dimenPattern.FindStringSubmatch(s)
	if len(d) < 2 {
		return 0, false, errors.New("format error parsing dimension")
	}
	ispcnt := d[2] == "%"
	unit := d[2]
	if ispcnt {
		unit = "sp"
	}
	value, err := ParseAbsolute(d[1], unit)
	if err != nil {
		return 0, false, err
	}
	return value, ispcnt, nil
}

// ---------------------------------------------------------------------------

// Min returns the smaller of two dimensions.
func Min(a, b DU) DU {
	if a < b {
		return a
	}
	return b
}

// Max returns the greater of two dimensions.
func Max(a, b DU) DU {
	if a > b {
		return a
	}
	return b
}

// ParseScaled converts a decimal magnitude to design units, rounding half away
// from zero. Values outside the int32 range before rounding are rejected.
func ParseScaled(number string, scale DU) (DU, error) {
	return parseRatio(number, int64(scale), 1)
}

// ParseAbsolute converts a decimal CSS absolute length to DU. bp and sp are
// compatibility extensions; unitless input denotes sp. Metric units use exact
// inch ratios before the final rounding, unlike multiplying rounded MM/CM constants.
func ParseAbsolute(number, unit string) (DU, error) {
	switch strings.ToLower(unit) {
	case "px":
		return parseRatio(number, int64(IN), 96)
	case "pt", "bp":
		return parseRatio(number, int64(IN), 72)
	case "in":
		return parseRatio(number, int64(IN), 1)
	case "mm":
		return parseRatio(number, int64(IN)*5, 127)
	case "cm":
		return parseRatio(number, int64(IN)*50, 127)
	case "", "sp":
		return parseRatio(number, 1, 1)
	default:
		return 0, fmt.Errorf("unsupported absolute unit %q", unit)
	}
}

func parseRatio(number string, numerator, denominator int64) (DU, error) {
	if !regexp.MustCompile(`^[+\-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)$`).MatchString(number) {
		return 0, fmt.Errorf("invalid dimension magnitude %q", number)
	}
	value, ok := new(big.Rat).SetString(number)
	if !ok {
		return 0, fmt.Errorf("invalid dimension magnitude %q", number)
	}
	value.Mul(value, big.NewRat(numerator, denominator))
	if value.Cmp(new(big.Rat).SetInt64(math.MinInt32)) < 0 || value.Cmp(new(big.Rat).SetInt64(math.MaxInt32)) > 0 {
		return 0, fmt.Errorf("dimension overflow: %s", number)
	}
	sign := value.Sign()
	value.Abs(value)
	value.Add(value, big.NewRat(1, 2))
	rounded := new(big.Int).Quo(value.Num(), value.Denom()).Int64()
	return DU(int64(sign) * rounded), nil
}
