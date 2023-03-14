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
	"fmt"
)

func (e *Engine) EndParagraph() {
	// This must match the code in [Engine.VisualiseLineBreaks]

	// Gather the material for the line breaker.
	hList := e.hList
	// Add the final glue ...
	if e.ParFillSkip != nil {
		hList = append(hList, &hModePenalty{Penalty: PenaltyPreventBreak})
		parFillSkip := &hModeGlue{
			Glue: *e.ParFillSkip,
			Text: "\n",
		}
		hList = append(hList, parFillSkip)
	}
	// ... and a forced line break.
	hList = append(hList, &hModePenalty{Penalty: PenaltyForceBreak})

	e.hList = e.hList[:0]
	e.afterPunct = false
	e.afterSpace = false

	lineWidth := &Glue{Length: e.TextWidth}
	lineWidth = lineWidth.Minus(e.LeftSkip).Minus(e.RightSkip)

	// Break the paragraph into lines.
	br := &knuthPlassLineBreaker{
		α: 100,
		γ: 100,
		ρ: 1000,
		q: 0,
		lineWidth: func(lineNo int) *Glue {
			return lineWidth
		},
		hList: hList,
	}
	breaks := br.Run()

	// Add the lines to the vertical list.
	if len(e.vList) > 0 && e.ParSkip != nil {
		e.vList = append(e.vList, e.ParSkip)
	}
	prevPos := 0
	for i, pos := range breaks {
		var currentLine []Box
		if e.LeftSkip != nil {
			currentLine = append(currentLine, e.LeftSkip)
		}
		for _, item := range hList[prevPos:pos] {
			switch h := item.(type) {
			case *hModeGlue:
				currentLine = append(currentLine, &h.Glue)
			case *hModeBox:
				currentLine = append(currentLine, h.Box)
			case *hModePenalty:
				// TODO(voss)
			default:
				panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
			}
		}
		if e.RightSkip != nil {
			currentLine = append(currentLine, e.RightSkip)
		}

	skipDiscardible:
		for prevPos = pos; prevPos < len(hList); prevPos++ {
			switch h := br.hList[prevPos].(type) {
			case *hModeBox:
				break skipDiscardible
			case *hModePenalty:
				if prevPos > pos && h.Penalty == PenaltyForceBreak {
					break skipDiscardible
				}
			}
		}

		if i > 0 {
			p := e.InterLinePenalty
			if i == 1 {
				p += e.ClubPenalty
			}
			if i == len(breaks)-1 {
				p += e.WidowPenalty
			}
			e.VAddPenalty(p)
		}

		lineBox := HBoxTo(e.TextWidth, currentLine...)
		e.VAddBox(lineBox)
	}
}
