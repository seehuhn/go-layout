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

// VBox represents a Box which contains a column of sub-objects.
// The base point is the base point of the last box.
type VBox []Box

func (vbox VBox) Extent() *BoxExtent {
	height := 0.0
	depth := 0.0
	width := 0.0
	for i, child := range vbox {
		ext := child.Extent()
		if !ext.WhiteSpaceOnly && ext.Width > width {
			width = ext.Width
		}

		height += ext.Height
		if i < len(vbox)-1 {
			height += ext.Depth
		} else {
			depth = ext.Depth
		}
	}
	return &BoxExtent{
		Height: height,
		Depth:  depth,
		Width:  width,
	}
}

// Draw implements the Box interface.
func (vbox VBox) Draw(page *graphics.Page, xPos, yPos float64) {
	for i, child := range vbox {
		ext := child.Extent()
		if i > 0 {
			yPos -= ext.Height
		}
		if i < len(vbox)-1 {
			yPos -= ext.Depth
		}
	}

	for i, child := range vbox {
		ext := child.Extent()
		if i > 0 {
			yPos -= ext.Height
		}
		child.Draw(page, xPos, yPos)
		yPos -= ext.Depth
	}
}

// VTop represents a Box which contains a column of sub-objects.
// The base point is the base point of the first box.
type VTop []Box

func (vtop VTop) Extent() *BoxExtent {
	height := 0.0
	depth := 0.0
	width := 0.0
	for i, child := range vtop {
		childExt := child.Extent()
		if !childExt.WhiteSpaceOnly && childExt.Width > width {
			width = childExt.Width
		}

		if i == 0 {
			height = childExt.Height
		} else {
			depth += childExt.Height
		}
		depth += childExt.Depth
	}
	return &BoxExtent{
		Height: height,
		Depth:  depth,
		Width:  width,
	}
}

// Draw implements the Box interface.
func (vtop VTop) Draw(page *graphics.Page, xPos, yPos float64) {
	for i, child := range vtop {
		ext := child.Extent()
		if i > 0 {
			yPos -= ext.Height
		}
		child.Draw(page, xPos, yPos)
		yPos -= ext.Depth
	}
}
