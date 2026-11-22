# Source text-style snapshots

`textstyle.Resolve(nodes...)` computes detached `Style` values in input order.
Each call resolves original ancestors once, without visiting unrelated subtrees,
and returns no partial slice on error. Select the nodes your application needs;
the service does not decide visibility, paragraph eligibility or routing. Keep
the original HTML and styled tree unchanged during resolution, then discard them
when the application's other source work is complete. Snapshots contain only
values, with no DOM pointers, font objects or long-lived caches.

`ResolveWithContext(node, Context{Parent: parent, RootFontSize: rootSize})` supports
callers that already have computed source context. Use `Initial()` for the
document's parent and its initial root-size basis. `ResolveFontSize` exposes the
same checked size conversion without DOM traversal. Font sizes and line-height
lengths use this module's `dimen.DU`; convert explicitly at library boundaries.
One CSS pixel is 49152 DU; 16 CSS pixels are 12 big points, or 786432 DU.

The initial text profile uses serif, 16px, weight 400, normal slant, black,
language `und`, LTR, normal whitespace/bidi/line-height and manual hyphenation.
It preserves the original tytui adapter's fixed print defaults; arbitrary CSSOM
UA overrides are outside this service's profile. Supported values are:

| Property/context | Resolution |
| --- | --- |
| Font family | Checked comma-separated identifier sequences or quoted names; retained as a CSS family list, without font selection. |
| Font size | Positive absolute CSS lengths (`px`, `pt`, `pc`, `in`, `cm`, `mm`, `q`), `em`, `rem`, `%`, `xx-small` through `xx-large`, `larger`, `smaller`. Relative units use computed source fonts; the root's font-size uses the initial size for `rem`. |
| Font weight | Integral values 1–1000, normal/bold, bolder/lighter relative to the source parent. |
| Font style | Normal, italic, oblique without an angle. |
| Foreground | The DOM's checked sRGB color subset. `currentcolor` on `color` resolves to the parent's computed foreground. Both checked spelling and NRGBA are retained. |
| Whitespace | Normal, nowrap, pre, pre-wrap, pre-line, break-spaces. No source bytes are transformed. |
| Line height | Normal, nonnegative unitless factor, or computed nonnegative length/percentage. Factors inherit as factors; relative lengths/percentages inherit as lengths computed on the declaring element. Normal needs later font metrics. |
| Hyphenation | None, manual, auto; no hyphenation is performed and soft-hyphen bytes remain untouched. |
| Direction/bidi | LTR/RTL; normal, embed, isolate, bidi-override, isolate-override, plaintext. Bidi algorithms and isolate ranges belong to the application/preparation layer. |
| HTML semantics | Canonical `lang` (empty resets to `und`), `dir=ltr/rtl/auto`, implicit auto/isolate on `bdi`, bold on `b`/`strong`, italic on `i`/`em`, pre whitespace on `pre`. Author declarations override these defaults/hints. |

Inherited values and CSS `inherit`, `initial`, `unset` are computed before any
detachment. `unicode-bidi` resets on child elements; text nodes carry their
parent's complete context. `dir=auto` scans original text even outside the
selected nodes, skipping independently directed descendants, `bdi`, inert
templates, scripts, styles and textarea contents. Neutral-only scans use the
source parent's direction. This is the bounded HTML subset migrated from tytui,
not general HTML form-control or writing-mode support.

Deferred math/substitutions, metric/viewport font units, zero font size,
fractional font weights, advanced whitespace syntax, and cascade rollback
(`revert`/`revert-layer`) return errors. This service checks effective values;
strict loaders must inspect authored declarations before the permissive CSSOM
drops invalid or losing declarations. Unrelated properties are not audited here.

`ValidateDeclaration(property, value)` checks authored text-style longhands before
cascade reduction, including losing and nonmatching declarations. It accepts
the snapshot's text properties and the keyword forms below, plus CSS-wide
inherit/initial/unset. Relative font sizes and line-heights are checked without
inventing a source font size; effective `Resolve` still checks their computed
range. Unknown properties, unsupported syntax and rollback keywords fail.

`ParseKeyword(property, value)` decodes and checks the keyword form of font-style,
white-space, direction, unicode-bidi and hyphens. It also supports normal for
line-height/word-spacing/letter-spacing, normal/break-all/keep-all for word-break,
and normal/break-word/anywhere for overflow-wrap. It does not accept CSS-wide
keywords, numbers or lengths. Strict applications such as pagelayout use these
shared grammar helpers while choosing their own narrower preparation profile.

`*textstyle.Error` carries the original HTML node, property/value and wrapped
cause, so callers can map diagnostics to their own source identities without
parsing error text. Errors retain the source node; successful snapshots do not.
CSS value failures support `errors.Is(err, style.ErrInvalidValue)`.

The tytui facade maps these values to `typography/style.Style`, explicitly
converting DOM DU to typography DU and directions to its bidi enum. Line-height
and hyphenation remain separate policies, since that typography type has no
fields for them. Pagelayout's content projection retains the complete DOM
snapshot before creating its preparation adapter.

Use those source font snapshots with `css.GetBorderWithFontContext` for em/rem
border widths. Its `css.FontContext` accepts the root computed size and a lookup
for element/ancestor sizes, so inherited widths retain their original computed
value. See [effective borders](../README.md#effective-borders-and-source-fonts).

Run `go test ./style/textstyle` and `go test -race ./style/textstyle` for source
inheritance, HTML/CSS precedence, checked values, policies, structured failures,
snapshot ownership and concurrent calls. Existing tytui extraction, display,
paragraph and document-layout regressions exercise the compatibility facade.
