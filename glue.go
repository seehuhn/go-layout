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
	"math"

	"seehuhn.de/go/pdf/graphics"
)

// Skip returns a new "glue" box with the given natural length and
// stretchability.
func Skip(length float64, plus float64, plusLevel int, minus float64, minusLevel int) *Glue {
	return &Glue{
		Length:  length,
		Stretch: glueAmount{plus, plusLevel},
		Shrink:  glueAmount{minus, minusLevel},
	}
}

type Glue struct {
	Length  float64
	Stretch glueAmount
	Shrink  glueAmount
}

func (g *Glue) isInfiniteShrink() bool {
	shrink := g.Shrink
	return shrink.Val != 0 && shrink.Order > 0
}

func (g *Glue) Plus(other *Glue) *Glue {
	if other == nil {
		return g.Clone()
	}
	return &Glue{
		Length:  g.Length + other.Length,
		Stretch: g.Stretch.Plus(other.Stretch),
		Shrink:  g.Shrink.Plus(other.Shrink),
	}
}

// SetMinus sets g to a-b.
func (g *Glue) SetMinus(a, b *Glue) {
	g.Length = a.Length - b.Length
	if a.Stretch.Order > b.Stretch.Order {
		g.Stretch = a.Stretch
	} else if a.Stretch.Order < b.Stretch.Order {
		g.Stretch = glueAmount{-b.Stretch.Val, b.Stretch.Order}
	} else {
		g.Stretch = glueAmount{a.Stretch.Val - b.Stretch.Val, a.Stretch.Order}
	}
	if a.Shrink.Order > b.Shrink.Order {
		g.Shrink = a.Shrink
	} else if a.Shrink.Order < b.Shrink.Order {
		g.Shrink = glueAmount{-b.Shrink.Val, b.Shrink.Order}
	} else {
		g.Shrink = glueAmount{a.Shrink.Val - b.Shrink.Val, a.Shrink.Order}
	}
}

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

func (g *Glue) Clone() *Glue {
	return &Glue{
		Length:  g.Length,
		Stretch: g.Stretch,
		Shrink:  g.Shrink,
	}
}

func (g *Glue) minLength() float64 {
	if g == nil {
		return 0
	}
	if g.Shrink.Order > 0 {
		return math.Inf(-1)
	}
	return g.Length - g.Shrink.Val
}

func (g *Glue) maxLength() float64 {
	if g == nil {
		return 0
	}
	if g.Stretch.Order > 0 {
		return math.Inf(+1)
	}
	return g.Length + g.Stretch.Val
}

func (g *Glue) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          g.Length,
		Height:         g.Length,
		WhiteSpaceOnly: true,
	}
}

func (g *Glue) Draw(page *graphics.Writer, xPos, yPos float64) {}

func (g *Glue) GetStretch() glueAmount {
	return g.Stretch
}

func (g *Glue) GetShrink() glueAmount {
	return g.Shrink
}

func (g *Glue) Add(other *Glue) {
	if other == nil {
		return
	}
	g.Length += other.Length
	g.Stretch.IncrementBy(other.Stretch)
	g.Shrink.IncrementBy(other.Shrink)
}

func (g *Glue) addBoxHeightAndDepth(box Box) {
	ext := box.Extent()
	g.Length += ext.Height + ext.Depth
	if stretch, ok := box.(stretcher); ok {
		g.Stretch.IncrementBy(stretch.GetStretch())
	}
	if shrink, ok := box.(shrinker); ok {
		g.Shrink.IncrementBy(shrink.GetShrink())
	}
}

// MeasureHeight returns the total height, depth and stretchability of the given boxes.
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
			res.Stretch.IncrementBy(stretch.GetStretch())
		}
		if shrink, ok := box.(shrinker); ok {
			res.Shrink.IncrementBy(shrink.GetShrink())
		}
	}
	return res
}

type glueAmount struct {
	Val   float64
	Order int
}

func (s *glueAmount) IncrementBy(other glueAmount) {
	if other.Order > s.Order {
		s.Val = other.Val
		s.Order = other.Order
	} else if other.Order == s.Order {
		s.Val += other.Val
	}
}

func (s *glueAmount) Plus(other glueAmount) glueAmount {
	if other.Order == s.Order {
		return glueAmount{
			Val:   s.Val + other.Val,
			Order: s.Order,
		}
	} else if other.Order > s.Order {
		return glueAmount{
			Val:   other.Val,
			Order: other.Order,
		}
	} else { // other.Order < s.Order
		return glueAmount{
			Val:   s.Val,
			Order: s.Order,
		}
	}
}

func (s *glueAmount) Minus(other glueAmount) glueAmount {
	if other.Order == s.Order {
		return glueAmount{
			Val:   s.Val - other.Val,
			Order: s.Order,
		}
	} else if other.Order > s.Order {
		return glueAmount{
			Val:   -other.Val,
			Order: other.Order,
		}
	} else { // other.Order < s.Order
		return glueAmount{
			Val:   s.Val,
			Order: s.Order,
		}
	}
}

type stretcher interface {
	GetStretch() glueAmount
}

type shrinker interface {
	GetShrink() glueAmount
}

func getStretch(box Box, order int) float64 {
	stretch, ok := box.(stretcher)
	if !ok {
		return 0
	}
	info := stretch.GetStretch()
	if info.Order != order {
		return 0
	}
	return info.Val
}

func getShrink(box Box, order int) float64 {
	shrink, ok := box.(shrinker)
	if !ok {
		return 0
	}
	info := shrink.GetShrink()
	if info.Order != order {
		return 0
	}
	return info.Val
}
