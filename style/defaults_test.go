package style

import "testing"

func TestPropertyMetadataConsistency(t *testing.T) {
	defaults := InitializeDefaultPropertyValues(nil)
	for key, meta := range propertyRegistry {
		if meta.initial.IsEmpty() {
			t.Errorf("empty initial: %s", key)
		}
		if GroupNameFromPropertyKey(key) != meta.group || IsCascading(key) != meta.inherited {
			t.Errorf("metadata mismatch: %s", key)
		}
		if got, ok := defaults.Property(key); !ok || got != InitialProperty(key) {
			t.Errorf("default %s = %q, %v", key, got, ok)
		}
	}
	overrides := InitializeDefaultPropertyValues([]KeyValue{{Key: "color", Value: "purple"}, {Key: "width", Value: "42px"}, {Key: "custom", Value: "value"}})
	for key, want := range map[string]Property{"color": "purple", "width": "42px", "custom": "value"} {
		if got, ok := overrides.Property(key); !ok || got != want {
			t.Errorf("override %s = %q", key, got)
		}
	}
	if IsCascading("position") || IsCascading("flow-into") || IsCascading("flow-from") {
		t.Error("non-inherited property inherits")
	}
	for _, key := range []string{"word-wrap", "overflow-wrap", "hyphens"} {
		if !IsCascading(key) || GroupNameFromPropertyKey(key) != PGText {
			t.Errorf("text metadata: %s", key)
		}
	}
}

func TestWrapAliasStorageAndDefaults(t *testing.T) {
	pm := NewPropertyMap()
	pm.Add("word-wrap", "normal")
	pm.Add("overflow-wrap", "break-word")
	for _, key := range []string{"word-wrap", "overflow-wrap"} {
		if got, _ := pm.Property(key); got != "break-word" {
			t.Errorf("%s=%q", key, got)
		}
	}
	if got := pm.Group(PGText).Properties(); len(got) != 1 || got[0].Key != "overflow-wrap" {
		t.Errorf("stored keys: %v", got)
	}
	defaults := InitializeDefaultPropertyValues([]KeyValue{{Key: "word-wrap", Value: "break-word"}})
	for _, key := range []string{"word-wrap", "overflow-wrap"} {
		if defaults.DefaultProperty(nil, key) != "break-word" || InitialProperty(key) != "normal" {
			t.Errorf("alias defaults: %s", key)
		}
	}
}
