package style

import "github.com/npillmayer/w3c-dom/internal/csssyntax"

// These lists describe the CSS Backgrounds and Borders Level 3 reset sets.
// Border-image properties are reset-only subproperties of border. Radius and
// background-blend-mode are deliberately not part of these shorthands.
var borderImageLonghands = []string{
	"border-image-source", "border-image-slice", "border-image-width",
	"border-image-outset", "border-image-repeat",
}

var backgroundLonghands = []string{
	"background-color", "background-image", "background-position", "background-size",
	"background-repeat", "background-attachment", "background-origin", "background-clip",
}

func paintShorthand(key string) bool {
	switch key {
	case "border", "border-top", "border-right", "border-bottom", "border-left", "background":
		return true
	}
	return false
}

// expandPaintShorthand is also the validator used before duplicate reduction.
// No partial expansion escapes on invalid or unsupported input. Substitutions
// require a pending-substitution model and are not approximated by resets.
func expandPaintShorthand(key string, value Property) ([]KeyValue, error) {
	var keys []string
	if key == "background" {
		keys = backgroundLonghands
	} else {
		for _, side := range fourDirs {
			if key != "border" && key != "border-"+side {
				continue
			}
			for _, component := range []string{"width", "style", "color"} {
				keys = append(keys, "border-"+side+"-"+component)
			}
		}
		if key == "border" {
			keys = append(keys, borderImageLonghands...)
		}
	}
	if len(keys) == 0 {
		return nil, invalidValue(key, string(value))
	}
	values := make(map[string]Property)
	wide := DeclarationKeyword(string(value))
	if wide == "" {
		parts, err := csssyntax.Components(string(value))
		if err != nil || len(parts) == 0 {
			return nil, invalidValue(key, string(value))
		}
		if key == "background" {
			seenNone := false
			for _, part := range parts {
				word, _ := csssyntax.Ident(part)
				if csssyntax.Lower(word) == "none" && !seenNone {
					seenNone = true
					continue
				}
				if _, err := ParseColorValue(part); err != nil || values["background-color"] != "" {
					return nil, invalidValue(key, string(value))
				}
				values["background-color"] = Property(part)
			}
		} else {
			for _, part := range parts {
				component := ""
				if _, err := ParseBoxLength("border-top-width", part); err == nil {
					component = "width"
				} else if _, err := ParseBoxKeyword("border-top-style", part); err == nil {
					component = "style"
				} else if _, err := ParseColorValue(part); err == nil {
					component = "color"
				}
				if component == "" || values[component] != "" {
					return nil, invalidValue(key, string(value))
				}
				values[component] = Property(part)
			}
		}
	}
	result := make([]KeyValue, 0, len(keys))
	for _, longhand := range keys {
		v := InitialProperty(longhand)
		if wide != "" {
			v = Property(wide) // Includes reset-only properties for CSS-wide values.
		} else if key == "background" {
			if specified := values[longhand]; specified != "" {
				v = specified
			}
		} else {
			for _, component := range []string{"width", "style", "color"} {
				if borderSideProperty(longhand, component) && values[component] != "" {
					v = values[component]
				}
			}
		}
		result = append(result, KeyValue{Key: longhand, Value: v})
	}
	return result, nil
}
