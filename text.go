// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import (
	"math"

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

// Extent implements the [Box] interface
func (obj *TextBox) Extent() *BoxExtent {
	font := obj.F.Font
	geom := font.GetGeometry()

	width := 0.0
	height := math.Inf(-1)
	depth := math.Inf(-1)
	for _, glyph := range obj.Glyphs.Seq {
		width += glyph.Advance

		bbox := &geom.GlyphExtents[glyph.GID]
		if bbox.IsZero() {
			continue
		}
		thisDepth := -(bbox.LLy*obj.F.Size + glyph.Rise)
		thisHeight := bbox.URy*obj.F.Size + glyph.Rise

		if thisDepth > depth {
			depth = thisDepth
		}
		if thisHeight > height {
			height = thisHeight
		}
	}

	return &BoxExtent{
		Width:  width,
		Height: height,
		Depth:  depth,
	}
}

// Draw implements the [Box] interface.
func (obj *TextBox) Draw(page *builder.Builder, xPos, yPos float64) {
	font := obj.F.Font

	page.TextBegin()
	page.TextSetFont(font, obj.F.Size)
	if obj.F.Color != nil {
		page.SetFillColor(obj.F.Color)
	} else {
		page.SetFillColor(color.Black)
	}
	page.TextFirstLine(xPos, yPos)
	page.TextShowGlyphs(obj.Glyphs)
	page.TextEnd()
}
