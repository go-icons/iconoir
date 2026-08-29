// Copyright (c) 2026, the go-icons authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file.

package iconoir

// SVG is the icon's source document, for a caller that wants the vector rather
// than pixels — a web page, a PDF, another rasteriser.
//
// It was called Icon until the renderer moved in here, and a package cannot
// hold both a function and a type of that name. Rendering is what nearly every
// caller wants (ninety-six call sites against ten), so the type kept the name
// and this took the one that says what it returns.
func SVG(name string) string {
	ic, ok := Get(name)
	if !ok {
		return ""
	}
	return string(ic.raw)
}

// Has reports whether the set holds an icon by that name.
func Has(name string) bool {
	_, ok := Get(name)
	return ok
}
