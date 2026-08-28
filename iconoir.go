// Copyright (c) 2026 the go-icons authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package iconoir serves UI icons from the Iconoir set (iconoir.com, MIT — see
// ICONOIR-LICENSE), as SVG documents keyed by the icon's own name.
//
// It is a data package: a curated subset of the line ("regular") icons is
// embedded, and [Icon] returns one by name ("zoom-in", "folder", "undo", …). A
// renderer such as go-widgets/toolkit's SVGIcon turns the SVG into a drawn glyph
// (Iconoir icons stroke in currentColor, so the renderer's ink recolours them);
// this package draws nothing itself.
package iconoir

import (
	"embed"
	"sort"
	"strings"
)

//go:embed svg/*.svg
var files embed.FS

// Icon returns the SVG document for the Iconoir icon named name (e.g. "zoom-in"),
// or "" when the icon is not in the embedded subset.
func Icon(name string) string {
	b, err := files.ReadFile("svg/" + name + ".svg")
	if err != nil {
		return ""
	}
	return string(b)
}

// Has reports whether name is in the embedded subset.
func Has(name string) bool { return Icon(name) != "" }

// Names lists the embedded icon names, sorted.
func Names() []string {
	entries, _ := files.ReadDir("svg")
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".svg"))
	}
	sort.Strings(out)
	return out
}
