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

func (ext BoxExtent) String() string {
	extra := ""
	if ext.WhiteSpaceOnly {
		extra = "W"
	}
	return fmt.Sprintf("%gx(%g%+g)%s", ext.Width, ext.Height, ext.Depth, extra)
}

// Extent allows for objects to embed a BoxExtent in order to implement part of
// the Box interface.
func (ext *BoxExtent) Extent() *BoxExtent {
	return ext
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

// Draw implements the [Box] interface.
func (obj *ruleBox) Draw(page *graphics.Page, xPos, yPos float64) {
	if obj.Width > 0 && obj.Depth+obj.Height > 0 {
		page.Rectangle(xPos, yPos-obj.Depth, obj.Width, obj.Depth+obj.Height)
		page.Fill()
	}
}

// Kern represents a fixed amount of space between boxes.
type Kern float64

// Extent implements the [Box] interface.
func (obj Kern) Extent() *BoxExtent {
	return &BoxExtent{
		Width:          float64(obj),
		Height:         float64(obj),
		WhiteSpaceOnly: true,
	}
}

// Draw implements the [Box] interface.
func (obj Kern) Draw(page *graphics.Page, xPos, yPos float64) {}

// Raise raises the box by the given amount.
func Raise(delta float64, box Box) Box {
	return raiseBox{
		box:   box,
		delta: delta,
	}
}

type raiseBox struct {
	box   Box
	delta float64
}

func (obj raiseBox) Extent() *BoxExtent {
	ext := obj.box.Extent()
	return &BoxExtent{
		Width:          ext.Width,
		Height:         ext.Height + obj.delta,
		Depth:          ext.Depth - obj.delta,
		WhiteSpaceOnly: ext.WhiteSpaceOnly,
	}
}

func (obj raiseBox) Draw(page *graphics.Page, xPos, yPos float64) {
	obj.box.Draw(page, xPos, yPos+obj.delta)
}

// HBox creates a new HBox
func HBox(children ...Box) Box {
	res := &hBox{
		Contents: children,
	}
	first := true
	for _, box := range children {
		ext := box.Extent()
		res.Width += ext.Width
		if ext.WhiteSpaceOnly {
			continue
		}

		if ext.Height > res.Height || first {
			res.Height = ext.Height
		}
		if ext.Depth > res.Depth || first {
			res.Depth = ext.Depth
		}
		first = false
	}
	return res
}

// HBoxTo creates a new HBox with the given width
func HBoxTo(width float64, contents ...Box) Box {
	res := &hBox{
		BoxExtent: BoxExtent{
			Width: width,
		},
		Contents: contents,
	}
	first := true
	for _, box := range contents {
		ext := box.Extent()
		if ext.WhiteSpaceOnly {
			continue
		}

		if ext.Height > res.Height || first {
			res.Height = ext.Height
		}
		if ext.Depth > res.Depth || first {
			res.Depth = ext.Depth
		}
		first = false
	}
	return res
}

// hBox represents a Box which contains a row of sub-objects.
type hBox struct {
	BoxExtent

	Contents []Box
}

// Draw implements the [Box] interface.
func (obj *hBox) Draw(page *graphics.Page, xPos, yPos float64) {
	xx := horizontalLayout(xPos, obj.Width, obj.Contents...)
	for i, box := range obj.Contents {
		box.Draw(page, xx[i], yPos)
	}
}

func horizontalLayout(xLeft, width float64, boxes ...Box) []float64 {
	x := xLeft
	xx := make([]float64, 0, len(boxes))
	total := totalWidthAndGlue(boxes)
	if total.Length < width-eps && total.Stretch.Val > 0 {
		// contents are too narrow, stretch all available glue
		q := (width - total.Length) / total.Stretch.Val
		for _, box := range boxes {
			xx = append(xx, x)
			x += box.Extent().Width + q*getStretch(box, total.Stretch.Order)
		}
	} else if total.Length > width+eps && total.Shrink.Val > 0 {
		// contents are too wide, shrink all available glue
		q := (total.Length - width) / total.Shrink.Val
		if total.Shrink.Order == 0 && q > 1 {
			// glue can't shrink beyond its minimum width
			q = 1
		}
		for _, box := range boxes {
			xx = append(xx, x)
			x += box.Extent().Width - q*getShrink(box, total.Shrink.Order)
		}
	} else {
		// lay out contents at their natural width
		for _, box := range boxes {
			xx = append(xx, x)
			x += box.Extent().Width
		}
	}
	return xx
}

// VTop creates a VBox with the given contents.
// The baseline of the box is at the baseline of the first child box.
func VTop(contents ...Box) Box {
	res := &vBox{
		Contents: contents,
	}
	for _, box := range contents {
		ext := box.Extent()
		if ext.Width > res.Width && !ext.WhiteSpaceOnly {
			res.Width = ext.Width
		}
		res.Depth += ext.Height + ext.Depth
	}
	if len(contents) > 0 {
		h := contents[0].Extent().Height
		res.Height += h
		res.Depth -= h
	}
	return res
}

// VBox creates a VBox with the given contents.
// The baseline of the box is at the baseline of the last child box.
func VBox(contents ...Box) Box {
	res := &vBox{
		Contents: contents,
	}
	for _, box := range contents {
		ext := box.Extent()
		if ext.Width > res.Width && !ext.WhiteSpaceOnly {
			res.Width = ext.Width
		}
		res.Height += ext.Height + ext.Depth
	}
	if len(contents) > 0 {
		d := contents[0].Extent().Depth
		res.Height -= d
		res.Depth += d
	}
	return res
}

// VBoxTo creates a VBox with the given height and contents.
// The baseline of the box is at the baseline of the last child box.
func VBoxTo(height float64, contents ...Box) Box {
	res := &vBox{
		BoxExtent: BoxExtent{
			Height: height,
		},
		Contents: contents,
	}
	if len(contents) > 0 {
		res.Depth = contents[len(contents)-1].Extent().Depth
	}
	for _, box := range contents {
		ext := box.Extent()
		if ext.Width > res.Width && !ext.WhiteSpaceOnly {
			res.Width = ext.Width
		}
	}
	return res
}

type vBox struct {
	BoxExtent
	Contents []Box
}

func (obj *vBox) Draw(page *graphics.Page, xPos, yPos float64) {
	yy := verticalLayout(yPos+obj.Height, obj.Height+obj.Depth, obj.Contents...)
	for i, box := range obj.Contents {
		box.Draw(page, xPos, yy[i])
	}
}

func verticalLayout(yTop, height float64, boxes ...Box) []float64 {
	y := yTop
	yy := make([]float64, 0, len(boxes))
	total := totalHeightAndGlue(boxes)
	if total.Length < height-eps && total.Stretch.Val > 0 {
		// contents are too short, stretch all available glue
		q := (height - total.Length) / total.Stretch.Val
		for _, box := range boxes {
			ext := box.Extent()
			y -= ext.Height + q*getStretch(box, total.Stretch.Order)
			yy = append(yy, y)
			y -= ext.Depth
		}
	} else if total.Length > height+eps && total.Shrink.Val > 0 {
		// contents are too tall, shrink all available glue
		q := (total.Length - height) / total.Shrink.Val
		if total.Shrink.Order == 0 && q > 1 {
			// glue can't shrink beyond its minimum width
			q = 1
		}
		for _, box := range boxes {
			ext := box.Extent()
			y -= ext.Height - q*getShrink(box, total.Shrink.Order)
			yy = append(yy, y)
			y -= ext.Depth
		}
	} else {
		// lay out contents at their natural height+depth
		for _, box := range boxes {
			ext := box.Extent()
			y -= ext.Height
			yy = append(yy, y)
			y -= ext.Depth
		}
	}
	return yy
}

const eps = 1e-3
