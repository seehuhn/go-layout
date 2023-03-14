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

	"seehuhn.de/go/pdf/graphics"
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

// Rule returns a box with the given dimensions filled solid black.
func Rule(width, height, depth float64) Box {
	return &ruleBox{
		BoxExtent: BoxExtent{
			Width:  width,
			Height: height,
			Depth:  depth,
		},
	}
}

// A ruleBox is a rectangular region on the page filled solid black.
type ruleBox struct {
	BoxExtent
}

// Draw implements the Box interface.
func (obj *ruleBox) Draw(page *graphics.Page, xPos, yPos float64) {
	if obj.Width > 0 && obj.Depth+obj.Height > 0 {
		page.Rectangle(xPos, yPos-obj.Depth, obj.Width, obj.Depth+obj.Height)
		page.Fill()
	}
}

// Kern represents a fixed amount of space between boxes.
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

// Raise raises the box by the given amount.
func Raise(delta float64, box Box) Box {
	return raiseBox{
		Box:   box,
		delta: delta,
	}
}

type raiseBox struct {
	Box
	delta float64
}

func (obj raiseBox) Extent() *BoxExtent {
	ext := obj.Box.Extent()
	return &BoxExtent{
		Width:          ext.Width,
		Height:         ext.Height + obj.delta,
		Depth:          ext.Depth - obj.delta,
		WhiteSpaceOnly: ext.WhiteSpaceOnly,
	}
}

func (obj raiseBox) Draw(page *graphics.Page, xPos, yPos float64) {
	obj.Box.Draw(page, xPos, yPos+obj.delta)
}
