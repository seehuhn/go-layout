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
	"math"
	"unicode"

	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/sfnt/funit"
)

// A list of horizontal mode items can contain the following types:
//  - *hModeBox: a box which is not affected by line breaking.
//        The only property relevant for line breaking is the width.
//  - *hModeGlue:
//  - *hModePenalty: an optional breakpoint

type hModeBox struct {
	Box
	width float64
}

type hModeGlue struct {
	Glue
	Text string
}

type hModePenalty struct {
	Penalty float64
	width   float64
	flagged bool
}

type Engine struct {
	TextWidth   float64
	ParIndent   *Glue
	LeftSkip    *Glue
	RightSkip   *Glue
	ParFillSkip *Glue

	TextHeight   float64
	TopSkip      float64 // TODO(voss): rename this, because it's not a glue?
	BottomGlue   *Glue
	BaseLineSkip float64
	ParSkip      *Glue

	InterLinePenalty float64
	ClubPenalty      float64
	WidowPenalty     float64

	PageNumber     int
	BeforePageFunc func(int, *graphics.Page) error
	AfterPageFunc  func(int, *graphics.Page) error
	AfterCloseFunc func(pageDict pdf.Dict) error

	DebugPageNumber int

	hList      []interface{} // list of hModeBox, hModeGlue, hModePenalty
	afterPunct bool
	afterSpace bool

	vList     []Box
	prevDepth float64
	vRecordCB []func(*BoxInfo)
	records   []*boxRecord
}

type BoxInfo struct {
	PageRef *pdf.Reference
	BBox    *pdf.Rectangle
	PageNo  int
}

func (e *Engine) HAddText(F *FontInfo, text string) {
	if len(e.hList) == 0 && e.ParIndent != nil {
		e.hList = append(e.hList, &hModeGlue{Glue: *e.ParIndent})
	}
	geom := F.Font.GetGeometry()
	space := F.Font.Layout(" ", F.Size)
	var spaceWidth funit.Int
	if len(space) == 1 && space[0].Gid != 0 {
		spaceWidth = funit.Int(space[0].Advance)
	} else {
		space = nil
		spaceWidth = funit.Int(geom.UnitsPerEm / 4)
	}
	pdfSpaceWidth := geom.ToPDF(F.Size, spaceWidth)

	spaceGlue := &hModeGlue{
		Glue: Glue{
			Length:  pdfSpaceWidth,
			Stretch: glueAmount{Val: pdfSpaceWidth / 2},
			Shrink:  glueAmount{Val: pdfSpaceWidth / 3},
		},
		Text: " ",
	}
	xSpaceGlue := &hModeGlue{
		Glue: Glue{
			Length:  1.5 * pdfSpaceWidth,
			Stretch: glueAmount{Val: pdfSpaceWidth * 1.5},
			Shrink:  glueAmount{Val: pdfSpaceWidth},
		},
		Text: " ",
	}
	addSpace := func() {
		if e.afterPunct {
			e.hList = append(e.hList, xSpaceGlue)
		} else {
			e.hList = append(e.hList, spaceGlue)
		}
	}
	addRunes := func(rr []rune) {
		gg := F.Font.Layout(string(rr), F.Size)
		box := &TextBox{
			F:      F,
			Glyphs: gg,
		}
		e.hList = append(e.hList, &hModeBox{
			Box:   box,
			width: geom.ToPDF(F.Size, gg.AdvanceWidth()),
		})
	}

	var run []rune
	for _, r := range text {
		if r == 0x200B { // ZERO WIDTH SPACE
			if len(run) > 0 {
				addRunes(run)
				run = run[:0]
			}
			e.hList = append(e.hList, &hModePenalty{})
		} else if unicode.IsSpace(r) &&
			r != 0x00A0 && // NO-BREAK SPACE
			r != 0x2007 && // FIGURE SPACE
			r != 0x202F { // NARROW NO-BREAK SPACE
			if len(run) > 0 {
				addRunes(run)
				run = run[:0]
			}
			if !e.afterSpace {
				addSpace()
			}
			e.afterSpace = true
			e.afterPunct = false
		} else {
			run = append(run, r)
			e.afterSpace = false
			e.afterPunct = r == '.' || r == '!' || r == '?'
		}
	}
	if len(run) > 0 {
		addRunes(run)
	}
}

// HAddGlue adds a glue item to the horizontal mode list.
func (e *Engine) HAddGlue(g *Glue) {
	e.hList = append(e.hList, &hModeGlue{Glue: *g})
}

func (e *Engine) VAddGlue(g *Glue) {
	// TODO(voss): check for infinite shrinkability
	e.vList = append(e.vList, g)
}

func (e *Engine) VAddBox(b Box) {
	ext := b.Extent()
	if len(e.vList) > 0 {
		gap := ext.Height + e.prevDepth
		if gap+eps < e.BaseLineSkip {
			e.vList = append(e.vList, Kern(e.BaseLineSkip-gap))
		}
	}
	if len(e.vRecordCB) > 0 {
		e.vList = append(e.vList, &recordPageLocation{
			Box: b,
			e:   e,
			cb:  e.vRecordCB,
		})
		e.vRecordCB = nil
	} else {
		e.vList = append(e.vList, b)
	}
	e.prevDepth = ext.Depth
}

func (e *Engine) VAddPenalty(p float64) {
	e.vList = append(e.vList, penalty(p))
}

var (
	PenaltyPreventBreak = math.Inf(+1)
	PenaltyForceBreak   = math.Inf(-1)
)

type penalty float64

func (obj penalty) Extent() *BoxExtent {
	return &BoxExtent{WhiteSpaceOnly: true}
}

func (obj penalty) Draw(page *graphics.Page, xPos, yPos float64) {
	// pass
}

func (e *Engine) VRecordNextBox(cb func(*BoxInfo)) {
	e.vRecordCB = append(e.vRecordCB, cb)
}
