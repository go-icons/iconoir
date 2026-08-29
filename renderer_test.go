// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package iconoir

import (
	"image"
	"image/color"
	"testing"

	"github.com/go-widgets/painter"
)

// ---- registry / lookup -----------------------------------------------------

func TestGetAndNames(t *testing.T) {
	names := Names()
	if len(names) < 1000 {
		t.Fatalf("expected many regular icons, got %d", len(names))
	}
	if len(SolidNames()) < 100 {
		t.Fatalf("expected many solid icons, got %d", len(SolidNames()))
	}
	ic, ok := Get("menu")
	if !ok {
		t.Fatal("menu not found")
	}
	if ic.Name() != "menu" || ic.Variant() != Regular {
		t.Fatalf("unexpected name/variant: %q %v", ic.Name(), ic.Variant())
	}
	if _, ok := Get("definitely-not-an-icon"); ok {
		t.Fatal("unknown icon reported as found")
	}
	sh, ok := GetSolid("heart")
	if !ok || sh.Variant() != Solid {
		t.Fatal("solid heart missing or wrong variant")
	}
}

func TestMustGet(t *testing.T) {
	if MustGet("heart").Name() != "heart" {
		t.Fatal("MustGet returned wrong icon")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet did not panic on unknown icon")
		}
	}()
	MustGet("nope-nope-nope")
}

// ---- self-validation: menu = 3 anti-aliased bands --------------------------

func TestMenuThreeBands(t *testing.T) {
	m := MustGet("menu").Rasterize(48)
	if m.Bounds().Dx() != 48 || m.Bounds().Dy() != 48 {
		t.Fatalf("wrong size: %v", m.Bounds())
	}
	col := 24
	bands, partial, inBand := 0, 0, false
	for y := 0; y < 48; y++ {
		a := m.AlphaAt(col, y).A
		if a > 0 && !inBand {
			bands++
			inBand = true
		}
		if a == 0 {
			inBand = false
		}
		if a > 0 && a < 255 {
			partial++
		}
	}
	if bands != 3 {
		t.Fatalf("menu: expected 3 horizontal bands, got %d", bands)
	}
	if partial == 0 {
		t.Fatal("menu: no partial-coverage (anti-aliased) pixels found")
	}
}

// ---- self-validation: curves (heart) & closed circle (ring) ---------------

func TestHeartCurvesNonEmpty(t *testing.T) {
	m := MustGet("heart").Rasterize(48)
	ink := 0
	for _, a := range m.Pix {
		if a > 0 {
			ink++
		}
	}
	if ink < 100 {
		t.Fatalf("heart: expected substantial coverage, got %d inked px", ink)
	}
}

func TestCircleIsARing(t *testing.T) {
	m := MustGet("circle").Rasterize(48)
	// centre must be empty, the ring must be inked.
	if m.AlphaAt(24, 24).A != 0 {
		t.Fatalf("circle centre should be empty, got alpha=%d", m.AlphaAt(24, 24).A)
	}
	ring := 0
	for x := 0; x < 48; x++ {
		if m.AlphaAt(x, 24).A > 0 {
			ring++
		}
	}
	// scanning through the middle row crosses the ring twice (left+right).
	if ring < 2 {
		t.Fatalf("circle: expected the ring on the centre row, got %d inked", ring)
	}
}

// ---- self-validation: painter adapter blits AA ink, clipped to r ----------

func TestPainterAdapterAA(t *testing.T) {
	const w, h = 60, 60
	buf := make([]byte, w*h*4)
	for i := 0; i < len(buf); i += 4 {
		buf[i], buf[i+1], buf[i+2], buf[i+3] = 255, 255, 255, 255 // white bg
	}
	p := painter.NewPixelPainter(buf, w, h)
	r := painter.Rect{X: 6, Y: 6, W: 48, H: 48}
	ink := painter.RGBA{R: 255, G: 0, B: 0, A: 255}
	if !Draw(p, r, "menu", ink) {
		t.Fatal("Draw returned false for known icon")
	}

	px := func(x, y int) (uint8, uint8, uint8) {
		o := (y*w + x) * 4
		return buf[o], buf[o+1], buf[o+2]
	}
	// corner outside r must stay white.
	if rr, g, b := px(0, 0); rr != 255 || g != 255 || b != 255 {
		t.Fatalf("pixel outside r was modified: %d,%d,%d", rr, g, b)
	}
	// find a fully-inked red pixel and a partially-blended (AA) one.
	full, partial := false, false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rr, g, b := px(x, y)
			if rr == 255 && g == 0 && b == 0 {
				full = true
			}
			if rr == 255 && g > 0 && g < 255 && b == g {
				partial = true // red blended over white → G,B equal & partial
			}
		}
	}
	if !full {
		t.Fatal("no fully-inked red pixel from menu blit")
	}
	if !partial {
		t.Fatal("no anti-aliased (partially blended) pixel from menu blit")
	}

	// unknown name draws nothing and reports false.
	if Draw(p, r, "no-such-icon", ink) {
		t.Fatal("Draw returned true for unknown icon")
	}
}

// ---- DrawIcon / DrawInto ---------------------------------------------------

func TestDrawIconNonSquareAndZero(t *testing.T) {
	buf := make([]byte, 40*20*4)
	p := painter.NewPixelPainter(buf, 40, 20)
	// non-square rect → size = min(W,H) = 16, exercises the H<W branch.
	DrawIcon(p, painter.Rect{X: 0, Y: 0, W: 40, H: 16}, MustGet("menu"), painter.RGBA{A: 255})
	// zero-size rect → no-op.
	DrawIcon(p, painter.Rect{X: 0, Y: 0, W: 0, H: 0}, MustGet("menu"), painter.RGBA{A: 255})
}

func TestDrawInto(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 50, 50))
	MustGet("heart").DrawInto(dst, image.Rect(1, 1, 49, 41), color.RGBA{0, 128, 255, 255})
	inked := false
	for i := 3; i < len(dst.Pix); i += 4 {
		if dst.Pix[i] != 0 {
			inked = true
			break
		}
	}
	if !inked {
		t.Fatal("DrawInto produced no output")
	}
	// zero-size rect → no-op.
	MustGet("heart").DrawInto(dst, image.Rect(5, 5, 5, 5), color.Black)
}

// ---- caching ---------------------------------------------------------------

func TestRasterCacheAndGuards(t *testing.T) {
	ic := MustGet("menu")
	a := ic.Rasterize(32)
	b := ic.Rasterize(32) // cache hit → same pointer
	if a != b {
		t.Fatal("cache miss on identical key")
	}
	if ic.Rasterize(33) == a {
		t.Fatal("different size returned cached mask")
	}
	// sub-1 size clamps to 1.
	if got := ic.Rasterize(0).Bounds().Dx(); got != 1 {
		t.Fatalf("Rasterize(0) size = %d, want 1", got)
	}
	if ic.RasterizeStroke(24, 4).Bounds().Dx() != 24 {
		t.Fatal("RasterizeStroke wrong size")
	}
}

// ---- solid fill + per-path stroke width + parse-error icon ----------------

func TestSyntheticIcons(t *testing.T) {
	// solid fill with a normal polygon subpath AND a degenerate 1-point
	// subpath (exercises the len<2 continue in renderFill).
	solid := newIcon("x", Solid, []byte(
		`<svg viewBox="0 0 24 24"><path d="M2 2 L20 2 L20 20 Z M5 5"/></svg>`))
	if sum(solid.Rasterize(24)) == 0 {
		t.Fatal("solid synthetic produced no ink")
	}
	// per-path stroke-width override (exercises pe.strokeW>0).
	sw := newIcon("x", Regular, []byte(
		`<svg viewBox="0 0 24 24"><path d="M2 2 L22 22" stroke-width="4"/></svg>`))
	if sum(sw.Rasterize(48)) == 0 {
		t.Fatal("stroke-width synthetic produced no ink")
	}
	// zero-length segment (exercises addSegQuad l==0 and disc-at-vertex).
	dot := newIcon("x", Regular, []byte(
		`<svg viewBox="0 0 24 24"><path d="M12 12 L12 12"/></svg>`))
	if sum(dot.Rasterize(48)) == 0 {
		t.Fatal("zero-length stroke produced no dot")
	}
	// lone move (single point) → round dot via len==1 branch.
	lone := newIcon("x", Regular, []byte(
		`<svg viewBox="0 0 24 24"><path d="M12 12"/></svg>`))
	if sum(lone.Rasterize(48)) == 0 {
		t.Fatal("lone-point stroke produced no dot")
	}
	// malformed SVG → parsed() falls back to an empty doc, renders blank.
	bad := newIcon("x", Regular, []byte(`<svg><path d="M0 0"></svg>`))
	if sum(bad.Rasterize(16)) != 0 {
		t.Fatal("malformed icon should render blank")
	}
	// RasterizeStroke on a solid icon still fills.
	if sum(solid.RasterizeStroke(24, 2)) == 0 {
		t.Fatal("solid RasterizeStroke empty")
	}
}

func sum(m *image.Alpha) int {
	t := 0
	for _, a := range m.Pix {
		t += int(a)
	}
	return t
}
