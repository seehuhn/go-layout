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

	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/pdf/pagetree"
)

func (e *Engine) MakeVTop() Box {
	vtop := VTop(e.vList...)
	e.vList = e.vList[:0]
	return vtop
}

func (e *Engine) AppendPages(tree *pagetree.Writer, final bool) error {
	for len(e.vList) > 0 {
		if !final && (e.vTotalHeight() < 2*e.TextHeight || len(e.vList) < 2) {
			break
		}

		e.PageNumber++
		if e.PageNumber == e.DebugPageNumber {
			err := e.DebugPageBreak(tree)
			if err != nil {
				return err
			}
		}

		vbox := e.makePage()

		if len(e.records) > 0 {
			panic("unexpected records")
		}

		compress := &pdf.FilterInfo{Name: pdf.CompressFilter}
		contentRef := tree.Out.Alloc()
		stream, err := tree.Out.OpenStream(contentRef, nil, compress)
		if err != nil {
			return err
		}
		page := graphics.NewPage(stream)

		if e.BeforePageFunc != nil {
			err = e.BeforePageFunc(e.PageNumber, page)
			if err != nil {
				return err
			}
		}

		vbox.Draw(page, 72, 72) // TODO(voss): make the margins configurable

		if e.AfterPageFunc != nil {
			err = e.AfterPageFunc(e.PageNumber, page)
			if err != nil {
				return err
			}
		}

		err = stream.Close()
		if err != nil {
			return err
		}
		pageDict := pdf.Dict{
			"Type":     pdf.Name("Page"),
			"Contents": contentRef,
		}
		if page.Resources != nil {
			pageDict["Resources"] = pdf.AsDict(page.Resources)
		}

		pageRef := tree.Out.Alloc()
		if len(e.records) > 0 {
			for i, br := range e.records {
				bi := br.BoxInfo
				bi.PageRef = pageRef
				bi.PageNo = e.PageNumber
				for _, cb := range br.cb {
					cb(br.BoxInfo)
				}
				e.records[i] = nil
			}
			e.records = e.records[:0]
		}

		if e.AfterCloseFunc != nil {
			err = e.AfterCloseFunc(pageDict)
			if err != nil {
				return err
			}
		}

		err = tree.AppendPageRef(pageRef, pageDict)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) makePage() Box {
	height := e.TextHeight

	cand := e.vGetCandidates(height)
	bestPos := -1
	bestCost := math.Inf(+1)
	for _, c := range cand {
		cost := c.badness + float64(c.penalty)
		if cost <= bestCost {
			bestCost = cost
			bestPos = c.pos
		}
	}

	ext0 := e.vList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}

	var res []Box
	if topSkip > 0 {
		res = append(res, Kern(topSkip))
	}
	res = append(res, e.vList[:bestPos]...)
	if e.BottomGlue != nil {
		res = append(res, e.BottomGlue)
	}

	e.vList = e.vList[bestPos:]
	for len(e.vList) > 0 && vDiscardible(e.vList[0]) {
		e.vList = e.vList[1:]
	}

	return VBoxTo(height, res...)
}

type vCandidate struct {
	pos     int
	badness float64
	penalty penalty
}

func (e *Engine) vGetCandidates(height float64) []vCandidate {
	if len(e.vList) == 0 {
		return nil
	}

	ext0 := e.vList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}

	total := &Glue{
		Length: topSkip,
	}
	total.Add(e.BottomGlue)

	var res []vCandidate
	prevDept := 0.0
	for i := 0; i <= len(e.vList); i++ {
		var box Box
		if i < len(e.vList) {
			box = e.vList[i]
		}

		minHeight := total.minLength()
		maxHeight := total.maxLength()
		if minHeight > height && len(res) > 0 {
			break
		}

		penalty, isPenalty := box.(penalty)

		if e.vCanBreak(i) && !math.IsInf(float64(penalty), +1) {
			var badness float64

			if minHeight > height {
				// overfull vbox
				badness = math.Inf(+1)
			} else if maxHeight < height {
				// underfull vbox
				badness = math.Inf(+1)
			} else if math.Abs(total.Length-height) < 1e-6 {
				// perfect vbox
				badness = 0
			} else if total.Length < height {
				// need to stretch
				needStretch := height - total.Length
				canStrech := total.Stretch.Val
				if total.Stretch.Order > 0 {
					canStrech = height
				}
				badness = 100 * math.Pow(needStretch/canStrech, 3)
			} else {
				// need to shrink
				needShrink := total.Length - height
				canShrink := total.Shrink.Val
				// no infinite shrinkage should occur here
				badness = math.Min(1e4, 100*math.Pow(needShrink/canShrink, 3))
			}

			res = append(res, vCandidate{
				pos:     i,
				badness: badness,
				penalty: penalty,
			})

			if math.IsInf(float64(penalty), -1) {
				break
			}
		}

		if box != nil && !isPenalty {
			ext := box.Extent()
			total.Length += ext.Height + prevDept
			prevDept = ext.Depth

			if stretch, ok := box.(stretcher); ok {
				total.Stretch.IncrementBy(stretch.GetStretch())
			}
			if shrink, ok := box.(shrinker); ok {
				total.Shrink.IncrementBy(shrink.GetShrink())
			}
		}
	}

	return res
}

// vCanBreak returns true if the vertical list can be broken before the
// element at position pos.
func (e *Engine) vCanBreak(pos int) bool {
	if pos == len(e.vList) {
		return true
	} else if pos < 1 || pos > len(e.vList) {
		return false
	}

	switch obj := e.vList[pos].(type) {
	case *Glue: // before glue, if following a non-discardible item
		return !vDiscardible(e.vList[pos-1])
	case Kern: // before kern, if followed by glue
		if pos < len(e.vList)-1 {
			_, followedByGlue := e.vList[pos+1].(*Glue)
			return followedByGlue
		}
		return false
	case penalty: // at a penalty
		return float64(obj) < PenaltyPreventBreak
	default:
		return false
	}
}

func vDiscardible(box Box) bool {
	return box.Extent().WhiteSpaceOnly
}

// vTotalHeight returns the total height plus depth of the vertical list.
func (e *Engine) vTotalHeight() float64 {
	var height float64
	for _, box := range e.vList {
		ext := box.Extent()
		height += ext.Height + ext.Depth
	}
	return height
}
