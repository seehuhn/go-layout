// seehuhn.de/go/layout - a PDF layout engine
// Copyright (C) 2021  Jochen Voss <voss@seehuhn.de>
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
	"fmt"
	"math"

	"seehuhn.de/go/pdf/color"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/sfnt/glyph"
)

// Box represents marks on a page within a rectangular area of known size.
type Box interface {
	Extent() *BoxExtent
	Draw(page *graphics.Page, xPos, yPos float64)
}

// BoxExtent gives the dimensions of a Box.
type BoxExtent struct {
	Width, Height, Depth float64
	WhiteSpaceOnly       bool
}

func (obj BoxExtent) String() string {
	extra := ""
	if obj.WhiteSpaceOnly {
		extra = "*"
	}
	return fmt.Sprintf("%gx%g%+g%s", obj.Width, obj.Height, obj.Depth, extra)
}

// Extent allows for objects to embed a BoxExtent in order to implement part of
// the Box interface.
func (obj *BoxExtent) Extent() *BoxExtent {
	return obj
}

// A RuleBox is a solidly filled rectangular region on the page.
type RuleBox struct {
	BoxExtent
}

// Rule returns a new rule box (a box filled solid black).
func Rule(width, height, depth float64) Box {
	return &RuleBox{
		BoxExtent: BoxExtent{
			Width:  width,
			Height: height,
			Depth:  depth,
		},
	}
}

// Draw implements the Box interface.
func (obj *RuleBox) Draw(page *graphics.Page, xPos, yPos float64) {
	if obj.Width > 0 && obj.Depth+obj.Height > 0 {
		page.Rectangle(xPos, yPos-obj.Depth, obj.Width, obj.Depth+obj.Height)
		page.Fill()
	}
}

// Kern represents a fixed amount of space.
type Kern float64

// Extent implements the Box interface.
func (obj Kern) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          float64(obj),
		Height:         float64(obj),
		WhiteSpaceOnly: true,
	}
}

// Draw implements the Box interface.
func (obj Kern) Draw(page *graphics.Page, xPos, yPos float64) {}

// TextBox represents a typeset string of characters as a Box object.
// The text is typeset using a single font and font size.
type TextBox struct {
	F      *FontInfo
	Glyphs glyph.Seq
}

type FontInfo struct {
	Font  font.Embedded
	Size  float64
	Color color.Color
}

// Text returns a new Text object.
func Text(F *FontInfo, text string) *TextBox {
	return &TextBox{
		F:      F,
		Glyphs: F.Font.Layout(text, F.Size),
	}
}

// Extent implements the Box interface
func (obj *TextBox) Extent() *BoxExtent {
	font := obj.F.Font
	geom := font.GetGeometry()
	q := obj.F.Size / float64(geom.UnitsPerEm)

	width := 0.0
	height := math.Inf(-1)
	depth := math.Inf(-1)
	for _, glyph := range obj.Glyphs {
		width += glyph.Advance.AsFloat(q)

		thisDepth := geom.Descent.AsFloat(q)
		thisHeight := geom.Ascent.AsFloat(q)
		if geom.GlyphExtents != nil {
			bbox := &geom.GlyphExtents[glyph.Gid]
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

// Draw implements the Box interface.
func (obj *TextBox) Draw(page *graphics.Page, xPos, yPos float64) {
	font := obj.F.Font

	page.BeginText()
	page.SetFont(font, obj.F.Size)
	if obj.F.Color != nil {
		page.SetFillColor(obj.F.Color)
	} else {
		page.SetFillColor(color.Default)
	}
	page.StartLine(xPos, yPos)
	page.ShowGlyphs(obj.Glyphs)
	page.EndText()
}

type raiseBox struct {
	Box
	delta float64
}

func (obj raiseBox) Extent() *BoxExtent {
	extent := obj.Box.Extent()
	extent.Height += obj.delta
	extent.Depth -= obj.delta
	return extent
}

func (obj raiseBox) Draw(page *graphics.Page, xPos, yPos float64) {
	obj.Box.Draw(page, xPos, yPos+obj.delta)
}

// Raise raises the box by the given amount.
func Raise(delta float64, box Box) Box {
	return raiseBox{
		Box:   box,
		delta: delta,
	}
}
