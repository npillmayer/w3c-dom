# Quality and accessor safety

Review date: 2026-11-10; source baseline: `4b7dda6`. “Reproduced” below means a
small public-API probe was run against this checkout, separately from its tests.
“Source review” means the implementation establishes the concern but a dedicated
runtime regression was not run. The A1–A5 fixes below were subsequently applied
and regression-tested on 2026-11-10. Empty-style extraction and missing head/body
handling were subsequently fixed as well; other findings remain open.

## Resolved accessor failures (A1–A5)

The table records the original failures at the baseline, all now resolved:

- A1: ancestor traversal checks termination safely and returns an error for an
  absent cascaded property. A nil node returns an error for either lookup path.
- A2: `PropertyGroup.Cascade` returns nil for an absent key or nil receiver.
  This is an intentional behavioral change from panicking; callers must check
  its result before calling group methods. `ForkOnProperty` handles missing keys.
- A3: the legacy styled-node accessor now examines ancestor groups and stops
  safely. The computed-style wrapper no longer invokes a second fallback on error;
  it returns `NullStyle`, as documented.
- A4: `PropertyMap.Add` initializes a fresh or zero-value map. The existing nil
  receiver no-op is preserved and documented.
- A5: empty attribute maps return zero length and nil items/named lookups.

Regression tests: `accessor_test.go`, `style/property_test.go`,
`style/css/cascade_test.go`, and `styledtree/node_test.go`. They cover default-only
style lookup, root and detached nodes, missing keys/groups, partial local groups,
ancestor and local values, explicit legacy inheritance, nil and fresh maps, and
empty/nonempty attributes. No API signatures changed.


| ID | Entry point and trigger | Observed result and cause |
| --- | --- | --- |
| A1 | `css.GetCascadedProperty`, reached through `ComputedStyles().GetPropertyValue("color")` on a document with default styles | Reproduced nil-pointer panic. The loop executes `node.Parent().Payload` even after finding a group on the synthetic root. A valid default does not protect the caller. |
| A2 | `PropertyGroup.Cascade` on a group without the requested key | Reproduced explicit panic. It cannot report absence; callers returning `(Property, error)` do not translate this panic to an error. `ForkOnProperty` also calls it before checking for nil. |
| A3 | `StyNode.GetPropertyValue("color", nil)` on the styled document | Reproduced nil-pointer panic. The ancestor loop never assigns its `group` variable and dereferences the parent at the root. This is also the fallback from the computed-style wrapper when the primary accessor returns an error. |
| A4 | `style.NewPropertyMap().Add("color", "red")` | Reproduced `assignment to entry in nil map`: the constructor leaves `m` nil and `Add` does not allocate it. A nil receiver silently does nothing, which is a different contract again. |
| A5 | `document.Attributes().Length()` | Reproduced nil-pointer panic. Documents return the nonnil `emptyNodeMap`, whose `forNode` is nil. `Length`, `Item`, and `GetNamedItem` guard the receiver but not that field. |

These are ordinary API failures, not evidence of memory corruption or Go `unsafe`
usage. The primary client risk is a panic or incorrect layout input.

### Minimal inherited-property reproduction

Save this as a temporary Go file outside the package and run `go run /path/to/file.go`
from the module directory with dependencies available:

```go
package main

import (
    "strings"

    dom "github.com/npillmayer/w3c-dom"
    "github.com/npillmayer/w3c-dom/style/cssom"
    "golang.org/x/net/html"
)

func main() {
    h, err := html.Parse(strings.NewReader("<p>text</p>"))
    if err != nil { panic(err) }
    engine := cssom.NewCSSOM(nil)
    root, err := engine.Style(h)
    if err != nil { panic(err) }
    document := dom.NodeFromStyledNode(root.Payload)
    _ = document.ComputedStyles().GetPropertyValue("color") // now returns "default"; panicked at baseline
}
```

### Safe local reads and limits

`(*PropertyMap).Size`, `Group`, and `Property` tolerate nil maps. So does
`css.GetLocalProperty`, through `Group`. These are useful for inspecting declarations,
but do not substitute for inherited/default resolution. `PropertyGroup.Get` handles
an unallocated dictionary, not a nil group receiver. `css.GetProperty(nil, key)`
now returns `NullStyle` and an error for every key.

### Inheritance and defaults resolved (2026-11-11)

`style/defaults.go` now contains one registry for group, initial value, and
inheritance. It drives group lookup, default map construction, and inheritance
classification. `word-wrap`, `overflow-wrap`, and `hyphens` have Text metadata;
region defaults are `none`; `position` and flow properties do not inherit implicitly.
Additional defaults are placed in the correct groups, including known keys.

The DOM and CSS accessors share `StyNode.GetPropertyValue`. It resolves local
values, explicit `inherit`, `initial`, and `unset`, and implicit inheritance through
parent elements. It resolves each parent's value rather than bypassing an `initial`
barrier through group links. Root `inherit` uses the initial value; document and
synthetic root nodes are bookkeeping, not CSS parent elements. Detached nodes
receive defaults. `GetCascadedProperty` remains a low-level raw group/ancestor
lookup; use `GetProperty` for semantic resolution.

Initial values ignore UA overrides and element display defaults. Absent properties
use configured UA overrides, then element defaults (notably display), then registry
initial values. Unknown extension keys use group X, do not implicitly inherit,
and have no built-in initial value; configured defaults and explicit `inherit`
remain usable. Nil-node CSS lookup still returns an error; missing unknown values
return `NullStyle`. Former missing-default tests now assert actual defaults.

`style/defaults_test.go` checks registry consistency and overrides;
`style/css/inheritance_test.go` checks parent/local precedence, initial barriers,
non-inherited properties, root/detached nodes, configured defaults, and DOM agreement.
Defaulting follows [CSS Cascade](https://www.w3.org/TR/css-cascade/#defaulting).
This is still a supported subset: renderer color sentinels and existing dimension
conventions remain, `word-wrap` is canonicalized to `overflow-wrap` in storage, defaults, and
declaration selection. Alias order and importance are covered for both inline and
stylesheet declarations, including inherited reads. `revert`, custom-property substitution, font
resolution, and layout-dependent computation are not implemented.

## Other confirmed behavior

| Priority | Area | Evidence |
| --- | --- | --- |
| Resolved | Empty embedded styles | Empty style elements without children are now skipped; nil input and missing head/body containers return no sheets for those containers. Adapter regressions verify later valid sheets and head/body order; DOM integration tests cover empty styles and missing containers. |
| Resolved | Element children | `Children()` now appends only element nodes, preserving tree order with the correct length and no typed nil padding. `children_test.go` covers mixed, empty, text-only, element-only, and comment-containing lists, item bounds, nil receivers, and unchanged `ChildNodes` counts. |
| Resolved | Position predicate | `IsAbsolute` tests the absolute variant; pattern matching now selects Static explicitly. |
| Resolved | Dimension variants | Scalar variants use equality instead of overlapping bit tests. Percentage checks use the relative-unit mask; unset, relative, percentage, and content kinds no longer collide. |
| Resolved | Percentages | Parsed percentages populate the percentage payload and match constructor values. Integer/string/dimension percentage helpers consistently clamp to [0, 100]; malformed input returns an error. This preserves the bounded Percent model, not unrestricted CSS percentages. |

Regression coverage in `style/css/variants_test.go` checks all scalar dimension and
position variants, pairwise kind matching, every relative unit, percentage payloads,
boundaries, and malformed input. Fractional dimensions and overflow detection were subsequently fixed; unrestricted
percentages remain outside the bounded Percent model.


## Further source-review findings (P2 unless stated)

- **Shorthands (confirmed expansion bugs resolved):** `margin` now expands to
  directional longhands (`margins` remains a compatibility alias). `border-radius`
  emits radius keys in top-left, top-right, bottom-right, bottom-left order, with
  optional slash-separated horizontal/vertical radii. Tests cover one through four
  values, asymmetric corners, invalid arity, and CSSOM integration. This remains
  token distribution rather than full CSS grammar validation: functions containing
  whitespace and invalid value semantics are not validated. Unsupported/invalid
  shorthands still fall back to raw properties in CSSOM.
- **Inline parsing (resolved 2026-11-11):** token-based splitting preserves quoted
  strings, escaped quotes, URLs, and nested delimiters. Duplicate keys select the
  last declaration at equal importance; important declarations beat normal ones.
  Importance is stripped from values and carried into cascade ranking, including
  shorthand expansion. Declaration order breaks ties within a rule. Regression
  tests cover parser output and cascade outcomes. Malformed declarations without
  a property/value are skipped; lexical errors stop parsing the remaining input.
  This is not full CSS value validation. Duplicate declarations in the douceur stylesheet adapter are now also resolved:
  the last declaration wins at equal importance, important beats normal, and
  winning declarations retain their original source positions for shorthand/longhand
  ordering. Adapter and cascade tests cover these cases, including competing rules.
- **Extension registration (resolved 2026-11-11):** `RegisterCompoundSplitter`
  now has a pointer receiver, retaining appended splitters on the CSSOM instance.
  The built-in splitter runs first, then custom splitters in registration order;
  the first successful expansion wins. Nil registrations are ignored. The styling
  regression covers multiple registrations, repeated styling, fallback after an
  unsupported key, built-in precedence, and unchanged unknown-property fallback.
  Register before styling; concurrent registration is not supported. This changes
  the Go method set: addressable CSSOM variables still work normally, but interfaces
  requiring this method must receive `*CSSOM`, and temporary constructor results
  must first be assigned to a variable.
- **Scopes (resolved 2026-11-11):** `FilterMatchesFor` collects global sheets,
  then ancestor scopes from outermost to innermost, including the target itself.
  Registration order is retained within a scope. This order breaks ties only
  after importance and specificity. Inline declarations remain local to their
  owning element. `scope_test.go` verifies scope-root/descendant application,
  sibling isolation, nested scopes, registration order, importance, specificity,
  and non-inherited inline values not leaking to descendants. This is the module's
  explicit subtree registration API, not implementation of CSS `@scope`.
- **Value preservation (resolved):** `PropertyGroup.Set` now preserves value case,
  matching `Add`. This intentionally removes automatic whole-value lowercasing;
  consumers requiring case-insensitive keyword matching must normalize keywords.
- **Dimensions (resolved 2026-11-11):** both dimension parsers accept signed
  decimal values, including leading-dot fractions. Shared exact rational scaling
  rejects values outside int32 before rounding; results round half away from zero.
  Font/viewport-relative magnitudes are available through `RelativeValue()`;
  existing stored DU magnitudes are rounded, preserving integer behavior. Relative
  and percentage input magnitudes must also fit int32. CSS percentages round and
  clamp to [0,100]; `dimen.ParsePercent` rounds percentages to integer DU values.
  `PercentFromString` remains an integer-only helper. Exponent notation and trailing
  decimal points are not accepted. Tests cover fractions, signs, unit case,
  relative units, percentage rounding, exact boundaries, and overflow after scaling.
- **Color conversion (resolved 2026-11-11):** `ParseColor` supports named colors
  (the Go x/image palette plus rebeccapurple), transparent, short/long hex with alpha,
  RGB/RGBA and HSL/HSLA comma or space/slash syntax, and HWB. Numeric components
  clamp to their ranges; output is 8-bit non-premultiplied sRGB. `ColorString`
  emits CSS hex with alpha, transparent, or the renderer sentinel default for nil.
  Invalid/contextual values return errors through ParseColor; Property.Color keeps
  the legacy black fallback. currentcolor, system colors, relative colors,
  color()/lab()/lch()/oklab()/oklch(), calc(), none components and CSS escapes/comments
  in color expressions remain unsupported. Tests cover alpha, standard green,
  formats, clamping, invalid input and serialization round trips.
- **Display:** display parsing still supports fewer modes than its flags represent.
- **Errors and adapters:** `FromHTMLParseTree` logs construction errors and returns nil;
  `FromHTMLParseTreeWithError` propagates extraction/styling errors. Recursive
  extraction now reports errors and continues collecting valid sheets. `CSSStyles.AppendRules`
  asserts that another `StyleSheet` has the same concrete adapter type.
- **P3 diagnostics:** DOT output uses raw property strings in HTML-like labels and
  panics on several writer failures. The root test creates DOT files in the source
  directory rather than a test temporary directory.

## Verification and existing strengths

After the accessor fixes, `go test ./...` and `go vet ./...` pass. The coverage
figures below are retained as the original inventory baseline, not a new measurement.

On Go 1.25.6, darwin/arm64, with the local replacement dependencies:

| Check | Result |
| --- | --- |
| `go test ./... -cover` | Pass |
| `go vet ./...` | Pass, no diagnostics |
| Public-API audit probes | Confirmed the failures listed above |

Reported statement coverage: root 39.3%, `dimen` 37.5%, `style/css` 24.3%,
`style/cssom` 75.0%, adapter 49.1%; `style`, `styledtree`, and `domdbg` report
0.0% under this command. `w3cdom` has no tests. This is ordinary per-package
coverage, not cross-package instrumentation; 0% does not mean no integration test
ever executes those packages.

The suite includes meaningful selector specificity, matching-branch, source-order,
inline-priority, and explicit-local-default regressions. It also includes smoke
tests that mostly log output. The original suite did not cover the critical default
lookup path; the new accessor regression tests now cover it. No race run, fuzzing, benchmark, standards conformance suite, or complete
downstream integration run was performed for this documentation review.

## Suggested repair sequence and acceptance criteria

1. **Accessor and extraction safety complete.** Focused regressions cover A1–A5,
   empty style elements, nil extraction input, and missing head/body containers.
2. **Property metadata and basic defaulting complete.** The shared registry and
   resolver cover inherit/initial/unset and configured defaults. Broader CSS property
   coverage, aliases, revert semantics, and computed-value processing remain open.
3. **Typed variant and element-list fixes complete.** Mixed child lists now contain
   only elements without typed nil entries. Exponent notation and unrestricted percentage
   representation remain open; preserve intentionally unsupported values explicitly.
4. **Repair declarations and extension points.** Test duplicate declarations,
   inline importance, quoted values, shorthand expansion, registration, subtree
   scopes, and the origin policy the module actually intends to support.
5. **Make builds and maintenance reproducible.** Resolve sibling dependency setup,
   clean up stale documentation and inactive code, and use `t.TempDir` for generated
   artifacts. Run race tests before revisiting the pipeline-overlap TODO.

For clients today, a successful local map read says only that a declaration exists.
The confirmed accessor panic paths are fixed, but do not assume arbitrary malformed
node structures are supported, returned strings are resolved,
or converting an unsupported value reports an error. Repair and regression tests
in this module are preferable to duplicating fallback logic in clients.

## Downstream font metadata follow-up (2026-11-11)

The Text registry now includes inherited font-family (`serif`), font-size (`medium`),
font-weight (`normal`), and font-style (`normal`), plus non-inherited unicode-bidi
(`normal`). CSS leaves the initial font family to the UA; generic serif is this
module's choice. Additional defaults override UA fallback values, while explicit
`initial` continues to use the registry initial value. Family lists preserve case
and quotes. See [CSS Fonts](https://www.w3.org/TR/css-fonts-4/) and
[Writing Modes](https://www.w3.org/TR/css-writing-modes-3/).

`style/css/font_metadata_test.go` exercises all five properties through styled-tree
construction and CSS/DOM lookup, including root inherit, detached defaults,
parent/child inheritance, initial barriers, unset, and configured defaults.
This resolves the missing metadata finding, not font selection, shorthand `font`
expansion, conversion of medium/relative sizes, bolder/lighter computation, or bidi
layout. Downstream code doing those jobs remains necessary.

## Downstream extraction follow-up (2026-11-11)

`ExtractStyleElementsWithError` scans the supplied root and descendants in tree
order, without head/body assumptions. Style content is the concatenation of all
text descendants, without artificial separators; comment nodes are excluded.
Empty/whitespace-only content is skipped. Each malformed sheet is omitted, with
an exported `StyleExtractionError` carrying its original node and parser error.
Joined errors report all failures; valid sheets before and after failures remain
available. Extraction does not mutate the input. Nil input is an empty result.

The legacy `ExtractStyleElements` wrapper discards errors but returns all valid
sheets. `FromHTMLParseTreeWithError` rejects partial extraction and propagates the
error; the existing DOM constructor logs and returns nil on failure. This makes
malformed embedded CSS a construction failure instead of silently dropping it.
Tests cover nested sheets, multiple failures, root-style fragments, split text,
unchanged input, nil/empty input, and DOM error propagation. Parser acceptance is
still governed by douceur; media/type filtering and external stylesheet fetching
are not added by this change.

## CSS unit consistency (2026-11-11)

PX and PT now follow CSS ratios throughout the module, with both dimension parsers
using one absolute-unit conversion helper. PT equals BP (not a TeX printer's point).
Exact rational mm/cm scaling avoids compounding rounded per-unit constants.
`style/css/units_test.go` checks constants, equivalence across inch/pixel/point/metric
units, fractional/case variants, compatibility units, and updated overflow limits
in both parsers. Fixed-point rounding and overflow policy are unchanged.
The downstream absolute-unit mismatch is resolved; relative-size resolution is not.
