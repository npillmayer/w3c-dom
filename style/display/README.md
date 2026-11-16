# Normalized display values

`github.com/npillmayer/w3c-dom/style/display` implements the value grammar of
[CSS Display Level 3](https://www.w3.org/TR/css-display-3/#the-display-properties).
It is independent of DOM traversal, cascade resolution and layout support.
CSSOM validates display declarations with this package. Use `css.GetDisplay`
for normalized element values after cascade, or `display.Parse` for standalone
values. The obsolete `css.DisplayMode` bitmask and `css.ParseDisplay` API have
been removed; consumers must use Kind/Outer/Inner/InternalRole/IsListItem.
Tytui projects those values into css-box-layout’s typed display input.

```go
v, err := display.Parse("inline flow-root")
if err != nil {
    // errors.Is(err, display.ErrInvalidValue) identifies parse failures.
    return err
}
// v.Kind() == display.Box
// v.Outer() == display.Inline
// v.Inner() == display.FlowRoot
fmt.Println(v.CSS()) // inline-block
```

`Value` has private fields, is immutable and comparable, and has no mutators.
Equivalent forms compare equal with `==`; values can be map keys. `Parse` is the
only constructor. The zero value is invalid, `Valid()` is false, and `CSS()` is
empty. Every failed parse returns that zero value and an error wrapping
`ErrInvalidValue`; it never supplies a fallback display.

The variants are:

| Kind | Applicable accessors |
| --- | --- |
| `Box` | `Outer`, `Inner`, `IsListItem` |
| `None` | None; omit the subtree when implemented by layout |
| `Contents` | None; principal-box suppression, subject to layout rules |
| `Internal` | `InternalRole` (distinct table and ruby roles) |
| `Invalid` | None |

Inapplicable components return `NoOutside`, `NoInside`, or `NoInternalRole`;
`IsListItem` is false for special variants. Internal roles remain explicit rather
than being coerced to block/inline. These sentinel accessors do not describe the
used inner/outer semantics a table or ruby layout engine must implement.

The parser covers block/inline/run-in outer types; flow/flow-root/table/flex/grid/
ruby inner types; flow-based list items; none/contents; all Level 3 internal roles;
and inline-block/inline-table/inline-flex/inline-grid aliases. Missing inner type
means flow. Missing outer type means block except for ruby, which defaults to
inline. List items permit only flow and flow-root interiors.

`CSS()` emits canonical keyword ordering and backwards-compatible short forms.
For example, `block flow` becomes `block`, `inline flow-root` becomes
`inline-block`, and `block flow list-item` becomes `list-item`. `String()` is a
separate diagnostic representation, including for invalid values.

Input must be a single UTF-8 property value, not a declaration. The existing
Gorilla CSS scanner supplies token boundaries; identifiers are unescaped and
ASCII-case-folded before matching. Comments and CSS whitespace are accepted;
malformed tokens, including unterminated comments, are rejected. CSS input newline
and NUL preprocessing is applied. Comments separate tokens and never join parts
of a keyword (`bl/**/ock` is invalid).

`Parse` rejects CSS-wide keywords (`initial`, `inherit`, `unset`, `revert`, `revert-layer`),
variables/functions and `!important`; these require declaration/cascade handling. The private `block-inline` spelling is rejected by the new API.
Computed-value blockification/inlinification and layout support remain downstream.

Validation:

```sh
go test ./style/display
go test -race ./style/display
go test ./style/display -run '^$' -fuzz '^FuzzParse$' -fuzztime=20s -parallel=2
go test ./...
go vet ./...
```

## Declaration and cascade integration

`ParseDeclaration(input) (Value, WideKeyword, error)` additionally recognizes the
five CSS-wide keywords, with the same tokenization as `Parse`. A successful call
returns exactly one of an ordinary Value or a WideKeyword. It validates the value
portion only, after removal of `!important`; functions and variable substitution
remain outside this supported declaration grammar. Invalid declarations are
ignored by CSSOM. This is not custom-property/variable support.

CSSOM filters invalid display values before choosing a winner, including duplicate
declarations in inline styles and Douceur rules. Ordinary values retain their
spelling in `GetProperty`; CSS-wide tokens are stored in canonical spelling.
Third-party `Rule` implementations must preserve valid candidates when reducing
their own duplicates: CSSOM cannot recover declarations already discarded by a
provider. It validates every candidate exposed through the Rule interface.

`css.GetDisplay(*styledtree.StyNode) (display.Value, error)` accepts element nodes
and resolves defaults and `inherit`/`initial`/`unset` before normalization. It does
not treat a parsed grid/table/etc. value as a claim of layout support. Winning
`revert` and `revert-layer` values remain in the cascade; the accessor returns
`css.ErrUnsupportedDisplayCascade` rather than falling back. These keywords do
not implement origin/layer rollback. Nil/non-element nodes are errors. Invalid
manually supplied property values wrap `display.ErrInvalidValue`.

`GetProperty` remains available for raw inspection. CSSOM display
validation ensures that syntactically invalid display
declarations, including the private `block-inline`, no longer take precedence.
