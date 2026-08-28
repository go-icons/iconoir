# iconoir

The [Iconoir](https://iconoir.com) icon set (MIT), embedded as SVG and keyed by
the icon's own name — for pure-Go UIs that render their own icons.

```go
import "github.com/go-icons/iconoir"

svg := iconoir.Icon("zoom-in") // the Iconoir zoom-in glyph, as an SVG string
```

`Icon(name)` returns the line ("regular") icon by name (`zoom-in`, `folder`,
`undo`, `settings`, `nav-arrow-left`, …), or `""` when unknown; `Has` and `Names`
enumerate the full set.

It is a **data package**: it returns SVG strings and draws nothing. Rendering is
a separate concern — a renderer such as
[go-widgets/toolkit](https://github.com/go-widgets/toolkit)'s `SVGIcon`, over the
[go-gfx](https://github.com/go-gfx/gfx) SVG rasteriser, turns the SVG into a
glyph. Iconoir icons stroke in `currentColor`, so the renderer's ink recolours
them.

## Licence

The Go code is BSD-3-Clause (`LICENSE`). The embedded Iconoir artwork is MIT
(`ICONOIR-LICENSE`) — redistributed unmodified.
