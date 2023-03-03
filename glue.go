// seehuhn.de/go/pdf - a library for reading and writing PDF files
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

type stretcher interface {
	GetStretch() glueAmount
}

type shrinker interface {
	GetShrink() glueAmount
}

// Glue returns a new "glue" box with the given natural length and
// stretchability.
func Glue(length float64, plus float64, plusLevel int, minus float64, minusLevel int) *Skip {
	return &Skip{
		Length:  length,
		Stretch: glueAmount{plus, plusLevel},
		Shrink:  glueAmount{minus, minusLevel},
	}
}

type Skip struct {
	Length  float64
	Stretch glueAmount
	Shrink  glueAmount
}

func (g *Skip) Plus(other *Skip) *Skip {
	if other == nil {
		return g.Clone()
	}
	return &Skip{
		Length:  g.Length + other.Length,
		Stretch: g.Stretch.Plus(other.Stretch),
		Shrink:  g.Shrink.Plus(other.Shrink),
	}
}

// SetMinus sets g to a-b.
func (g *Skip) SetMinus(a, b *Skip) {
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

func (g *Skip) Minus(other *Skip) *Skip {
	if other == nil {
		return g.Clone()
	}
	return &Skip{
		Length:  g.Length - other.Length,
		Stretch: g.Stretch.Minus(other.Stretch),
		Shrink:  g.Shrink.Minus(other.Shrink),
	}
}

func (g *Skip) Clone() *Skip {
	return &Skip{
		Length:  g.Length,
		Stretch: g.Stretch,
		Shrink:  g.Shrink,
	}
}

func (g *Skip) minLength() float64 {
	if g == nil {
		return 0
	}
	if g.Shrink.Order > 0 {
		return math.Inf(-1)
	}
	return g.Length - g.Shrink.Val
}

func (g *Skip) maxLength() float64 {
	if g == nil {
		return 0
	}
	if g.Stretch.Order > 0 {
		return math.Inf(+1)
	}
	return g.Length + g.Stretch.Val
}

func (obj *Skip) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          obj.Length,
		Height:         obj.Length,
		WhiteSpaceOnly: true,
	}
}

func (obj *Skip) Draw(page *graphics.Page, xPos, yPos float64) {}

func (obj *Skip) GetStretch() glueAmount {
	return obj.Stretch
}

func (obj *Skip) GetShrink() glueAmount {
	return obj.Shrink
}

func (obj *Skip) Add(other *Skip) {
	if other == nil {
		return
	}
	obj.Length += other.Length
	obj.Stretch.IncrementBy(other.Stretch)
	obj.Shrink.IncrementBy(other.Shrink)
}

func (obj *Skip) addBoxHeightAndDepth(box Box) {
	ext := box.Extent()
	obj.Length += ext.Height + ext.Depth
	if stretch, ok := box.(stretcher); ok {
		obj.Stretch.IncrementBy(stretch.GetStretch())
	}
	if shrink, ok := box.(shrinker); ok {
		obj.Shrink.IncrementBy(shrink.GetShrink())
	}
}

func measureHeight(boxes []Box) *Skip {
	res := &Skip{}
	for _, box := range boxes {
		res.addBoxHeightAndDepth(box)
	}
	if len(boxes) > 0 {
		ext := boxes[len(boxes)-1].Extent()
		res.Length -= ext.Depth
	}
	return res
}

func measureWidth(boxes []Box) *Skip {
	res := &Skip{}
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
