// seehuhn.de/go/layout - a PDF layout engine
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
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
	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/graphics"
)

type recordPageLocation struct {
	Box
	e  *Engine
	cb []func(*BoxInfo)
}

func (r *recordPageLocation) Draw(page *graphics.Page, xPos, yPos float64) {
	ext := r.Extent()

	// TODO(voss): undo any coordinate transformations the user may have
	// applied, to get "default user space units".
	bbox := &pdf.Rectangle{
		LLx: xPos,
		LLy: yPos - ext.Depth,
		URx: xPos + ext.Width,
		URy: yPos + ext.Height,
	}
	bi := &BoxInfo{
		BBox: bbox,
	}
	br := &boxRecord{
		BoxInfo: bi,
		cb:      r.cb,
	}
	r.e.records = append(r.e.records, br)

	r.Box.Draw(page, xPos, yPos)
}

type boxRecord struct {
	*BoxInfo
	cb []func(*BoxInfo)
}
