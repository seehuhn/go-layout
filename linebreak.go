package layout

import (
	"fmt"
)

func (e *Engine) EndParagraph() {
	// This must match the code in [Engine.VisualiseLineBreaks]

	// Gather the material for the line breaker.
	hList := e.HList
	// Add the final glue ...
	if e.ParFillSkip != nil {
		parFillSkip := &hModeGlue{
			GlueBox: *e.ParFillSkip,
			Text:    "\n",
		}
		hList = append(e.HList, parFillSkip)
	}
	// ... and a forced line break.
	hList = append(hList, &hModePenalty{Penalty: PenaltyForceBreak})

	e.HList = e.HList[:0]

	// Break the paragraph into lines.
	br := &knuthPlassLineBreaker{
		α: 100,
		γ: 100,
		ρ: 10,
		q: 0,
		lineWidth: func(lineNo int) float64 {
			return e.TextWidth
		},
		hList: hList,
	}
	breaks := br.Run()

	// Add the lines to the vertical list.
	if len(e.VList) > 0 && e.ParSkip != nil {
		e.VList = append(e.VList, e.ParSkip)
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
				currentLine = append(currentLine, &h.GlueBox)
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

		if i > 0 {
			penalty := e.InterLinePenalty
			if i == 1 {
				penalty += e.ClubPenalty
			}
			if i == len(breaks)-1 {
				penalty += e.WidowPenalty
			}
			e.VList = append(e.VList, Penalty(penalty))
		}

		prevPos = pos

		lineBox := HBoxTo(e.TextWidth, currentLine...)
		e.VAddBox(lineBox)
	}
}

// breaknode represents a possible line break.
type breakNode struct {
	pos         int          // the position of the break in the hModeMaterial list
	lineNo      int          // the number of the line before the break
	prevBadness fitnessClass // the badness of the line before the break
}

func (v *breakNode) Before(other *breakNode) bool {
	if v.pos != other.pos {
		return v.pos < other.pos
	}
	if v.lineNo != other.lineNo {
		return v.lineNo < other.lineNo
	}
	return v.prevBadness < other.prevBadness
}

// lineBreaker implements the dag.DynamicGraph[*breakNode, int, float64] interface.
type lineBreaker struct {
	*Engine
	hModeMaterial []interface{}
}

// AppendEdges appends the edges originating from v to res.
// The edges are represented by the positions of the next line break.
func (br lineBreaker) AppendEdges(res []int, v *breakNode) []int {
	totalWidth := br.LeftSkip.minLength() + br.RightSkip.minLength()
	hList := br.hModeMaterial
breakLoop:
	for pos := v.pos + 1; pos < len(hList); pos++ {
		// Line breaks can occur at a penalty, if the penalty is not +oo,
		// or at a glue, if the glue is immediately preceded by a box.
		switch h := hList[pos].(type) {
		case *hModePenalty:
			if h.Penalty < PenaltyPreventBreak {
				res = append(res, pos)
			}
			if h.Penalty == PenaltyForceBreak {
				break breakLoop
			}
			totalWidth += h.width
		case *hModeGlue:
			if _, prevIsBox := hList[pos-1].(*hModeBox); prevIsBox {
				res = append(res, pos)
			}
			totalWidth += h.Length - h.Minus.Val
		case *hModeBox:
			totalWidth += h.width
		}
		if totalWidth > br.TextWidth && len(res) > 0 {
			break breakLoop
		}
	}

	return res
}

// Length returns the "cost" of adding a line break at pos.
func (br lineBreaker) Length(v *breakNode, pos int) float64 {
	q := br.getRelStretch(v, pos)

	cost := 0.0
	if q < -1 {
		cost += 1000
	} else {
		cost += 100 * q * q
	}
	thisBadness := getFitnessClass(q)
	if v.lineNo > 0 && abs(thisBadness-v.prevBadness) > 1 {
		cost += 10
	}
	return cost * cost
}

// To returns the endpoint of a edge e starting at vertex v.
func (br lineBreaker) To(v *breakNode, pos int) *breakNode {
	pos0 := pos
	for pos < len(br.hModeMaterial) && hDiscardible(br.hModeMaterial[pos]) {
		pos++
	}
	return &breakNode{
		lineNo:      v.lineNo + 1,
		pos:         pos,
		prevBadness: getFitnessClass(br.getRelStretch(v, pos0)),
	}
}

func (br lineBreaker) getRelStretch(v *breakNode, end int) float64 {
	width := &GlueBox{}
	width.Add(br.LeftSkip)
	for pos := v.pos; pos < end; pos++ {
		switch h := br.hModeMaterial[pos].(type) {
		case *hModeBox:
			width.Length += h.width
		case *hModeGlue:
			width.Add(&h.GlueBox)
		case *hModePenalty:
			width.Length += h.width
		}
	}
	width.Add(br.RightSkip)

	absStretch := br.TextWidth - width.Length

	var relStretch float64
	if absStretch >= 0 { // loose line
		plusVal := width.Plus.Val
		if width.Plus.Order > 0 {
			plusVal = br.TextWidth
		}
		relStretch = absStretch / plusVal
	} else { // tight line
		if width.Minus.Order > 0 {
			panic("infinite shrinkage")
		}
		relStretch = absStretch / width.Minus.Val
	}
	return relStretch
}

func hDiscardible(h interface{}) bool {
	switch h.(type) {
	case *hModeBox:
		return false
	case *hModeGlue:
		return true
	case *hModePenalty:
		return true
	default:
		panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
	}
}
