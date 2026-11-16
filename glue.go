// Copyright 2021 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import (
	"math"

	"seehuhn.de/go/pdf/graphics/content/builder"
)

// Skip returns new glue with the given natural length, stretchability and
// shrinkability.  The orders give the order of infinity of plus and minus,
// as described for [GlueAmount]; valid orders are 0 to 3.
func Skip(length, plus float64, plusOrder int, minus float64, minusOrder int) *Glue {
	g := &Glue{Length: length}
	g.Stretch[plusOrder] = plus
	g.Shrink[minusOrder] = minus
	return g
}

// Glue represents elastic space that can stretch or shrink to fill available
// room. The Length field gives the natural size, while Stretch and Shrink
// specify how much the glue can expand or contract.
//
// Glue is a [Box]: it has no ink, and its extent in both directions is
// its natural length.
type Glue struct {
	Length  float64
	Stretch GlueAmount
	Shrink  GlueAmount
}

// SetMinus sets g to a-b.  The receiver g may be the same as a or b.
func (g *Glue) SetMinus(a, b *Glue) {
	g.Length = a.Length - b.Length
	g.Stretch = a.Stretch.Minus(b.Stretch)
	g.Shrink = a.Shrink.Minus(b.Shrink)
}

// Minus returns g minus other as a new Glue.
// A nil argument is treated as zero glue.
func (g *Glue) Minus(other *Glue) *Glue {
	if other == nil {
		return g.Clone()
	}
	return &Glue{
		Length:  g.Length - other.Length,
		Stretch: g.Stretch.Minus(other.Stretch),
		Shrink:  g.Shrink.Minus(other.Shrink),
	}
}

// Clone returns a copy of g.
func (g *Glue) Clone() *Glue {
	return &Glue{
		Length:  g.Length,
		Stretch: g.Stretch,
		Shrink:  g.Shrink,
	}
}

func (g *Glue) minLength() float64 {
	shrink, order := g.Shrink.Dominant()
	if order > 0 {
		return math.Inf(-1)
	}
	return g.Length - shrink
}

func (g *Glue) maxLength() float64 {
	stretch, order := g.Stretch.Dominant()
	if order > 0 {
		return math.Inf(+1)
	}
	return g.Length + stretch
}

// Extent returns the extent of g, using the natural length as both width
// and height.
func (g *Glue) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          g.Length,
		Height:         g.Length,
		WhiteSpaceOnly: true,
	}
}

// Draw implements the [Box] interface.  Glue draws nothing.
func (g *Glue) Draw(page *builder.Builder, xPos, yPos float64) {}

// GetStretch returns the stretchability of g.
func (g *Glue) GetStretch() GlueAmount {
	return g.Stretch
}

// GetShrink returns the shrinkability of g.
func (g *Glue) GetShrink() GlueAmount {
	return g.Shrink
}

// Add increases g by other.
// A nil argument is treated as zero glue.
func (g *Glue) Add(other *Glue) {
	if other == nil {
		return
	}
	g.Length += other.Length
	g.Stretch.Add(other.Stretch)
	g.Shrink.Add(other.Shrink)
}

func (g *Glue) addBoxHeightAndDepth(box Box) {
	ext := box.Extent()
	g.Length += ext.Height + ext.Depth
	if stretch, ok := box.(stretcher); ok {
		g.Stretch.Add(stretch.GetStretch())
	}
	if shrink, ok := box.(shrinker); ok {
		g.Shrink.Add(shrink.GetShrink())
	}
}

// totalHeightAndGlue returns the total height, depth, stretchability and
// shrinkability of the given boxes.
func totalHeightAndGlue(boxes []Box) *Glue {
	res := &Glue{}
	for _, box := range boxes {
		res.addBoxHeightAndDepth(box)
	}
	return res
}

func totalWidthAndGlue(boxes []Box) *Glue {
	res := &Glue{}
	for _, box := range boxes {
		ext := box.Extent()
		res.Length += ext.Width
		if stretch, ok := box.(stretcher); ok {
			res.Stretch.Add(stretch.GetStretch())
		}
		if shrink, ok := box.(shrinker); ok {
			res.Shrink.Add(shrink.GetShrink())
		}
	}
	return res
}

// GlueAmount is an amount of stretch or shrink, split by order of infinity.
// Element 0 holds the finite part, and elements 1, 2 and 3 hold the parts of
// first, second and third order of infinity (TeX's fil, fill and filll).
// When glue is set, only the highest order with a non-zero amount takes
// part; see [GlueAmount.Dominant].
//
// Infinite amounts are resolved to multiples of 1/65536 when they are added
// or subtracted, so that amounts which cancel leave exactly zero.
type GlueAmount [4]float64

// infiniteUnits is the number of representable steps per unit of infinite
// glue.  Sums of multiples of 1/infiniteUnits are exact in a float64, which
// makes the comparison with zero in [GlueAmount.Dominant] reliable.
const infiniteUnits = 1 << 16

// resolved returns a with the infinite amounts rounded to the resolution
// used in sums.
func (a GlueAmount) resolved() GlueAmount {
	for i := 1; i < len(a); i++ {
		a[i] = math.Round(a[i]*infiniteUnits) / infiniteUnits
	}
	return a
}

// Add adds other to a.
func (a *GlueAmount) Add(other GlueAmount) {
	*a = a.resolved()
	other = other.resolved()
	for i := range a {
		a[i] += other[i]
	}
}

// Minus returns a minus other.
func (a GlueAmount) Minus(other GlueAmount) GlueAmount {
	a = a.resolved()
	other = other.resolved()
	for i := range a {
		a[i] -= other[i]
	}
	return a
}

// Dominant returns the amount of the highest order with a non-zero amount,
// together with that order.  If all amounts are zero, it returns 0, 0.
func (a GlueAmount) Dominant() (val float64, order int) {
	for order := len(a) - 1; order > 0; order-- {
		if a[order] != 0 {
			return a[order], order
		}
	}
	return a[0], 0
}

type stretcher interface {
	GetStretch() GlueAmount
}

type shrinker interface {
	GetShrink() GlueAmount
}

func getStretch(box Box, order int) float64 {
	stretch, ok := box.(stretcher)
	if !ok {
		return 0
	}
	return stretch.GetStretch().resolved()[order]
}

func getShrink(box Box, order int) float64 {
	shrink, ok := box.(shrinker)
	if !ok {
		return 0
	}
	return shrink.GetShrink().resolved()[order]
}
