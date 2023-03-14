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
