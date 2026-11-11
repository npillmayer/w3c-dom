package douceuradapter

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var myhtml = `
<html><head>
<style>
  body { border-color: red; }
</style>
</head><body>
  <p>The quick brown fox jumps over the lazy dog.</p>
  <p id="world">Hello <b>World</b>!</p>
  <p style="padding-left: 5px;">This is a test.</p>
</body>
`

func TestExtract1(t *testing.T) {
	h, errhtml := html.Parse(strings.NewReader(myhtml))
	if errhtml != nil {
		t.Error(errhtml)
	}
	css := ExtractStyleElements(h)
	if len(css) != 1 {
		t.Error("Should extract 1 stylesheet")
	}
}

func TestExtractMissingContainers(t *testing.T) {
	for _, kind := range []atom.Atom{atom.Head, atom.Body} {
		t.Run(kind.String(), func(t *testing.T) {
			container := &html.Node{Type: html.ElementNode, DataAtom: kind, Data: kind.String()}
			container.AppendChild(&html.Node{Type: html.ElementNode, DataAtom: atom.Style, Data: "style"})
			valid := &html.Node{Type: html.ElementNode, DataAtom: atom.Style, Data: "style"}
			valid.AppendChild(&html.Node{Type: html.TextNode, Data: "p { color: red; }"})
			container.AppendChild(valid)
			sheets := ExtractStyleElements(container)
			if len(sheets) != 1 {
				t.Fatalf("got %d sheets, want 1", len(sheets))
			}
			if got := sheets[0].Rules()[0].Value("color"); got != "red" {
				t.Fatalf("color = %q", got)
			}
		})
	}
	for _, input := range []*html.Node{nil, {Type: html.DocumentNode}, {Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}} {
		if got := ExtractStyleElements(input); len(got) != 0 {
			t.Fatalf("got %d sheets without containers", len(got))
		}
	}
}

func TestExtractEmptyStyleFollowedByValidStyle(t *testing.T) {
	h, err := html.Parse(strings.NewReader(`<head><style></style><style>p {color: red}</style></head><body><style></style><style>p {color: blue}</style></body>`))
	if err != nil {
		t.Fatal(err)
	}
	sheets := ExtractStyleElements(h)
	if len(sheets) != 2 {
		t.Fatalf("got %d sheets, want 2", len(sheets))
	}
	for i, want := range []string{"red", "blue"} {
		if got := string(sheets[i].Rules()[0].Value("color")); got != want {
			t.Fatalf("sheet %d: color = %q", i, got)
		}
	}
}
