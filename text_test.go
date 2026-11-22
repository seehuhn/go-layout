// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"seehuhn.de/go/geom/matrix"
	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/font/type3"
	"seehuhn.de/go/pdf/graphics/content"
	"seehuhn.de/go/pdf/graphics/content/builder"
)

// approx compares extents up to floating-point rounding.
var approx = cmpopts.EquateApprox(0, 1e-9)

// testType3Font returns a Type 3 font with ascent 800 and descent -200 in
// glyph space.  Glyph "A" strokes a path without setting the line width, so
// that its ink has no bound; glyph "space" is blank.  Both glyphs have
// width 500.
func testType3Font(t *testing.T) font.Layouter {
	t.Helper()

	b := builder.New(content.Glyph, nil, pdf.V2_0)
	b.Type3ColoredGlyph(500, 0)
	b.MoveTo(0, 0)
	b.LineTo(500, 700)
	b.Stroke()
	unbounded, err := b.Harvest()
	if err != nil {
		t.Fatal(err)
	}

	b = builder.New(content.Glyph, nil, pdf.V2_0)
	b.Type3ColoredGlyph(500, 0)
	blank, err := b.Harvest()
	if err != nil {
		t.Fatal(err)
	}

	F, err := (&type3.Font{
		Glyphs: []*type3.Glyph{
			{},
			{Name: "A", Content: unbounded},
			{Name: "space", Content: blank},
		},
		FontMatrix: matrix.Matrix{0.001, 0, 0, 0.001, 0, 0},
		Ascent:     800,
		Descent:    -200,
	}).New()
	if err != nil {
		t.Fatal(err)
	}
	return F
}

// TestTextExtentUnbounded checks that a glyph whose ink has no bound
// extends a text box to the font's ascent and descent.
func TestTextExtentUnbounded(t *testing.T) {
	F := testType3Font(t)

	box := Text(&FontInfo{Font: F, Size: 10}, "AA")
	want := &BoxExtent{Width: 10, Height: 8, Depth: 2}
	if diff := cmp.Diff(want, box.Extent(), approx); diff != "" {
		t.Errorf("wrong extent (-want +got):\n%s", diff)
	}
}

// TestTextExtentBlank checks that a text box without ink has zero height
// and depth.
func TestTextExtentBlank(t *testing.T) {
	F := testType3Font(t)

	for _, text := range []string{"", "  "} {
		box := Text(&FontInfo{Font: F, Size: 10}, text)
		want := &BoxExtent{Width: 5 * float64(len(text))}
		if diff := cmp.Diff(want, box.Extent(), approx); diff != "" {
			t.Errorf("%q: wrong extent (-want +got):\n%s", text, diff)
		}
	}
}
