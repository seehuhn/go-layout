// seehuhn.de/go/layout - a PDF layout engine
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package layout

import (
	"math"

	"seehuhn.de/go/pdf/color"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics"
)

// TextBox represents a typeset string of characters as a Box object.
// The text is typeset using a single font and size.
type TextBox struct {
	F      *FontInfo
	Glyphs *font.GlyphSeq
}

type FontInfo struct {
	Font  font.Layouter
	Size  float64
	Color color.Color
}

// Text returns a new [TextBox] object.
func Text(F *FontInfo, text string) *TextBox {
	return &TextBox{
		F:      F,
		Glyphs: F.Font.Layout(F.Size, text),
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
		thisHeight := (bbox.URy*obj.F.Size + glyph.Rise)

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
func (obj *TextBox) Draw(page *graphics.Writer, xPos, yPos float64) {
	font := obj.F.Font

	page.TextStart()
	page.TextSetFont(font, obj.F.Size)
	if obj.F.Color != nil {
		page.SetFillColorOld(obj.F.Color)
	} else {
		page.SetFillColorOld(color.Default)
	}
	page.TextFirstLine(xPos, yPos)
	page.TextShowGlyphs(obj.Glyphs)
	page.TextEnd()
}
