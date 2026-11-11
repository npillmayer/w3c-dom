package style

import (
	"fmt"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

// Color converts supported CSS colors. The renderer sentinel "default" returns
// nil. Invalid or context-dependent values retain the legacy black fallback;
// use ParseColor when errors must be distinguished from black.
func (p Property) Color() color.Color {
	if strings.EqualFold(strings.TrimSpace(string(p)), "default") {
		return nil
	}
	c, err := ParseColor(string(p))
	if err != nil {
		return color.Black
	}
	return c
}

var colorNumber = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

func colorFloat(s string) (float64, error) {
	if !colorNumber.MatchString(s) {
		return 0, fmt.Errorf("invalid color number %q", s)
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid color number %q", s)
	}
	return n, nil
}
func clampColor(v float64) float64 { return math.Max(0, math.Min(1, v)) }
func colorByte(v float64) uint8    { return uint8(math.Round(255 * clampColor(v))) }
func component(s string, scale float64) (float64, error) {
	if strings.HasSuffix(s, "%") {
		scale = 100
		s = strings.TrimSuffix(s, "%")
	}
	n, err := colorFloat(s)
	return clampColor(n / scale), err
}

// ParseColor parses named colors, transparent, 3/4/6/8-digit hex, rgb/rgba,
// hsl/hsla and hwb. RGB/HSL accept legacy comma or modern space/slash syntax.
// Results are non-premultiplied sRGB. Contextual and advanced color spaces return errors.
func ParseColor(s string) (color.NRGBA, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	bad := func() (color.NRGBA, error) {
		return color.NRGBA{}, fmt.Errorf("unsupported or invalid CSS color %q", s)
	}
	if s == "transparent" {
		return color.NRGBA{}, nil
	}
	if s == "rebeccapurple" {
		return color.NRGBA{102, 51, 153, 255}, nil
	}
	if c, ok := colornames.Map[s]; ok {
		return color.NRGBA{c.R, c.G, c.B, 255}, nil
	}
	if strings.HasPrefix(s, "#") {
		h := s[1:]
		if len(h) == 3 || len(h) == 4 {
			var b strings.Builder
			for _, c := range h {
				b.WriteRune(c)
				b.WriteRune(c)
			}
			h = b.String()
		}
		if len(h) != 6 && len(h) != 8 {
			return bad()
		}
		for _, digit := range h {
			if !strings.ContainsRune("0123456789abcdef", digit) {
				return bad()
			}
		}
		n, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return bad()
		}
		if len(h) == 6 {
			n = n<<8 | 255
		}
		return color.NRGBA{uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n)}, nil
	}
	i := strings.IndexByte(s, '(')
	if i < 0 || !strings.HasSuffix(s, ")") {
		return bad()
	}
	name, body := s[:i], s[i+1:len(s)-1]
	if name != "rgb" && name != "rgba" && name != "hsl" && name != "hsla" && name != "hwb" {
		return bad()
	}
	var fields []string
	alpha := "1"
	legacy := strings.Contains(body, ",")
	if legacy {
		if name == "hwb" || strings.Contains(body, "/") {
			return bad()
		}
		fields = strings.Split(body, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		if len(fields) == 4 {
			alpha = fields[3]
			fields = fields[:3]
		}
	} else {
		parts := strings.Split(body, "/")
		if len(parts) > 2 {
			return bad()
		}
		fields = strings.Fields(parts[0])
		if len(parts) == 2 {
			alpha = strings.TrimSpace(parts[1])
		}
	}
	if len(fields) != 3 {
		return bad()
	}
	a, err := component(alpha, 1)
	if err != nil {
		return bad()
	}
	var rgb [3]float64
	if name == "rgb" || name == "rgba" {
		for i, f := range fields {
			if legacy && strings.HasSuffix(f, "%") != strings.HasSuffix(fields[0], "%") {
				return bad()
			}
			rgb[i], err = component(f, 255)
			if err != nil {
				return bad()
			}
		}
	} else {
		hue := fields[0]
		factor := 1.0
		for _, u := range []struct {
			suffix string
			factor float64
		}{{"grad", .9}, {"turn", 360}, {"rad", 180 / math.Pi}, {"deg", 1}} {
			if strings.HasSuffix(hue, u.suffix) {
				hue = strings.TrimSuffix(hue, u.suffix)
				factor = u.factor
				break
			}
		}
		h, e := colorFloat(hue)
		if e != nil {
			return bad()
		}
		h = math.Mod(h, 360/factor) * factor / 60
		if h < 0 {
			h += 6
		}
		if !strings.HasSuffix(fields[1], "%") || !strings.HasSuffix(fields[2], "%") {
			return bad()
		}
		v, e := component(fields[1], 100)
		if e != nil {
			return bad()
		}
		w, e := component(fields[2], 100)
		if e != nil {
			return bad()
		}
		c := 1.0
		m := 0.0
		if name != "hwb" {
			c = (1 - math.Abs(2*w-1)) * v
			m = w - c/2
		}
		x := c * (1 - math.Abs(math.Mod(h, 2)-1))
		switch {
		case h < 1:
			rgb = [3]float64{c, x, 0}
		case h < 2:
			rgb = [3]float64{x, c, 0}
		case h < 3:
			rgb = [3]float64{0, c, x}
		case h < 4:
			rgb = [3]float64{0, x, c}
		case h < 5:
			rgb = [3]float64{x, 0, c}
		default:
			rgb = [3]float64{c, 0, x}
		}
		for i := range rgb {
			if name == "hwb" {
				if v+w >= 1 {
					rgb[i] = v / (v + w)
				} else {
					rgb[i] = rgb[i]*(1-v-w) + v
				}
			} else {
				rgb[i] += m
			}
		}
	}
	return color.NRGBA{colorByte(rgb[0]), colorByte(rgb[1]), colorByte(rgb[2]), colorByte(a)}, nil
}

// ColorString returns canonical CSS hex, including alpha when nonopaque.
// Nil retains the renderer sentinel. Conversion unpremultiplies color.Color values.
func ColorString(c color.Color) string {
	if c == nil {
		return "default"
	}
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	if n.A == 0 {
		return "transparent"
	}
	if n.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", n.R, n.G, n.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", n.R, n.G, n.B, n.A)
}
