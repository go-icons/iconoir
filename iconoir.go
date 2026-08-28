// Copyright (c) 2026 the go-icons authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package iconoir serves the Iconoir icon set (iconoir.com, MIT — see
// ICONOIR-LICENSE) as SVG documents keyed by the icon's own name.
//
// It is a DATA package: it embeds the Iconoir line ("regular") SVGs and returns
// one by name; it draws nothing. Rendering is a separate concern — a renderer
// such as go-widgets/toolkit's SVGIcon (over the go-gfx SVG rasteriser) turns the
// returned SVG into a glyph. Iconoir icons stroke in currentColor, so the
// renderer's ink recolours them.
package iconoir

import (
	"embed"
	"sort"
	"strings"
)

//go:embed svg/*.svg
var files embed.FS

// Icon returns the SVG document for the Iconoir icon named name (e.g. "zoom-in",
// "folder", "nav-arrow-left"), or "" when the name is not in the set.
func Icon(name string) string {
	b, err := files.ReadFile("svg/" + name + ".svg")
	if err != nil {
		return ""
	}
	return string(b)
}

// Has reports whether name is a known icon.
func Has(name string) bool { return Icon(name) != "" }

// Names lists every embedded icon name, sorted.
func Names() []string {
	entries, _ := files.ReadDir("svg")
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".svg"))
	}
	sort.Strings(out)
	return out
}
