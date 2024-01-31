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
	"seehuhn.de/go/sfnt/glyph"
)

// TextBox represents a typeset string of characters as a Box object.
// The text is typeset using a single font and size.
type TextBox struct {
	F      *FontInfo
	Glyphs glyph.Seq
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
		Glyphs: F.Font.Layout(text),
	}
}

// Extent implements the [Box] interface
func (obj *TextBox) Extent() *BoxExtent {
	font := obj.F.Font
	geom := font.GetGeometry()
	q := obj.F.Size / float64(geom.UnitsPerEm)

	width := 0.0
	height := math.Inf(-1)
	depth := math.Inf(-1)
	for _, glyph := range obj.Glyphs {
		width += glyph.Advance.AsFloat(q)

		thisDepth := geom.Descent * obj.F.Size
		thisHeight := geom.Ascent * obj.F.Size
		if geom.GlyphExtents != nil {
			bbox := &geom.GlyphExtents[glyph.GID]
			if bbox.IsZero() {
				continue
			}
			thisDepth = -(bbox.LLy + glyph.YOffset).AsFloat(q)
			thisHeight = (bbox.URy + glyph.YOffset).AsFloat(q)
		}
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
		page.SetFillColor(obj.F.Color)
	} else {
		page.SetFillColor(color.Default)
	}
	page.TextFirstLine(xPos, yPos)
	page.TextShowGlyphsOld(obj.Glyphs)
	page.TextEnd()
}
