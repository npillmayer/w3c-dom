# Finding from downstream modules:

- **Font metadata resolved (2026-11-11):** font-family, font-size, font-weight, font-style, and unicode-bidi now have registry entries in Text. The four font properties inherit; unicode-bidi does not. Initial values are serif (this module’s UA choice), medium, normal, normal, and normal respectively. Regression tests cover defaults, overrides, inheritance and explicit keywords. Downstream metadata fallbacks can now be reassessed; font selection, relative sizing, weight computation and bidi processing still require downstream handling.
- **Style extraction resolved (2026-11-11):** ExtractStyleElementsWithError recursively scans any subtree, concatenates complete descendant text for each style element, and returns successful sheets plus joined node-specific parse errors. FromHTMLParseTreeWithError propagates failures. Downstream recursive/error-reporting code can be reassessed against these APIs; the legacy extraction wrapper intentionally discards errors.
- **CSS absolute units resolved (2026-11-11):** shared PX and PT constants and both parsers now use 96 px/in and 72 pt/in. DU remains 1/65536 of a CSS point. Metric parsing uses exact inch ratios before rounding. This removes the absolute-unit mismatch; downstream parent/root-relative computation remains necessary.
- HTML defaults remain limited. Several supported inline tags still default to block upstream. Our supported-element checks cannot simply be removed.



# Reconciliation of the root README TODOs

Checked 2026-11-10 against `4b7dda6`. Most comments still exist, but presence of a
TODO is not evidence of priority or an adequate description of the remaining work.
Use symbols and files below rather than historical line numbers.

| Original item | Current status | Suggested disposition |
| --- | --- | --- |
| `dom.go:302` pseudo-elements | Still open: `computedStyles` has no pseudo-element selector argument or model | Feature backlog; after safe property resolution |
| `dimen/dimen.go:66` image.Point methods | Still open; `Point` has `Shift`, but no broader parity goal is defined | Low priority; specify actual client geometry needs |
| `style/defaults.go:66` user-agent defaults | Resolved for supported properties: shared metadata, element defaults, and correctly grouped overrides | Broader CSS property coverage remains open |
| `style/defaults.go:182` transparent background | Still uses the renderer-specific `default` sentinel; `Property.Color` now supports transparent | Decide the renderer contract and implement/test transparency end to end |
| `style/defaults.go:221` position | Historical commented-out `Background` type, removed during registry consolidation | Archive as design history; do not confuse it with active position helper defects |
| `style/cssom/cssom.go:495` pipeline overlap | Still open in `CSSOM.Style`; construction and styling remain separate synchronized passes | Defer until correctness and parent/group ordering invariants are tested |
| `style/cssom/cssom.go:664` node path | `nodePath` still provides only a short label | Low priority diagnostic enhancement |
| `style/convert.go:5` palette | Resolved: standard palette and common CSS color formats, with ParseColor errors | Advanced/context-dependent color formats remain open |
| `style/css/option.go.refact:174` options | In inactive refactoring source, excluded from the build | Archive or remove once useful design material has been identified |
| `style/css/option.go.refact:486` bidi variants | Also inactive; not an implemented bidirectional-text feature | Move to an explicit future scope decision if clients need it |
| `style/css/display.go:126` display | `ParseDisplay` is implemented for a limited subset; its bare TODO overstates the absence of implementation | Replace with a supported-value list and specific missing modes |

The root README does not capture the highest-impact current work: unsafe ancestor
lookup, fresh map initialization, empty attribute maps, empty style extraction,
dimension/position misclassification, or declaration/scoping defects. See the
[quality review](quality.md) for evidence and acceptance criteria. Its A1–A5 accessor
failures (ancestor lookup, group absence, fresh maps, and empty attribute maps) are
now resolved with focused regression tests. Empty-style extraction and missing
head/body handling, dimension/position variant matching, and percentage parsing
are also resolved. The mixed-child `Children()` length/typed-nil bug is fixed
and regression-tested. Margin and border-radius expansion are also corrected;
the other findings remain open.

Other documentation drift: `AGENTS.md` says there is no module manifest and only
an initial commit; both are outdated. Package comments mention old tree/builder
interfaces and a `creator` argument no longer accepted by `CSSOM.Style`.
`ChildNodes` promises a live list but constructs a snapshot. Dependency descriptions
omit libraries now in `go.mod`. Update these comments with the corresponding API
repairs so the intended contracts and implementation stay aligned.

Follow-up 2026-11-11: inline quoted-value parsing, duplicate declaration selection,
`!important`, and value-case preservation are resolved. Stylesheet-adapter
duplicate handling is also resolved: importance and winner source order are
preserved, including shorthand/longhand conflicts, with regression coverage.

`RegisterCompoundSplitter` is also resolved: its pointer receiver retains custom
splitters for subsequent styling. See the quality review for method-set compatibility.

Subtree scope propagation is resolved: registered sheets reach descendants while
inline declarations remain local. Nested scope ordering is documented and tested.

Property metadata and inherit/initial/unset resolution are now consolidated and
regression-tested (2026-11-11); see the quality review for supported-subset limits.

The `word-wrap` alias now shares canonical `overflow-wrap` storage and metadata.
Inline and stylesheet declarations compete as one property using declaration order
and importance; both lookup names work. Property enumeration uses `overflow-wrap`.
Focused alias regressions were added on 2026-11-11.

Dimension parsing now supports signed decimal values and checked scaling with
half-away-from-zero rounding. RelativeValue exposes fractional font/viewport
magnitudes. Overflow and fractional regression tests were added (2026-11-11).
