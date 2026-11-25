package css_test

import (
	"github.com/npillmayer/w3c-dom/style"
	"github.com/npillmayer/w3c-dom/style/css"
	"testing"
)

func TestBreakBeforeValues(t *testing.T) {
	if style.InitialProperty("break-before") != "auto" || style.IsCascading("break-before") || style.GroupNameFromPropertyKey("break-before") != style.PGDisplay {
		t.Fatal("incorrect break-before metadata")
	}
	for _, value := range []string{"auto", "page", "left", "right", "recto", "verso", "avoid", "avoid-page", "column", "avoid-column", "region", "avoid-region"} {
		parsed, err := style.ParseBoxKeyword("break-before", value)
		if err != nil || parsed != value {
			t.Fatalf("parse %s: %s %v", value, parsed, err)
		}
		nodes := displayNodes(t, `<p id="x" style="break-before:`+value+`">text</p>`, nil)
		got, err := css.GetBoxKeyword(nodes["x"], "break-before")
		if err != nil || got != value {
			t.Fatalf("resolve %s: %s %v", value, got, err)
		}
	}
	for _, value := range []string{"", "page verso", "always", "10px", "default", "var(--break)", "inherit"} {
		if _, err := style.ParseBoxKeyword("break-before", value); err == nil {
			t.Fatalf("accepted unresolved/invalid value %q", value)
		}
	}
	for _, value := range []string{"revert", "revert-layer", "var(--break)"} {
		nodes := displayNodes(t, `<p id="x" style="break-before:`+value+`">text</p>`, nil)
		if _, err := css.GetBoxKeyword(nodes["x"], "break-before"); err == nil {
			t.Fatalf("unchecked %s", value)
		}
	}
	if _, err := css.GetBoxKeyword(nil, "break-before"); err == nil {
		t.Fatal("nil accepted")
	}
}

func TestBreakBeforeCascadeAndInheritance(t *testing.T) {
	nodes := displayNodes(t, `<html id="root" style="break-before:inherit"><body><div id="parent" style="break-before:recto"><p id="plain">x</p><p id="inherit" style="break-before:inherit">x</p><p id="initial" style="break-before:initial">x</p><p id="unset" style="break-before:unset">x</p></div><p id="inline" style="break-before:left">x</p><p id="important" style="break-before:left">x</p><p id="ordered">x</p><p id="escaped" style="break-before:/**/\72 ECTO/**/">x</p></body></html>`, nil,
		displaySheet(t, `#inline {break-before:page} #important {break-before:right!important} #ordered {break-before:recto}`),
		displaySheet(t, `#ordered {break-before:verso}`))
	for id, want := range map[string]string{"root": "auto", "parent": "recto", "plain": "auto", "inherit": "recto", "initial": "auto", "unset": "auto", "inline": "left", "important": "right", "ordered": "verso", "escaped": "recto"} {
		got, err := css.GetBoxKeyword(nodes[id], "break-before")
		if err != nil || got != want {
			t.Errorf("%s: %s %v, want %s", id, got, err, want)
		}
	}
}
