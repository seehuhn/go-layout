package layout

import (
	"fmt"
	"math"

	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/pdf/pages"
)

func (e *Engine) AppendPage(tree *pages.Tree, height float64) error {
	vbox := e.MakePage(height)
	if vbox == nil {
		return nil
	}

	e.PageNumber++

	page, err := graphics.AppendPage(tree)
	if err != nil {
		return nil
	}

	vbox.Draw(page, 72, 72) // TODO(voss): make the margins configurable

	if e.AfterPageFunc != nil {
		err = e.AfterPageFunc(e, page)
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

	ext0 := e.VList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}

	total := &GlueBox{
		Length: topSkip,
	}
	total.Add(e.BottomGlue)
	bestPos := -1
	bestCost := math.Inf(+1)
	prevDept := 0.0
	for i := 0; i <= len(e.VList); i++ {
		var penalty Penalty

		if i < len(e.VList) {
			box := e.VList[i]
			ext := box.Extent()
			total.Length += ext.Height + prevDept
			if stretch, ok := box.(stretcher); ok {
				total.Plus.Add(stretch.Stretch())
			}
			if shrink, ok := box.(shrinker); ok {
				total.Minus.Add(shrink.Shrink())
			}
			prevDept = ext.Depth

			if p, isPenalty := box.(Penalty); isPenalty {
				penalty = p
			}
		}

		if !e.vCanBreak(i) || penalty == PenaltyPreventBreak {
			continue
		}

		var cost float64
		if d := total.minLength() - height; d > 0 {
			// overfull vbox
			cost = 100 + d/height
		} else if d := total.maxLength() - height; d < 0 {
			// underfull vbox
			cost = 10 - d/height
		} else {
			d := (total.Length - height) / height
			cost = d * d
		}
		fmt.Printf("%10.3f %g\n", cost, penalty)
		if cost < bestCost {
			bestCost = cost
			bestPos = i
		}
	}
	fmt.Println()

	var res []Box
	if topSkip > 0 {
		res = append(res, Kern(topSkip))
	}
	res = append(res, e.VList[:bestPos]...)
	if e.BottomGlue != nil {
		res = append(res, e.BottomGlue)
	}
	e.VList = e.VList[bestPos:]

	return VBox2To(height, res...)
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
	case *GlueBox: // before glue, if following a non-discardible item
		return !vDiscardible(e.VList[pos-1])
	case Kern: // before kern, if followed by glue
		if pos < len(e.VList)-1 {
			_, followedByGlue := e.VList[pos+1].(*GlueBox)
			return followedByGlue
		}
		return false
	case Penalty: // at a penalty
		return obj > PenaltyPreventBreak
	default:
		return false
	}
}

func vDiscardible(box Box) bool {
	return box.Extent().WhiteSpaceOnly
}
