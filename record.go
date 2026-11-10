// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import (
	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/graphics/content/builder"
)

type recordPageLocation struct {
	Box
	e  *Engine
	cb []func(*BoxInfo)
}

func (r *recordPageLocation) Draw(page *builder.Builder, xPos, yPos float64) {
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
