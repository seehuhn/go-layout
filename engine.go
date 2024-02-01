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
	"slices"
	"unicode"

	"seehuhn.de/go/sfnt/glyph"

	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics"
)

// A list of horizontal mode items can contain the following types:
//  - *hModeBox: a box which is not affected by line breaking.
//        The only property relevant for line breaking is the width.
//  - *Glue:
//  - *hModePenalty: an optional breakpoint

type hModeBox struct {
	Box
	width float64
}

type hModePenalty struct {
	Penalty float64
	width   float64
	flagged bool
}

// Engine is the main layout engine.
type Engine struct {
	PageSize *pdf.Rectangle

	TextWidth   float64
	ParIndent   *Glue
	LeftSkip    *Glue
	RightSkip   *Glue
	ParFillSkip *Glue

	TextHeight   float64
	TopSkip      float64 // TODO(voss): rename this, because it's not a "skip"?
	BottomGlue   *Glue
	BaseLineSkip float64 // TODO(voss): rename this, because it's not a "skip"?
	ParSkip      *Glue

	InterLinePenalty float64
	ClubPenalty      float64
	WidowPenalty     float64

	PageNumber     int
	BeforePageFunc func(int, *graphics.Writer) error
	AfterPageFunc  func(int, *graphics.Writer) error
	AfterCloseFunc func(pageDict pdf.Dict) error

	DebugPageNumber int

	hList      []interface{} // list of *hModeBox, *Glue, *hModePenalty
	afterPunct bool
	afterSpace bool

	vList     []Box
	prevDepth float64
	vRecordCB []func(*BoxInfo)
	records   []*boxRecord
}

type BoxInfo struct {
	PageRef pdf.Reference
	BBox    *pdf.Rectangle
	PageNo  int
}

func (e *Engine) HAddText(F *FontInfo, text string) {
	if len(e.hList) == 0 && e.ParIndent != nil {
		e.hList = append(e.hList, e.ParIndent)
	}

	var spaceGID glyph.ID
	var pdfSpaceWidth float64
	seq := F.Font.Layout(F.Size, " ")
	if len(seq.Seq) == 1 {
		spaceGID = seq.Seq[0].GID
		pdfSpaceWidth = seq.Seq[0].Advance
	} else {
		pdfSpaceWidth = F.Size / 4
	}

	spaceGlue := &Glue{
		Length:  pdfSpaceWidth,
		Stretch: glueAmount{Val: pdfSpaceWidth / 2},
		Shrink:  glueAmount{Val: pdfSpaceWidth / 3},
	}
	xSpaceGlue := &Glue{
		Length:  1.5 * pdfSpaceWidth,
		Stretch: glueAmount{Val: pdfSpaceWidth * 1.5},
		Shrink:  glueAmount{Val: pdfSpaceWidth},
	}

	var run []rune
	flushSpace := func() {
		if spaceGID != 0 {
			gg := []font.Glyph{
				{
					GID:     spaceGID,
					Text:    slices.Clone(run),
					Advance: 0, // no width for space glyph, since we add glue below
				},
			}

			var prevText *TextBox
			if k := len(e.hList); k > 0 {
				if box, ok := e.hList[k-1].(*hModeBox); ok {
					prevText, _ = box.Box.(*TextBox)
				}
			}
			if prevText != nil {
				prevText.Glyphs.Seq = append(prevText.Glyphs.Seq, gg...)
			} else {
				box := &TextBox{F: F, Glyphs: &font.GlyphSeq{Seq: gg}}
				e.hList = append(e.hList, &hModeBox{Box: box})
			}
		}

		// if len(run) == 1 && run[0] == 0x200B { // ZERO WIDTH SPACE
		// 	e.hList = append(e.hList, &hModePenalty{})
		// }

		if e.afterPunct {
			e.hList = append(e.hList, xSpaceGlue)
		} else {
			e.hList = append(e.hList, spaceGlue)
		}

		run = run[:0]
	}

	flushRunes := func() {
		gg := F.Font.Layout(F.Size, string(run))
		box := &TextBox{F: F, Glyphs: gg}
		e.hList = append(e.hList, &hModeBox{
			Box:   box,
			width: gg.TotalLength(),
		})
		run = run[:0]
	}

	for _, r := range text {
		if unicode.IsSpace(r) &&
			r != 0x00A0 && // NO-BREAK SPACE
			r != 0x2007 && // FIGURE SPACE
			r != 0x202F { // NARROW NO-BREAK SPACE

			if !e.afterSpace && len(run) > 0 {
				flushRunes()
			}

			run = append(run, r)
			e.afterSpace = true
		} else {
			if e.afterSpace && len(run) > 0 {
				flushSpace()
			}

			run = append(run, r)
			e.afterSpace = false
		}
		e.afterPunct = r == '.' || r == '!' || r == '?'
	}
	if len(run) > 0 {
		if e.afterSpace {
			flushSpace()
		} else {
			flushRunes()
		}
	}
}

// HAddGlue adds a glue item to the horizontal mode list.
func (e *Engine) HAddGlue(g *Glue) {
	e.hList = append(e.hList, g)
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

func (obj penalty) Draw(page *graphics.Writer, xPos, yPos float64) {
	// pass
}

func (e *Engine) VRecordNextBox(cb func(*BoxInfo)) {
	e.vRecordCB = append(e.vRecordCB, cb)
}
