package dom_test

import (
	"os"
	"strings"
	"testing"

	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tree"
	dom "github.com/npillmayer/w3c-dom"
	"github.com/npillmayer/w3c-dom/domdbg"
	"github.com/npillmayer/w3c-dom/style/cssom"
	"github.com/npillmayer/w3c-dom/style/cssom/douceuradapter"
	"github.com/npillmayer/w3c-dom/styledtree"
	"golang.org/x/net/html"
)

var graphviz = false

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

var mycss = `
p { margin-bottom: 10pt; }
#world { padding-top: 20pt; }
`

func buildDOM(t *testing.T) *dom.W3CNode {
	h, err := html.Parse(strings.NewReader(myhtml))
	if err != nil {
		t.Errorf("Cannot create test document")
	}
	return dom.FromHTMLParseTree(h, nil)
}

func TestW3CDoc(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.engine")
	defer teardown()
	//
	root := buildDOM(t)
	//
	t.Logf("root is RootNode: %v", root.IsRoot())
	t.Logf("root = %+v", root)
	t.Logf("root is Doc-Node: %v", root.IsDocument())
	if !root.IsDocument() {
		t.Logf("root node is %s", root.NodeName())
		t.Errorf("node should be document root, but isn't")
	}
}

func TestW3CDom1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.engine")
	defer teardown()
	//
	root := buildDOM(t)
	if graphviz {
		gvz, _ := os.CreateTemp(".", "w3c-*.dot")
		defer gvz.Close()
		domdbg.ToGraphViz(root, gvz, nil)
	}
	if root.NodeName() != "#document" {
		t.Errorf("name of root element expected to be '#document")
	}
	t.Logf("root node is %s", root.NodeName())
}

func TestW3CTextContent1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.engine")
	defer teardown()
	//
	root := buildDOM(t)
	isLeaf := tree.NodeIsLeaf[*styledtree.StyNode]()
	calcRank := tree.CalcRank[*styledtree.StyNode]
	root.Walk().DescendentsWith(isLeaf).BottomUp(calcRank).Promise()()
	n, _ := dom.TreeNodeFromNode(root)
	t.Logf("DOM has size=%d", n.Rank)
	text, err := root.TextContent()
	t.Logf("text content = '%s\n'", text)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestW3CStyles1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.engine")
	defer teardown()
	//
	root := buildDOM(t)
	body := root.FirstChild().FirstChild().NextSibling().(*dom.W3CNode)
	props := body.ComputedStyles()
	color := props.GetPropertyValue("border-top-color")
	if color != "red" {
		t.Errorf("border-color = %v", color)
	}
}

func prepareStyledTree(t *testing.T) *tree.Node[*styledtree.StyNode] {
	h, errhtml := html.Parse(strings.NewReader(myhtml))
	styles := douceuradapter.ExtractStyleElements(h)
	t.Logf("Extracted %d <style> elements", len(styles))
	c, errcss := parser.Parse(mycss)
	if errhtml != nil || errcss != nil {
		t.Fatalf("Cannot create test document")
	}
	s := cssom.NewCSSOM(nil)
	for _, sty := range styles {
		s.AddStylesForScope(nil, sty, cssom.Script)
	}
	s.AddStylesForScope(nil, douceuradapter.Wrap(c), cssom.Author)
	doc, err := s.Style(h)
	if err != nil {
		t.Errorf("Cannot style test document: %s", err.Error())
	}
	return doc
}

func TestDom1(t *testing.T) {
	sn := prepareStyledTree(t)
	PrintTree(sn, t, domFmt)
}

func TestDom2(t *testing.T) {
	sn := prepareStyledTree(t)
	doc, err := dom.NodeFromTreeNode(sn)
	if err != nil {
		t.Fatalf("Cannot create DOM node: %s", err.Error())
	}
	gvz, _ := os.CreateTemp(".", "graphviz-*.dot")
	defer gvz.Close()
	domdbg.ToGraphViz(doc, gvz, nil)
}

// --- Helpers ----------------------------------------------------------

func domFmt(dn *dom.W3CNode) string {
	if dn == nil {
		return "<nil>"
	}
	return dn.NodeName()
}

func PrintTree(n *tree.Node[*styledtree.StyNode], t *testing.T, fmtnode func(*dom.W3CNode) string) {
	indent := 0
	dn, err := dom.NodeFromTreeNode(n)
	if err != nil {
		t.Fatalf("Cannot create DOM node: %s", err.Error())
	}
	printNode(dn, indent, t, fmtnode)
}

func printNode(dn *dom.W3CNode, w int, t *testing.T, fmtnode func(*dom.W3CNode) string) {
	if dn.NodeName() == "#text" {
		t.Logf("%s%s", indent(w), fmtnode(dn))
	} else {
		t.Logf("%s%s = {", indent(w), fmtnode(dn))
		for ch := dn.FirstChild(); ch != nil; ch = ch.NextSibling() {
			printNode(ch.(*dom.W3CNode), w+1, t, fmtnode)
		}
		t.Logf("%s}", indent(w))
	}
}

func indent(w int) string {
	return strings.Repeat("   ", w)
}
