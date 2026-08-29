// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package iconoir provides idiomatic, pure-Go (CGO=0) access to the
// Iconoir icon set (https://iconoir.com). Every icon is a 24×24 SVG that
// this package rasterizes to an anti-aliased alpha coverage mask with a
// small self-contained SVG-path renderer (no third-party SVG library),
// then blits through the go-widgets painter or any *image.RGBA.
//
// The Iconoir artwork is MIT-licensed (see LICENSE-ICONOIR); the Go code
// in this package is BSD-3-Clause (see LICENSE).
package iconoir

import (
	"embed"
	"fmt"
	"image"
	"io/fs"
	"math"
	"sort"
	"strings"
	"sync"
)

//go:embed assets/regular/*.svg
var regularFS embed.FS

//go:embed assets/solid/*.svg
var solidFS embed.FS

// Variant selects which Iconoir style an Icon renders in.
type Variant int

const (
	// Regular is the stroked, 1.5-unit outline style.
	Regular Variant = iota
	// Solid is the filled style.
	Solid
)

// Icon is a single, lazily-parsed Iconoir glyph. Rasterized masks are
// cached per (size, stroke) key; an Icon is safe for concurrent use.
type Icon struct {
	name    string
	variant Variant
	raw     []byte

	parseOnce sync.Once
	doc       *svgDoc

	mu    sync.Mutex
	cache map[rasterKey]*image.Alpha
}

// Name returns the icon's kebab-case identifier (e.g. "arrow-up").
func (ic *Icon) Name() string { return ic.name }

// Variant returns whether the icon renders Regular (stroked) or Solid.
func (ic *Icon) Variant() Variant { return ic.variant }

func newIcon(name string, v Variant, raw []byte) *Icon {
	return &Icon{name: name, variant: v, raw: raw}
}

func (ic *Icon) parsed() *svgDoc {
	ic.parseOnce.Do(func() {
		d, err := parseSVG(ic.raw)
		if err != nil {
			d = &svgDoc{vbW: 24, vbH: 24, strokeW: 1.5}
		}
		ic.doc = d
	})
	return ic.doc
}

// registry is a lazily-populated name→Icon table for one variant.
type registry struct {
	fsys    fs.FS
	dir     string
	variant Variant

	once   sync.Once
	byName map[string]*Icon
	names  []string
}

func (r *registry) load() {
	r.once.Do(func() {
		r.byName = make(map[string]*Icon)
		entries, _ := fs.ReadDir(r.fsys, r.dir)
		for _, e := range entries {
			name := strings.TrimSuffix(e.Name(), ".svg")
			raw, _ := fs.ReadFile(r.fsys, r.dir+"/"+e.Name())
			r.byName[name] = newIcon(name, r.variant, raw)
			r.names = append(r.names, name)
		}
		sort.Strings(r.names)
	})
}

func (r *registry) get(name string) (*Icon, bool) {
	r.load()
	ic, ok := r.byName[name]
	return ic, ok
}

func (r *registry) list() []string {
	r.load()
	return append([]string(nil), r.names...)
}

var (
	regularReg = &registry{fsys: regularFS, dir: "assets/regular", variant: Regular}
	solidReg   = &registry{fsys: solidFS, dir: "assets/solid", variant: Solid}
)

// Get returns the Regular (stroked) icon with the given name.
func Get(name string) (*Icon, bool) { return regularReg.get(name) }

// GetSolid returns the Solid (filled) icon with the given name.
func GetSolid(name string) (*Icon, bool) { return solidReg.get(name) }

// MustGet is Get but panics if the icon does not exist. Handy for
// package-level icon references that must exist.
func MustGet(name string) *Icon {
	ic, ok := Get(name)
	if !ok {
		panic(fmt.Sprintf("iconoir: unknown icon %q", name))
	}
	return ic
}

// Names returns the sorted names of every Regular icon.
func Names() []string { return regularReg.list() }

// SolidNames returns the sorted names of every Solid icon.
func SolidNames() []string { return solidReg.list() }

// rasterKey identifies a cached mask. stroke is the forced stroke width
// scaled by 256 (a negative value means "use the SVG's own widths").
type rasterKey struct {
	size   int
	stroke int64
}

// Rasterize returns the icon's anti-aliased coverage mask at sizePx×sizePx
// using the icon's own stroke width(s). The result is cached and must not
// be mutated by the caller.
func (ic *Icon) Rasterize(sizePx int) *image.Alpha {
	return ic.rasterize(sizePx, -1)
}

// RasterizeStroke is like Rasterize but forces every stroke to strokePx
// device pixels wide (ignored for Solid icons, which are filled).
func (ic *Icon) RasterizeStroke(sizePx int, strokePx float64) *image.Alpha {
	return ic.rasterize(sizePx, strokePx)
}

func (ic *Icon) rasterize(sizePx int, strokeOverride float64) *image.Alpha {
	if sizePx < 1 {
		sizePx = 1
	}
	key := rasterKey{size: sizePx, stroke: int64(math.Round(strokeOverride * 256))}
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if ic.cache == nil {
		ic.cache = make(map[rasterKey]*image.Alpha)
	}
	if a, ok := ic.cache[key]; ok {
		return a
	}
	a := ic.render(sizePx, strokeOverride)
	ic.cache[key] = a
	return a
}
