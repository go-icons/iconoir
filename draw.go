// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package iconoir

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/go-widgets/painter"
)

// DrawInto composites the icon into dst, fitted (centered) inside r and
// tinted with col. Coverage from the anti-aliased mask modulates col's
// alpha, and the result is src-over blended onto dst.
func (ic *Icon) DrawInto(dst *image.RGBA, r image.Rectangle, col color.Color) {
	size := min(r.Dx(), r.Dy())
	if size < 1 {
		return
	}
	mask := ic.Rasterize(size)
	off := image.Pt(r.Min.X+(r.Dx()-size)/2, r.Min.Y+(r.Dy()-size)/2)
	dr := image.Rectangle{Min: off, Max: off.Add(image.Pt(size, size))}
	draw.DrawMask(dst, dr, image.NewUniform(col), image.Point{}, mask, image.Point{}, draw.Over)
}

// DrawIcon rasterizes ic at r's (square, centered) size and blits it
// through the painter, modulating ink's alpha by each pixel's coverage.
// Its signature matches the icon-drawing callback that go-widgets/toolkit
// widgets (Banner.Icon, IconButton.Icon, …) expect, so it composes
// directly as one of those callbacks.
func DrawIcon(p painter.Painter, r painter.Rect, ic *Icon, ink painter.RGBA) {
	size := r.W
	if r.H < size {
		size = r.H
	}
	if size < 1 {
		return
	}
	mask := ic.Rasterize(size)
	offX := r.X + (r.W-size)/2
	offY := r.Y + (r.H-size)/2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			cov := uint32(mask.AlphaAt(x, y).A)
			if cov == 0 {
				continue
			}
			a := uint32(ink.A) * cov / 255
			p.PutPixel(offX+x, offY+y, painter.RGBA{R: ink.R, G: ink.G, B: ink.B, A: uint8(a)})
		}
	}
}

// Draw looks the Regular icon up by name and blits it via DrawIcon. It
// reports whether the icon exists; an unknown name draws nothing.
func Draw(p painter.Painter, r painter.Rect, name string, ink painter.RGBA) bool {
	ic, ok := Get(name)
	if !ok {
		return false
	}
	DrawIcon(p, r, ic, ink)
	return true
}
