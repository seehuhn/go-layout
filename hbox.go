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
	"seehuhn.de/go/pdf/graphics"
)

// hBox represents a Box which contains a row of sub-objects.
type hBox struct {
	BoxExtent

	Contents []Box
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

// Draw implements the Box interface.
func (obj *hBox) Draw(page *graphics.Page, xPos, yPos float64) {
	xx := horizontalLayout(xPos, obj.Width, obj.Contents...)
	for i, box := range obj.Contents {
		box.Draw(page, xx[i], yPos)
	}
}

func horizontalLayout(x, width float64, boxes ...Box) []float64 {
	xx := make([]float64, 0, len(boxes))
	total := measureWidth(boxes)
	if total.Length < width-1e-3 && total.Plus.Val > 0 {
		// contents are too narrow, stretch all glue
		q := (width - total.Length) / total.Plus.Val
		for _, box := range boxes {
			xx = append(xx, x)
			x += box.Extent().Width + q*getStretch(box, total.Plus.Order)
		}
	} else if total.Length > width+1e-3 && total.Minus.Val > 0 {
		// contents are too wide, shrink all glue
		q := (total.Length - width) / total.Minus.Val
		if total.Minus.Order == 0 && q > 1 {
			// glue can't shrink beyond its minimum width
			q = 1
		}
		for _, box := range boxes {
			xx = append(xx, x)
			x += box.Extent().Width - q*getShrink(box, total.Minus.Order)
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

func getStretch(box Box, order int) float64 {
	stretch, ok := box.(stretcher)
	if !ok {
		return 0
	}
	info := stretch.Stretch()
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
	info := shrink.Shrink()
	if info.Order != order {
		return 0
	}
	return info.Val
}
