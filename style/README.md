# HTML/CSS Styling

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
