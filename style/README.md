# HTML/CSS Styling

For checked, detached source font/text values after cascading, use
[`style/textstyle`](textstyle/README.md). It owns source inheritance and HTML
language/direction semantics without depending on a font or typography engine.

We strive to separate content from presentation. In typesetting, this is
probably an impossible claim, but we'll try anyway. Presentation
is governed with CSS (Cascading Style Sheets). CSS uses a box model more
complex than that of the TeX typesetting system and is well described here:

   https://developer.mozilla.org/en-US/docs/Learn/CSS/Introduction_to_CSS/Box_model

If you think about it: a typesetter using the HTML/CSS box model is
effectively a browser with output type PDF.
We therefore employ styling of HTML nodes like a web browser does.
A good explanation of styling may be found in

   https://hacks.mozilla.org/2017/08/inside-a-super-fast-css-engine-quantum-css-aka-stylo/

We will produce a "styled tree", which associates HTML nodes with CSS
styles. Given the HTML input

    <html><head></head><body>
      <p>The quick brown fox jumps over the lazy dog.</p>
      <p id="world">Hello <b>World</b>!</p>
      <p>This is a test.</p>
    </body>

and a style sheet like this:

    p {
    	margin-bottom: 10pt;
    }
    #world {
    	padding-top: 20pt;
    }

a styling engine wil produce a tree like this (simplified version – the
real data structures are more involved):

![styling](https://user-images.githubusercontent.com/4531688/52282401-a4ccdf80-2960-11e9-8ede-0ceee394b6ab.png)

CSSOM is the "CSS Object Model", similar to the DOM for HTML.

## Source-tree styling boundaries

`cssom.CSSOM.Style` styles the complete input tree. `StyleWithOptions` accepts
`StyleOptions{SkipSubtree: ...}` to exclude selected source subtrees from the
styled tree and inline-style collection. The callback must be read-only and
safe for concurrent calls; excluding the root is an error. The zero options
preserve ordinary Style traversal.

Neither entry point edits or reparents the HTML input. Selectors still see its
original ancestry, attributes and sibling positions, including excluded nodes.
This lets a content adapter retain source context while excluding inert template
contents or metadata from its styled material. CSS applies to all included
elements, including `bdi`, `small` and custom elements; supported layout-element
profiles belong to consumers, not the CSS cascade.

Callers extract and inspect head stylesheets explicitly before registration.
Traversal exclusion does not inspect stylesheets or define resource-loading
policy for them. Pagelayout's CONT-01 adapter supplies those policies.

## Effective borders and source fonts

`css.GetBorder(node, side)` resolves an absolute border in DOM `dimen.DU`.
`css.GetBorderWithFontContext(node, side, fonts)` additionally resolves `em` and
`rem`, using computed source fonts supplied in a `css.FontContext`:

```go
fonts := css.FontContext{
    RootFontSize: rootStyle.FontSize, // from textstyle.Resolve
    FontSize: func(node *styledtree.StyNode) (dimen.DU, error) {
        s, ok := sourceStyles[node] // previously computed textstyle.Style values
        if !ok {
            return 0, fmt.Errorf("missing source font snapshot")
        }
        return s.FontSize, nil
    },
}
border, err := css.GetBorderWithFontContext(node, "left", fonts)
// After checking err, convert at the adapter boundary when CSS pixels are needed:
pixels := float64(border.Width) / float64(dimen.PX)
```

The callback receives the element whose width is being computed. With explicit
`inherit`, that may be an ancestor: a parent's `.1em` border at 20px computes to
2px, and stays 2px when inherited by a child with a 40px font. `rem` uses the
source document element's computed size, including for that element's borders.
No receiving frame, containing geometry, font lookup or local CSS parser is
involved. Keep source ancestry stable while resolving, then retain the detached
`Border` value with the application's other snapshots.

Both getters retain effective zero widths for `none`/`hidden`, including when
that zero is inherited. Style/color/currentcolor and CSS-wide keyword behavior
are shared with the existing checked getters. The context is consulted only for
nonzero relative widths: `em` needs the element callback, `rem` needs the root
size, and absolute/zero/suppressed widths need neither. The old `GetBorder` API
still rejects nonzero relative widths.

Used font sizes must be positive and below `dimen.Fil`. Relative results round
to nearest DU (half upward); overflow into the reserved/infinite range is an
error. Nonzero metric/viewport units, border percentages and deferred expressions
remain unsupported. Failures return a zero Border, preserve callback causes,
and use `style.ErrInvalidValue` for invalid relative values/context/range.
As with other effective getters, strict applications must inspect authored CSS
before permissive cascade reduction discards invalid declarations.

## Caveats

Browsers are large and complex pieces of software, a fact that implies that
we should seek out where to reduce complexity.
There is not very much open source Go code around for supporting us
in implementing a styling engine, except the great work of
https://godoc.org/github.com/andybalholm/cascadia.
Therefore we will have to compromise
on many features in order to complete this in a realistic time frame.

This package relies on just one non-standard external library: cascadia.
CSS handling is de-coupled by introducing appropriate interfaces
StyleSheet and Rule. Concrete implementations may be found in sub-packages
of package style.

## Status

This is a first draft. It is unstable and the API will change without
notice. Please be patient.

## Read break-before values

Use `style.ParseBoxKeyword("break-before", value)` to check a concrete keyword,
and `css.GetBoxKeyword(node, "break-before")` to resolve its cascaded value.
The default is `auto`; the property does not inherit unless explicitly requested.
The checked vocabulary is `auto`, `avoid`, `avoid-page`, `page`, `left`, `right`,
`recto`, `verso`, `avoid-column`, `column`, `avoid-region`, and `region`.
Page-side values remain distinct. Applications can impose a narrower profile.

Declaration normalization handles CSS-wide keywords; effective resolution supports
`initial`, `inherit`, and `unset`. Unsupported rollback and unresolved substitutions
produce checked-accessor errors. Stylesheet inspection retains losing and
nonmatching declarations for application audits. This accessor reads values;
fragmentation, break propagation and page selection belong to the consumer.
