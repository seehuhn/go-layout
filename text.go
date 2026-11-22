// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import (
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics/color"
	"seehuhn.de/go/pdf/graphics/content/builder"
)

// TextBox represents a typeset string of characters as a Box object.
// The text is typeset using a single font and size.
type TextBox struct {
	F      *FontInfo
	Glyphs *font.GlyphSeq
}

// FontInfo describes the font, size, and color to use for typesetting text.
type FontInfo struct {
	Font  font.Layouter
	Size  float64
	Color color.Color
}

// Text returns a new [TextBox] object.
func Text(F *FontInfo, text string) *TextBox {
	return &TextBox{
		F:      F,
		Glyphs: F.Font.Layout(nil, F.Size, text),
	}
}

// Extent implements the [Box] interface.
// Height and depth are never negative.  A glyph whose ink has no bound
// counts as reaching from the font's descent to its ascent.
func (obj *TextBox) Extent() *BoxExtent {
	g := obj.F.Font.GetGeometry()

	var width, height, depth float64
	for _, glyph := range obj.Glyphs.Seq {
		width += glyph.Advance

		var lly, ury float64
		switch {
		case g.IsBlank(glyph.GID):
			continue
		case g.IsUnbounded(glyph.GID):
			lly, ury = g.Descent, g.Ascent
		default:
			bbox := &g.GlyphExtents[glyph.GID]
			lly, ury = bbox.LLy, bbox.URy
		}
		depth = max(depth, -(lly*obj.F.Size + glyph.Rise))
		height = max(height, ury*obj.F.Size+glyph.Rise)
	}

	return &BoxExtent{
		Width:  width,
		Height: height,
		Depth:  depth,
	}
}

// Draw implements the [Box] interface.
func (obj *TextBox) Draw(page *builder.Builder, xPos, yPos float64) {
	page.TextBegin()
	page.TextSetFont(obj.F.Font, obj.F.Size)
	if obj.F.Color != nil {
		page.SetFillColor(obj.F.Color)
	} else {
		page.SetFillColor(color.Black)
	}
	page.TextFirstLine(xPos, yPos)
	page.TextShowGlyphs(obj.Glyphs)
	page.TextEnd()
}
