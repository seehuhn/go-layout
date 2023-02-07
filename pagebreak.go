package layout

import "math"

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
	bestPos := 0
	bestCost := math.Inf(+1)
	prevDept := 0.0
	for i, box := range e.VList {
		ext := box.Extent()
		total.Length += ext.Height + prevDept
		if stretch, ok := box.(stretcher); ok {
			total.Plus.Add(stretch.Stretch())
		}
		if shrink, ok := box.(shrinker); ok {
			total.Minus.Add(shrink.Shrink())
		}
		prevDept = ext.Depth

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
		if cost < bestCost {
			bestCost = cost
			bestPos = i + 1
		}
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

	return VBox2To(height, res...)
}
