// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package iconoir

import (
	"math"
	"strconv"
)

// ptf is a point in the SVG viewBox coordinate space.
type ptf struct{ x, y float64 }

// subpath is one contiguous polyline produced by flattening an SVG
// path's curves/arcs to straight segments. closed reports whether the
// original subpath ended with a Z/z command.
type subpath struct {
	pts    []ptf
	closed bool
}

// dScanner walks an SVG path "d" attribute a token at a time. It is
// deliberately lenient: on the first malformed token it flips bad and
// the caller stops, keeping whatever was parsed so far (this mirrors how
// forgiving SVG renderers behave and avoids an unreachable error path).
type dScanner struct {
	s   string
	i   int
	bad bool
}

func (sc *dScanner) skipSep() {
	for sc.i < len(sc.s) {
		c := sc.s[sc.i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',' {
			sc.i++
			continue
		}
		break
	}
}

// peekIsNumberStart reports whether the next non-separator byte begins a
// number, without consuming it.
func (sc *dScanner) peekIsNumberStart() bool {
	sc.skipSep()
	if sc.i >= len(sc.s) {
		return false
	}
	c := sc.s[sc.i]
	return c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9')
}

// readNumber consumes and returns one SVG number (with optional sign,
// fraction and exponent). It sets bad when no number is present.
func (sc *dScanner) readNumber() float64 {
	sc.skipSep()
	start := sc.i
	if sc.i < len(sc.s) && (sc.s[sc.i] == '+' || sc.s[sc.i] == '-') {
		sc.i++
	}
	seenDigit := false
	for sc.i < len(sc.s) && sc.s[sc.i] >= '0' && sc.s[sc.i] <= '9' {
		sc.i++
		seenDigit = true
	}
	if sc.i < len(sc.s) && sc.s[sc.i] == '.' {
		sc.i++
		for sc.i < len(sc.s) && sc.s[sc.i] >= '0' && sc.s[sc.i] <= '9' {
			sc.i++
			seenDigit = true
		}
	}
	if !seenDigit {
		sc.bad = true
		return 0
	}
	if sc.i < len(sc.s) && (sc.s[sc.i] == 'e' || sc.s[sc.i] == 'E') {
		j := sc.i + 1
		if j < len(sc.s) && (sc.s[j] == '+' || sc.s[j] == '-') {
			j++
		}
		if j < len(sc.s) && sc.s[j] >= '0' && sc.s[j] <= '9' {
			sc.i = j
			for sc.i < len(sc.s) && sc.s[sc.i] >= '0' && sc.s[sc.i] <= '9' {
				sc.i++
			}
		}
	}
	v, _ := strconv.ParseFloat(sc.s[start:sc.i], 64)
	return v
}

// readFlag consumes a single-character arc flag ('0' or '1').
func (sc *dScanner) readFlag() float64 {
	sc.skipSep()
	if sc.i < len(sc.s) && (sc.s[sc.i] == '0' || sc.s[sc.i] == '1') {
		c := sc.s[sc.i]
		sc.i++
		if c == '1' {
			return 1
		}
		return 0
	}
	sc.bad = true
	return 0
}

func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// parsePath flattens an SVG "d" string into a set of polyline subpaths.
// flat is the maximum chord error (in viewBox units) used when
// subdividing curves and arcs.
func parsePath(d string, flat float64) []subpath {
	sc := &dScanner{s: d}
	var subs []subpath
	var cur subpath
	var cx, cy, sx, sy float64
	var lastCX, lastCY float64 // last cubic control point (for S/s)
	var lastQX, lastQY float64 // last quadratic control point (for T/t)
	var prevCmd byte

	flush := func(closed bool) {
		if len(cur.pts) > 0 {
			cur.closed = closed
			subs = append(subs, cur)
			cur = subpath{}
		}
	}
	add := func(x, y float64) { cur.pts = append(cur.pts, ptf{x, y}) }

	for {
		sc.skipSep()
		if sc.i >= len(sc.s) || !isAlpha(sc.s[sc.i]) {
			break
		}
		c := sc.s[sc.i]
		sc.i++
		rel := c >= 'a' && c <= 'z'
		cmd := c
		if rel {
			cmd -= 'a' - 'A'
		}

		switch cmd {
		case 'M':
			x := sc.readNumber()
			y := sc.readNumber()
			if rel {
				x += cx
				y += cy
			}
			flush(false)
			cx, cy, sx, sy = x, y, x, y
			add(cx, cy)
			for sc.peekIsNumberStart() {
				x = sc.readNumber()
				y = sc.readNumber()
				if rel {
					x += cx
					y += cy
				}
				cx, cy = x, y
				add(cx, cy)
			}
		case 'L':
			for {
				x := sc.readNumber()
				y := sc.readNumber()
				if rel {
					x += cx
					y += cy
				}
				cx, cy = x, y
				add(cx, cy)
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'H':
			for {
				x := sc.readNumber()
				if rel {
					x += cx
				}
				cx = x
				add(cx, cy)
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'V':
			for {
				y := sc.readNumber()
				if rel {
					y += cy
				}
				cy = y
				add(cx, cy)
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'C':
			for {
				x1 := sc.readNumber()
				y1 := sc.readNumber()
				x2 := sc.readNumber()
				y2 := sc.readNumber()
				x := sc.readNumber()
				y := sc.readNumber()
				if rel {
					x1 += cx
					y1 += cy
					x2 += cx
					y2 += cy
					x += cx
					y += cy
				}
				flattenCubic(&cur, cx, cy, x1, y1, x2, y2, x, y, flat)
				lastCX, lastCY = x2, y2
				cx, cy = x, y
				prevCmd = 'C'
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'S':
			for {
				x2 := sc.readNumber()
				y2 := sc.readNumber()
				x := sc.readNumber()
				y := sc.readNumber()
				if rel {
					x2 += cx
					y2 += cy
					x += cx
					y += cy
				}
				x1, y1 := cx, cy
				if prevCmd == 'C' || prevCmd == 'S' {
					x1 = 2*cx - lastCX
					y1 = 2*cy - lastCY
				}
				flattenCubic(&cur, cx, cy, x1, y1, x2, y2, x, y, flat)
				lastCX, lastCY = x2, y2
				cx, cy = x, y
				prevCmd = 'S'
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'Q':
			for {
				qx := sc.readNumber()
				qy := sc.readNumber()
				x := sc.readNumber()
				y := sc.readNumber()
				if rel {
					qx += cx
					qy += cy
					x += cx
					y += cy
				}
				flattenQuad(&cur, cx, cy, qx, qy, x, y, flat)
				lastQX, lastQY = qx, qy
				cx, cy = x, y
				prevCmd = 'Q'
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'T':
			for {
				x := sc.readNumber()
				y := sc.readNumber()
				if rel {
					x += cx
					y += cy
				}
				qx, qy := cx, cy
				if prevCmd == 'Q' || prevCmd == 'T' {
					qx = 2*cx - lastQX
					qy = 2*cy - lastQY
				}
				flattenQuad(&cur, cx, cy, qx, qy, x, y, flat)
				lastQX, lastQY = qx, qy
				cx, cy = x, y
				prevCmd = 'T'
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'A':
			for {
				rx := sc.readNumber()
				ry := sc.readNumber()
				rot := sc.readNumber()
				laf := sc.readFlag()
				swf := sc.readFlag()
				x := sc.readNumber()
				y := sc.readNumber()
				if rel {
					x += cx
					y += cy
				}
				flattenArc(&cur, cx, cy, rx, ry, rot, laf != 0, swf != 0, x, y, flat)
				cx, cy = x, y
				prevCmd = 'A'
				if !sc.peekIsNumberStart() {
					break
				}
			}
		case 'Z':
			flush(true)
			cx, cy = sx, sy
			prevCmd = 'Z'
		default:
			sc.bad = true
		}

		if cmd != 'C' && cmd != 'S' && cmd != 'Q' && cmd != 'T' && cmd != 'A' {
			prevCmd = cmd
		}
		if sc.bad {
			break
		}
	}
	flush(false)
	return subs
}

// distToLine returns the perpendicular distance from (px,py) to the line
// through (ax,ay)-(bx,by); if the two line points coincide it falls back
// to point distance.
func distToLine(px, py, ax, ay, bx, by float64) float64 {
	dx := bx - ax
	dy := by - ay
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	cross := (px-ax)*dy - (py-ay)*dx
	return math.Abs(cross) / math.Sqrt(l2)
}

// flattenCubic subdivides a cubic Bézier and appends the resulting flat
// segment endpoints (never the start point, which is already present).
func flattenCubic(cur *subpath, x0, y0, x1, y1, x2, y2, x3, y3, flat float64) {
	const maxDepth = 18
	var rec func(x0, y0, x1, y1, x2, y2, x3, y3 float64, depth int)
	rec = func(x0, y0, x1, y1, x2, y2, x3, y3 float64, depth int) {
		d1 := distToLine(x1, y1, x0, y0, x3, y3)
		d2 := distToLine(x2, y2, x0, y0, x3, y3)
		if depth >= maxDepth || d1+d2 <= flat {
			cur.pts = append(cur.pts, ptf{x3, y3})
			return
		}
		x01, y01 := (x0+x1)/2, (y0+y1)/2
		x12, y12 := (x1+x2)/2, (y1+y2)/2
		x23, y23 := (x2+x3)/2, (y2+y3)/2
		xa, ya := (x01+x12)/2, (y01+y12)/2
		xb, yb := (x12+x23)/2, (y12+y23)/2
		xm, ym := (xa+xb)/2, (ya+yb)/2
		rec(x0, y0, x01, y01, xa, ya, xm, ym, depth+1)
		rec(xm, ym, xb, yb, x23, y23, x3, y3, depth+1)
	}
	rec(x0, y0, x1, y1, x2, y2, x3, y3, 0)
}

// flattenQuad elevates a quadratic Bézier to a cubic and reuses the cubic
// flattener.
func flattenQuad(cur *subpath, x0, y0, qx, qy, x1, y1, flat float64) {
	c1x := x0 + 2.0/3*(qx-x0)
	c1y := y0 + 2.0/3*(qy-y0)
	c2x := x1 + 2.0/3*(qx-x1)
	c2y := y1 + 2.0/3*(qy-y1)
	flattenCubic(cur, x0, y0, c1x, c1y, c2x, c2y, x1, y1, flat)
}

// vecAngle returns the signed angle (radians) from vector u to vector v.
func vecAngle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	l := math.Hypot(ux, uy) * math.Hypot(vx, vy)
	c := math.Min(1, math.Max(-1, dot/l))
	a := math.Acos(c)
	if ux*vy-uy*vx < 0 {
		a = -a
	}
	return a
}

// flattenArc converts an SVG elliptical arc (endpoint parameterization)
// to its center form and appends sampled points to cur.
func flattenArc(cur *subpath, x0, y0, rx, ry, rotDeg float64, largeArc, sweep bool, x1, y1, flat float64) {
	if x0 == x1 && y0 == y1 {
		return
	}
	if rx == 0 || ry == 0 {
		cur.pts = append(cur.pts, ptf{x1, y1})
		return
	}
	rx = math.Abs(rx)
	ry = math.Abs(ry)
	phi := rotDeg * math.Pi / 180
	cosPhi := math.Cos(phi)
	sinPhi := math.Sin(phi)

	dx := (x0 - x1) / 2
	dy := (y0 - y1) / 2
	x1p := cosPhi*dx + sinPhi*dy
	y1p := -sinPhi*dx + cosPhi*dy

	lambda := (x1p*x1p)/(rx*rx) + (y1p*y1p)/(ry*ry)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx *= s
		ry *= s
	}

	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	co := math.Sqrt(math.Max(0, num/den))
	sign := 1.0
	if largeArc == sweep {
		sign = -1
	}
	coef := sign * co
	cxp := coef * rx * y1p / ry
	cyp := coef * (-ry * x1p / rx)

	cx := cosPhi*cxp - sinPhi*cyp + (x0+x1)/2
	cy := sinPhi*cxp + cosPhi*cyp + (y0+y1)/2

	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := (-x1p - cxp) / rx
	vy := (-y1p - cyp) / ry
	theta1 := vecAngle(1, 0, ux, uy)
	dtheta := vecAngle(ux, uy, vx, vy)
	if !sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	}
	if sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	r := math.Max(rx, ry)
	maxStep := math.Pi / 2
	if r > flat {
		maxStep = 2 * math.Acos(1-flat/r)
	}
	n := int(math.Max(1, math.Ceil(math.Abs(dtheta)/maxStep)))
	for i := 1; i <= n; i++ {
		t := theta1 + dtheta*float64(i)/float64(n)
		ct := math.Cos(t)
		st := math.Sin(t)
		ex := cx + rx*ct*cosPhi - ry*st*sinPhi
		ey := cy + rx*ct*sinPhi + ry*st*cosPhi
		cur.pts = append(cur.pts, ptf{ex, ey})
	}
}
