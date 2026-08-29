// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package iconoir

import (
	"image"
	"math"

	"golang.org/x/image/vector"
)

// f32pt is a device-space (pixel) point.
type f32pt struct{ x, y float32 }

// render rasterizes the icon into a fresh alpha mask of sizePx×sizePx.
// Regular icons are drawn as round-capped/round-joined thick strokes;
// Solid icons are filled with nonzero winding.
func (ic *Icon) render(sizePx int, strokeOverride float64) *image.Alpha {
	doc := ic.parsed()
	sx := float64(sizePx) / doc.vbW
	sy := float64(sizePx) / doc.vbH
	avg := (sx + sy) / 2
	flat := 0.15 / math.Max(sx, sy)

	z := vector.NewRasterizer(sizePx, sizePx)
	tx := func(p ptf) f32pt {
		return f32pt{
			x: float32((p.x - doc.vbMinX) * sx),
			y: float32((p.y - doc.vbMinY) * sy),
		}
	}

	for _, pe := range doc.paths {
		subs := parsePath(pe.d, flat)
		if ic.variant == Solid {
			renderFill(z, subs, tx)
			continue
		}
		var strokeDev float64
		if strokeOverride > 0 {
			strokeDev = strokeOverride
		} else {
			sw := doc.strokeW
			if pe.strokeW > 0 {
				sw = pe.strokeW
			}
			strokeDev = sw * avg
		}
		renderStroke(z, subs, tx, strokeDev)
	}

	dst := image.NewAlpha(image.Rect(0, 0, sizePx, sizePx))
	z.Draw(dst, dst.Bounds(), image.Opaque, image.Point{})
	return dst
}

// renderFill adds each subpath to the rasterizer as a closed contour,
// preserving winding so oppositely-wound holes cut out correctly.
func renderFill(z *vector.Rasterizer, subs []subpath, tx func(ptf) f32pt) {
	for _, sp := range subs {
		if len(sp.pts) < 2 {
			continue
		}
		p0 := tx(sp.pts[0])
		z.MoveTo(p0.x, p0.y)
		for _, p := range sp.pts[1:] {
			d := tx(p)
			z.LineTo(d.x, d.y)
		}
		z.ClosePath()
	}
}

// renderStroke draws each subpath as a union of round-capped thick
// segments plus a filled disc at every vertex (giving round joins and
// round caps) with a consistent winding so the union accumulates.
func renderStroke(z *vector.Rasterizer, subs []subpath, tx func(ptf) f32pt, strokeDev float64) {
	r := strokeDev / 2
	for _, sp := range subs {
		dp := make([]f32pt, len(sp.pts))
		for i, p := range sp.pts {
			dp[i] = tx(p)
		}
		if len(dp) == 1 {
			addDisc(z, dp[0], r)
			continue
		}
		for i := 0; i < len(dp)-1; i++ {
			addSegQuad(z, dp[i], dp[i+1], r)
		}
		if sp.closed {
			addSegQuad(z, dp[len(dp)-1], dp[0], r)
		}
		for _, p := range dp {
			addDisc(z, p, r)
		}
	}
}

// addSegQuad adds a filled rectangle covering the segment a→b with the
// given half-width r.
func addSegQuad(z *vector.Rasterizer, a, b f32pt, r float64) {
	dx := float64(b.x - a.x)
	dy := float64(b.y - a.y)
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx := float32(-dy / l * r)
	ny := float32(dx / l * r)
	fillPoly(z, []f32pt{
		{a.x + nx, a.y + ny},
		{b.x + nx, b.y + ny},
		{b.x - nx, b.y - ny},
		{a.x - nx, a.y - ny},
	})
}

// addDisc adds a filled polygonal approximation of a disc of radius r.
func addDisc(z *vector.Rasterizer, c f32pt, r float64) {
	n := int(math.Max(8, r*2))
	pts := make([]f32pt, n)
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = f32pt{c.x + float32(r*math.Cos(a)), c.y + float32(r*math.Sin(a))}
	}
	fillPoly(z, pts)
}

// fillPoly emits a closed contour, forcing a positive signed area so that
// every stroke shape shares one winding direction and overlaps union
// (rather than cancel) under nonzero winding.
func fillPoly(z *vector.Rasterizer, pts []f32pt) {
	if signedArea(pts) < 0 {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	z.MoveTo(pts[0].x, pts[0].y)
	for _, p := range pts[1:] {
		z.LineTo(p.x, p.y)
	}
	z.ClosePath()
}

func signedArea(p []f32pt) float64 {
	a := 0.0
	for i := range p {
		j := (i + 1) % len(p)
		a += float64(p[i].x)*float64(p[j].y) - float64(p[j].x)*float64(p[i].y)
	}
	return a
}
