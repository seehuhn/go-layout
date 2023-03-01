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

// Glue returns a new "glue" box with the given natural length and
// stretchability.
func Glue(length float64, plus float64, plusLevel int, minus float64, minusLevel int) *Skip {
	return &Skip{
		Length: length,
		Plus:   stretchAmount{plus, plusLevel},
		Minus:  stretchAmount{minus, minusLevel},
	}
}

type Skip struct {
	Length float64
	Plus   stretchAmount
	Minus  stretchAmount
}

func (g *Skip) Clone() *Skip {
	return &Skip{
		Length: g.Length,
		Plus:   g.Plus,
		Minus:  g.Minus,
	}
}

func (g *Skip) minLength() float64 {
	if g == nil {
		return 0
	}
	if g.Minus.Order > 0 {
		return math.Inf(-1)
	}
	return g.Length - g.Minus.Val
}

func (g *Skip) maxLength() float64 {
	if g == nil {
		return 0
	}
	if g.Plus.Order > 0 {
		return math.Inf(+1)
	}
	return g.Length + g.Plus.Val
}

func (obj *Skip) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          obj.Length,
		Height:         obj.Length,
		WhiteSpaceOnly: true,
	}
}

func (obj *Skip) Draw(page *graphics.Page, xPos, yPos float64) {}

func (obj *Skip) Stretch() stretchAmount {
	return obj.Plus
}

func (obj *Skip) Shrink() stretchAmount {
	return obj.Minus
}

func (obj *Skip) Add(other *Skip) {
	if other == nil {
		return
	}
	obj.Length += other.Length
	obj.Plus.Add(other.Plus)
	obj.Minus.Add(other.Minus)
}

func (obj *Skip) addBoxHeightAndDepth(box Box) {
	ext := box.Extent()
	obj.Length += ext.Height + ext.Depth
	if stretch, ok := box.(stretcher); ok {
		obj.Plus.Add(stretch.Stretch())
	}
	if shrink, ok := box.(shrinker); ok {
		obj.Minus.Add(shrink.Shrink())
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
			res.Plus.Add(stretch.Stretch())
		}
		if shrink, ok := box.(shrinker); ok {
			res.Minus.Add(shrink.Shrink())
		}
	}
	return res
}

type stretcher interface {
	Stretch() stretchAmount
}

type shrinker interface {
	Shrink() stretchAmount
}

type stretchAmount struct {
	Val   float64
	Order int
}

func (s *stretchAmount) Add(other stretchAmount) {
	if other.Order > s.Order {
		s.Val = other.Val
		s.Order = other.Order
	} else if other.Order == s.Order {
		s.Val += other.Val
	}
}

func (s *stretchAmount) Minus(other stretchAmount) stretchAmount {
	if other.Order == s.Order {
		return stretchAmount{
			Val:   s.Val - other.Val,
			Order: s.Order,
		}
	} else if other.Order > s.Order {
		return stretchAmount{
			Val:   -other.Val,
			Order: other.Order,
		}
	} else { // other.Order < s.Order
		return stretchAmount{
			Val:   s.Val,
			Order: s.Order,
		}
	}
}
