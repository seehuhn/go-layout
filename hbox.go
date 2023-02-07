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

// hBox represents a Box which contains a row of sub-objects.
type hBox struct {
	BoxExtent

	Contents []Box
}

// HBox creates a new HBox
func HBox(children ...Box) Box {
	hbox := &hBox{
		BoxExtent: BoxExtent{
			Height: math.Inf(-1),
			Depth:  math.Inf(-1),
		},
		Contents: children,
	}
	for _, box := range children {
		ext := box.Extent()
		hbox.Width += ext.Width
		if ext.Height > hbox.Height && !ext.WhiteSpaceOnly {
			hbox.Height = ext.Height
		}
		if ext.Depth > hbox.Depth && !ext.WhiteSpaceOnly {
			hbox.Depth = ext.Depth
		}
	}
	return hbox
}

// HBoxTo creates a new HBox with the given width
func HBoxTo(width float64, contents ...Box) Box {
	res := &hBox{
		BoxExtent: BoxExtent{
			Width:  width,
			Height: math.Inf(-1),
			Depth:  math.Inf(-1),
		},
		Contents: contents,
	}
	for _, box := range contents {
		ext := box.Extent()
		if ext.Height > res.Height && !ext.WhiteSpaceOnly {
			res.Height = ext.Height
		}
		if ext.Depth > res.Depth && !ext.WhiteSpaceOnly {
			res.Depth = ext.Depth
		}
	}
	return res
}

func (obj *hBox) stretchTo(width float64) {
	naturalWidth := 0.0
	for _, child := range obj.Contents {
		ext := child.Extent()
		naturalWidth += ext.Width
	}

	if naturalWidth < width-1e-3 {
		level := -1
		var ii []int
		stretchTotal := 0.0
		for i, child := range obj.Contents {
			stretch, ok := child.(stretcher)
			if !ok {
				continue
			}
			info := stretch.Stretch()

			if info.Level > level {
				level = info.Level
				ii = nil
				stretchTotal = 0
			}
			ii = append(ii, i)
			stretchTotal += info.Val
		}

		if stretchTotal > 0 {
			q := (width - naturalWidth) / stretchTotal
			if level == 0 && q > 1 {
				// glue can't shrink beyond its minimum width
				q = 1
			}
			for _, i := range ii {
				child := obj.Contents[i]
				ext := child.Extent()
				amount := ext.Width + child.(stretcher).Stretch().Val*q
				obj.Contents[i] = Kern(amount)
			}
		}
	} else if naturalWidth > width+1e-3 {
		level := -1
		var ii []int
		shrinkTotal := 0.0
		for i, child := range obj.Contents {
			shrink, ok := child.(shrinker)
			if !ok {
				continue
			}
			info := shrink.Shrink()

			if info.Level > level {
				level = info.Level
				ii = nil
				shrinkTotal = 0
			}
			ii = append(ii, i)
			shrinkTotal += info.Val
		}

		if shrinkTotal > 0 {
			q := (naturalWidth - width) / shrinkTotal
			// glue can stretch beyond its natural width, if needed
			for _, i := range ii {
				child := obj.Contents[i]
				ext := child.Extent()
				amount := ext.Width - child.(shrinker).Shrink().Val*q
				obj.Contents[i] = Kern(amount)
			}
		}
	}
}

// Draw implements the Box interface.
func (obj *hBox) Draw(page *graphics.Page, xPos, yPos float64) {
	obj.stretchTo(obj.Width)

	x := xPos
	for _, child := range obj.Contents {
		ext := child.Extent()
		child.Draw(page, x, yPos)
		x += ext.Width
	}
}
