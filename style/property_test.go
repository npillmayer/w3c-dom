package style

import "testing"

func TestPropertyMapAddFreshAndZero(t *testing.T) {
	for _, pm := range []*PropertyMap{NewPropertyMap(), new(PropertyMap)} {
		pm.Add("color", "red")
		pm.Add("color", "blue")
		if got, ok := pm.Property("color"); !ok || got != "blue" {
			t.Fatalf("Property = %q, %v", got, ok)
		}
	}
	var pm *PropertyMap
	pm.Add("color", "red") // documented no-op
	if _, ok := pm.Property("color"); ok {
		t.Fatal("nil map acquired a value")
	}
}

func TestPropertyGroupCascadeMissingAndInherited(t *testing.T) {
	parent := NewPropertyGroup(PGColor)
	parent.Set("color", "red")
	child := NewPropertyGroup(PGColor)
	child.Parent = parent
	child.Set("color", "")
	if got := child.Cascade("color"); got != parent {
		t.Fatal("did not find ancestor value")
	}
	for _, group := range []*PropertyGroup{nil, child, NewPropertyGroup(PGColor)} {
		if got := group.Cascade("missing"); got != nil {
			t.Fatal("missing key found")
		}
	}
	fork, changed := child.ForkOnProperty("background-color", "blue", true)
	if !changed {
		t.Fatal("missing property did not create fork")
	}
	if got, ok := fork.Get("background-color"); !ok || got != "blue" {
		t.Fatalf("fork value = %q, %v", got, ok)
	}
	if child.IsSet("background-color") {
		t.Fatal("fork mutated original")
	}
}
