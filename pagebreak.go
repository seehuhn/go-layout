package layout

import (
	"fmt"
	"math"

	"seehuhn.de/go/pdf/pages"
)

func (e *Engine) AppendPage(tree *pages.Tree, height float64) error {
	vbox := e.MakePage(height)
	if vbox == nil {
		return nil
	}

	e.PageNumber++

	page, err := pages.AppendPage(tree)
	if err != nil {
		return nil
	}

	vbox.Draw(page.Page, 72, 72) // TODO(voss): make the margins configurable

	if e.AfterPageFunc != nil {
		err = e.AfterPageFunc(e, page.Page)
		if err != nil {
			return err
		}
	}

	_, err = page.Close()
	if err != nil {
		return err
	}

	return nil
}

func (e *Engine) MakePage(height float64) Box {
	if len(e.VList) == 0 {
		return nil
	}

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

	if bestPos < 0 {
		for _, c := range e.VList {
			fmt.Printf("%T: %v\n", c, c)
		}
	}

	ext0 := e.VList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}

	var res []Box
	if topSkip > 0 {
		res = append(res, Kern(topSkip))
	}
	res = append(res, e.VList[:bestPos]...)
	if e.BottomGlue != nil {
		res = append(res, e.BottomGlue)
	}

	e.VList = e.VList[bestPos:]
	for len(e.VList) > 0 && vDiscardible(e.VList[0]) {
		e.VList = e.VList[1:]
	}

	return VBox2To(height, res...)
}

type vCandidate struct {
	pos     int
	badness float64
	penalty Penalty
}

func (e *Engine) vGetCandidates(height float64) []vCandidate {
	if len(e.VList) == 0 {
		return nil
	}

	ext0 := e.VList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}

	total := &Skip{
		Length: topSkip,
	}
	total.Add(e.BottomGlue)

	var res []vCandidate
	prevDept := 0.0
	for i := 0; i <= len(e.VList); i++ {
		var box Box
		if i < len(e.VList) {
			box = e.VList[i]
		}

		minHeight := total.minLength()
		maxHeight := total.maxLength()
		if minHeight > height && len(res) > 0 {
			break
		}

		penalty, isPenalty := box.(Penalty)

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
	if pos < 1 || pos > len(e.VList) {
		return false
	}
	if pos == len(e.VList) {
		return true
	}

	switch obj := e.VList[pos].(type) {
	case *Skip: // before glue, if following a non-discardible item
		return !vDiscardible(e.VList[pos-1])
	case Kern: // before kern, if followed by glue
		if pos < len(e.VList)-1 {
			_, followedByGlue := e.VList[pos+1].(*Skip)
			return followedByGlue
		}
		return false
	case Penalty: // at a penalty
		return float64(obj) < PenaltyPreventBreak
	default:
		return false
	}
}

func vDiscardible(box Box) bool {
	return box.Extent().WhiteSpaceOnly
}
