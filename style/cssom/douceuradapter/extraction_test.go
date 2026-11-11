package douceuradapter

import (
	"errors"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestRecursiveExtractionAndErrors(t *testing.T) {
	root, err := html.Parse(strings.NewReader(`<style>p{color:red}</style><div><style>p{color:blue}</style><section><style>p{broken;}</style><style>p{color:green}</style><style>p{bad;}</style></section></div>`))
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := ExtractStyleElementsWithError(root)
	if err == nil {
		t.Fatal("missing parse errors")
	}
	var failure *StyleExtractionError
	if !errors.As(err, &failure) || failure.Node == nil || failure.Unwrap() == nil {
		t.Fatalf("missing error context: %v", err)
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok || len(joined.Unwrap()) != 2 {
		t.Fatalf("want two failures: %v", err)
	}
	if len(sheets) != 3 {
		t.Fatalf("got %d valid sheets", len(sheets))
	}
	for i, want := range []string{"red", "blue", "green"} {
		if got := string(sheets[i].Rules()[0].Value("color")); got != want {
			t.Errorf("sheet %d = %s", i, got)
		}
	}
	if got := ExtractStyleElements(root); len(got) != 3 {
		t.Fatal("legacy wrapper lost valid sheets")
	}
}
func TestCompleteStyleContent(t *testing.T) {
	// DataAtom is intentionally absent, as in manually constructed fragments.
	style := &html.Node{Type: html.ElementNode, Data: "style"}
	style.AppendChild(&html.Node{Type: html.TextNode, Data: "p { co"})
	style.AppendChild(&html.Node{Type: html.CommentNode, Data: "ignored"})
	nested := &html.Node{Type: html.ElementNode, Data: "span"}
	nested.AppendChild(&html.Node{Type: html.TextNode, Data: "lor: re"})
	style.AppendChild(nested)
	style.AppendChild(&html.Node{Type: html.TextNode, Data: "d; }"})
	first := style.FirstChild
	sheets, err := ExtractStyleElementsWithError(style)
	if err != nil || len(sheets) != 1 {
		t.Fatalf("got %d sheets, %v", len(sheets), err)
	}
	if got := sheets[0].Rules()[0].Value("color"); got != "red" {
		t.Errorf("color=%q", got)
	}
	if style.FirstChild != first || first.Data != "p { co" {
		t.Fatal("input mutated")
	}
	for _, root := range []*html.Node{nil, {Type: html.ElementNode, Data: "style"}, {Type: html.ElementNode, Data: "div"}} {
		if sheets, err := ExtractStyleElementsWithError(root); err != nil || len(sheets) != 0 {
			t.Fatalf("empty input: %v %v", sheets, err)
		}
	}
}
