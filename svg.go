// Copyright (c) the go-iconoir/iconoir authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package iconoir

import (
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

// svgDoc is the minimal parsed form of an Iconoir SVG: the viewBox
// extent, the default stroke width and the ordered list of <path>
// elements.
type svgDoc struct {
	vbMinX, vbMinY float64
	vbW, vbH       float64
	strokeW        float64
	paths          []svgPath
}

// svgPath holds a single <path>'s geometry data plus an optional
// per-path stroke width (-1 means inherit the document default).
type svgPath struct {
	d       string
	strokeW float64
}

// parseViewBox parses "minX minY width height"; ok is false when the
// value does not contain four numbers.
func parseViewBox(v string) (minX, minY, w, h float64, ok bool) {
	f := strings.Fields(v)
	if len(f) != 4 {
		return 0, 0, 0, 0, false
	}
	minX, _ = strconv.ParseFloat(f[0], 64)
	minY, _ = strconv.ParseFloat(f[1], 64)
	w, _ = strconv.ParseFloat(f[2], 64)
	h, _ = strconv.ParseFloat(f[3], 64)
	return minX, minY, w, h, true
}

// parseSVG extracts the viewBox, stroke width and every <path> element
// (at any nesting depth) from an SVG document.
func parseSVG(data []byte) (*svgDoc, error) {
	doc := &svgDoc{vbW: 24, vbH: 24, strokeW: 1.5}
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "svg":
			for _, a := range se.Attr {
				switch a.Name.Local {
				case "viewBox":
					if minX, minY, w, h, ok := parseViewBox(a.Value); ok {
						doc.vbMinX, doc.vbMinY, doc.vbW, doc.vbH = minX, minY, w, h
					}
				case "stroke-width":
					if v, err := strconv.ParseFloat(a.Value, 64); err == nil {
						doc.strokeW = v
					}
				}
			}
		case "path":
			p := svgPath{strokeW: -1}
			for _, a := range se.Attr {
				switch a.Name.Local {
				case "d":
					p.d = a.Value
				case "stroke-width":
					if v, err := strconv.ParseFloat(a.Value, 64); err == nil {
						p.strokeW = v
					}
				}
			}
			if p.d != "" {
				doc.paths = append(doc.paths, p)
			}
		}
	}
	return doc, nil
}
