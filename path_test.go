// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package iconoir

import (
	"math"
	"testing"
)

// ---- scanner: readNumber / readFlag branches ------------------------------

func TestReadNumber(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"1e3", 1000},
		{"1E-2", 0.01},
		{"+2.5", 2.5},
		{".5", 0.5},
		{"-.25", -0.25},
		{"5e", 5},  // exponent with no digit → number stops before 'e'
		{"3e+", 3}, // exponent sign but no digit
		{"12", 12}, //
	}
	for _, c := range cases {
		sc := &dScanner{s: c.in}
		got := sc.readNumber()
		if math.Abs(got-c.want) > 1e-9 || sc.bad {
			t.Fatalf("readNumber(%q)=%v bad=%v, want %v", c.in, got, sc.bad, c.want)
		}
	}
	// no number present → bad.
	sc := &dScanner{s: "xyz"}
	if sc.readNumber(); !sc.bad {
		t.Fatal("expected bad on non-number")
	}
	// lone sign → bad.
	sc = &dScanner{s: "-"}
	if sc.readNumber(); !sc.bad {
		t.Fatal("expected bad on lone sign")
	}
}

func TestReadFlag(t *testing.T) {
	sc := &dScanner{s: "1"}
	if sc.readFlag() != 1 || sc.bad {
		t.Fatal("readFlag(1) wrong")
	}
	sc = &dScanner{s: " 0"}
	if sc.readFlag() != 0 || sc.bad {
		t.Fatal("readFlag(0) wrong")
	}
	sc = &dScanner{s: "9"}
	if sc.readFlag(); !sc.bad {
		t.Fatal("readFlag(9) should be bad")
	}
	sc = &dScanner{s: ""}
	if sc.readFlag(); !sc.bad {
		t.Fatal("readFlag(eof) should be bad")
	}
}

func TestSkipSepVariety(t *testing.T) {
	sc := &dScanner{s: " ,\t\n\r5"}
	if sc.readNumber() != 5 {
		t.Fatal("separators not skipped")
	}
}

// ---- parsePath: every command --------------------------------------------

func lastPoint(subs []subpath) ptf {
	sp := subs[len(subs)-1]
	return sp.pts[len(sp.pts)-1]
}

func TestParsePathCommands(t *testing.T) {
	// absolute M with implicit lineto, L, H, V.
	subs := parsePath("M1 1 2 2 L3 3 H5 V7", 0.1)
	if p := lastPoint(subs); p.x != 5 || p.y != 7 {
		t.Fatalf("M/L/H/V end = %+v", p)
	}
	// relative variants m/l/h/v, incl. relative m with implicit lineto.
	subs = parsePath("m1 1 2 2 l1 0 h1 v1", 0.1)
	if p := lastPoint(subs); math.Abs(p.x-5) > 1e-9 || math.Abs(p.y-4) > 1e-9 {
		t.Fatalf("m/l/h/v end = %+v", p)
	}
	// cubic C then smooth S (with and without preceding curve).
	subs = parsePath("M0 0 C1 0 1 1 2 1 S3 2 4 1", 0.01)
	if len(subs[0].pts) < 4 {
		t.Fatalf("cubic/smooth produced too few points: %d", len(subs[0].pts))
	}
	// smooth S with no preceding cubic (control = current point).
	subs = parsePath("M0 0 S1 1 2 0", 0.01)
	if len(subs[0].pts) < 2 {
		t.Fatal("lone S produced nothing")
	}
	// quadratic Q then smooth T (with and without preceding quad).
	subs = parsePath("M0 0 Q1 1 2 0 T4 0", 0.01)
	if len(subs[0].pts) < 3 {
		t.Fatal("Q/T produced too few points")
	}
	subs = parsePath("M0 0 T2 0", 0.01) // lone T
	if len(subs[0].pts) < 1 {
		t.Fatal("lone T produced nothing")
	}
	// relative curves c/s/q/t.
	if p := parsePath("M0 0 c1 0 1 1 2 1 s1 1 2 0 q1 1 2 0 t2 0", 0.05); len(p) == 0 {
		t.Fatal("relative curves produced nothing")
	}
	// Z closes; a second subpath follows.
	subs = parsePath("M0 0 L2 0 L2 2 Z M5 5 L6 6", 0.1)
	if len(subs) != 2 || !subs[0].closed {
		t.Fatalf("expected 2 subpaths, first closed; got %d closed=%v", len(subs), subs[0].closed)
	}
	// unknown command stops parsing; the good prefix survives.
	subs = parsePath("M1 1 L2 2 G 9 9", 0.1)
	if len(subs) != 1 {
		t.Fatalf("unknown-command parse produced %d subpaths", len(subs))
	}
	// truncated command (missing args) breaks cleanly.
	if p := parsePath("M1 1 L", 0.1); len(p) == 0 {
		t.Fatal("truncated parse lost the move point")
	}
	// leading garbage → nothing.
	if p := parsePath("Z", 0.1); len(p) != 0 {
		t.Fatalf("leading Z produced %d subpaths", len(p))
	}
}

// ---- flattenCubic degenerate (coincident endpoints) -----------------------

func TestFlattenCubicDegenerate(t *testing.T) {
	// start == end: the chord is a point → distToLine l2==0 branch.
	subs := parsePath("M0 0 C1 0 0 1 0 0", 0.01)
	if len(subs[0].pts) < 2 {
		t.Fatal("degenerate cubic produced no interior points")
	}
}

// ---- arcs: exercise every flattenArc branch -------------------------------

func TestFlattenArc(t *testing.T) {
	// normal large arc, laf==swf (sign=-1), sweep=1 dtheta<0 correction path.
	subs := parsePath("M2 12 A10 10 0 1 1 22 12", 0.05)
	if len(subs[0].pts) < 4 {
		t.Fatalf("arc produced too few points: %d", len(subs[0].pts))
	}
	// laf != swf (sign=+1), sweep=0 with dtheta>0 correction path.
	subs = parsePath("M2 12 A10 10 0 1 0 22 12", 0.05)
	if len(subs[0].pts) < 4 {
		t.Fatal("arc(sweep=0) produced too few points")
	}
	// radii too small → lambda>1 scale-up branch.
	subs = parsePath("M0 0 A1 1 0 0 0 10 0", 0.05)
	if len(subs[0].pts) < 2 {
		t.Fatal("undersized-radius arc produced nothing")
	}
	// rx==0 → straight line to endpoint.
	subs = parsePath("M0 0 A0 5 0 0 0 10 0", 0.05)
	if p := lastPoint(subs); p.x != 10 || p.y != 0 {
		t.Fatalf("zero-radius arc end = %+v", p)
	}
	// endpoints equal → arc omitted (only the move point remains).
	subs = parsePath("M5 5 A3 3 0 0 0 5 5", 0.05)
	if len(subs[0].pts) != 1 {
		t.Fatalf("degenerate arc added points: %d", len(subs[0].pts))
	}
	// tiny radius with a large tolerance → r<=flat else branch.
	subs = parsePath("M0 0 A0.5 0.5 0 0 0 0.5 0", 1.0)
	if len(subs[0].pts) < 2 {
		t.Fatal("tiny-radius arc produced nothing")
	}
	// rotated arc (non-zero x-axis rotation) sanity.
	subs = parsePath("M2 2 A6 3 30 0 1 18 10", 0.05)
	if len(subs[0].pts) < 4 {
		t.Fatal("rotated arc produced too few points")
	}
	// relative arc (a) → exercises the rel-offset branch.
	subs = parsePath("M2 12 a10 10 0 1 1 20 0", 0.05)
	if len(subs[0].pts) < 4 {
		t.Fatal("relative arc produced too few points")
	}
	// several sweep/large/direction combos to hit both dtheta corrections.
	for _, d := range []string{
		"M6 12 A10 10 0 1 1 18 12", // non-diametric large sweep=1 → raw dtheta<0
		"M18 12 A10 10 0 1 1 6 12", //
		"M22 12 A10 10 0 0 1 2 12", //
		"M2 12 A10 10 0 0 0 22 12", //
		"M12 2 A10 10 0 1 1 12 22", //
	} {
		if p := parsePath(d, 0.05); len(p[0].pts) < 2 {
			t.Fatalf("arc %q produced too few points", d)
		}
	}
}

// ---- svg parsing ----------------------------------------------------------

func TestParseViewBox(t *testing.T) {
	minX, minY, w, h, ok := parseViewBox("1 2 24 26")
	if !ok || minX != 1 || minY != 2 || w != 24 || h != 26 {
		t.Fatalf("parseViewBox = %v %v %v %v %v", minX, minY, w, h, ok)
	}
	if _, _, _, _, ok := parseViewBox("1 2 3"); ok {
		t.Fatal("short viewBox reported ok")
	}
}

func TestParseSVG(t *testing.T) {
	doc, err := parseSVG([]byte(
		`<svg viewBox="0 0 24 24" stroke-width="2">` +
			`<!-- comment --><g><path d="M0 0 L1 1" stroke-width="3"/>` +
			`<path/><rect width="1" height="1"/></g></svg>`))
	if err != nil {
		t.Fatalf("parseSVG error: %v", err)
	}
	if doc.strokeW != 2 || doc.vbW != 24 {
		t.Fatalf("svg attrs not read: sw=%v vbW=%v", doc.strokeW, doc.vbW)
	}
	if len(doc.paths) != 1 || doc.paths[0].strokeW != 3 {
		t.Fatalf("path parse wrong: %+v", doc.paths)
	}
	// malformed XML → error.
	if _, err := parseSVG([]byte(`<svg><path d="x"></svg>`)); err == nil {
		t.Fatal("expected error on malformed svg")
	}
	// non-numeric attrs are ignored (defaults kept).
	doc, _ = parseSVG([]byte(`<svg viewBox="bad" stroke-width="oops"><path d="M0 0"/></svg>`))
	if doc.vbW != 24 || doc.strokeW != 1.5 {
		t.Fatalf("bad attrs not defaulted: vbW=%v sw=%v", doc.vbW, doc.strokeW)
	}
}
