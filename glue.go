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

// Glue returns a new "glue" box with the given natural length and
// stretchability.
func Glue(length float64, plus float64, plusLevel int, minus float64, minusLevel int) *GlueBox {
	return &GlueBox{
		Length: length,
		Plus:   stretchAmount{plus, plusLevel},
		Minus:  stretchAmount{minus, minusLevel},
	}
}

type GlueBox struct {
	Length float64
	Plus   stretchAmount
	Minus  stretchAmount
}

func (g *GlueBox) minLength() float64 {
	if g == nil {
		return 0
	}
	if g.Minus.Order > 0 {
		return math.Inf(-1)
	}
	return g.Length - g.Minus.Val
}

func (g *GlueBox) maxLength() float64 {
	if g == nil {
		return 0
	}
	if g.Plus.Order > 0 {
		return math.Inf(+1)
	}
	return g.Length + g.Plus.Val
}

func (obj *GlueBox) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          obj.Length,
		Height:         obj.Length,
		WhiteSpaceOnly: true,
	}
}

func (obj *GlueBox) Draw(page *graphics.Page, xPos, yPos float64) {}

func (obj *GlueBox) Stretch() stretchAmount {
	return obj.Plus
}

func (obj *GlueBox) Shrink() stretchAmount {
	return obj.Minus
}

func (obj *GlueBox) Add(other *GlueBox) {
	if other == nil {
		return
	}
	obj.Length += other.Length
	obj.Plus.Add(other.Plus)
	obj.Minus.Add(other.Minus)
}

func (obj *GlueBox) addBoxHeightAndDepth(box Box) {
	ext := box.Extent()
	obj.Length += ext.Height + ext.Depth
	if stretch, ok := box.(stretcher); ok {
		obj.Plus.Add(stretch.Stretch())
	}
	if shrink, ok := box.(shrinker); ok {
		obj.Minus.Add(shrink.Shrink())
	}
}

func equivalentHeightGlue(boxes []Box) *GlueBox {
	res := &GlueBox{}
	for _, box := range boxes {
		res.addBoxHeightAndDepth(box)
	}
	if len(boxes) > 0 {
		ext := boxes[0].Extent()
		res.Length -= ext.Depth
	}
	return res
}
