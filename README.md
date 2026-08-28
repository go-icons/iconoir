# iconoir

UI icons from the [Iconoir](https://iconoir.com) set (MIT), as embedded SVG
documents keyed by the icon's own name — for pure-Go UIs that render their own
icons.

```go
import "github.com/go-icons/iconoir"

svg := iconoir.Icon("zoom-in") // the Iconoir zoom-in glyph, as an SVG string
```

`Icon(name)` returns the line ("regular") icon by name (`zoom-in`, `folder`,
`undo`, `settings`, `nav-arrow-left`, …), or `""` when it is not in the embedded
subset; `Has` and `Names` enumerate. It is a **data package**: it returns SVG
strings and draws nothing. A renderer such as
[go-widgets/toolkit](https://github.com/go-widgets/toolkit)'s `SVGIcon` turns the
SVG into a drawn glyph — Iconoir icons stroke in `currentColor`, so the
renderer's ink recolours them.

A curated subset is embedded (toolbar and navigation staples). Contributions
adding more are welcome.

## Licence

The Go code is BSD-3-Clause (`LICENSE`). The embedded Iconoir artwork is MIT
(`ICONOIR-LICENSE`) — redistributed unmodified.
