package layout

import (
	"fmt"

	"seehuhn.de/go/dag"
)

func (e *Engine) EndParagraph() {
	// TODO(voss): check that no node has infinite shrinkability (since
	// otherwise the whole paragraph would fit into a single line)

	if len(e.VList) > 0 && e.ParSkip != nil {
		e.VList = append(e.VList, e.ParSkip)
	}

	parFillSkip := &hModeGlue{
		GlueBox: GlueBox{
			Plus: stretchAmount{Val: 1, Order: 1},
		},
		Text:    "\n",
		NoBreak: true,
	}
	e.HList = append(e.HList, parFillSkip)

	findPath := dag.ShortestPathDyn[*breakNode, int, float64]
	e2 := lineBreaker{e}
	start := &breakNode{}
	end := &breakNode{
		pos:         len(e.HList),
		lineNo:      0,
		prevBadness: -100,
	}
	breaks, err := findPath(e2, start, end)
	if err != nil {
		panic(err) // unreachable
	}

	curBreak := &breakNode{}
	for i, pos := range breaks {
		var currentLine []Box
		if e.LeftSkip != nil {
			currentLine = append(currentLine, e.LeftSkip)
		}
		for _, item := range e.HList[curBreak.pos:pos] {
			switch h := item.(type) {
			case *hModeGlue:
				glue := GlueBox(h.GlueBox)
				currentLine = append(currentLine, &glue)
			case *hModeText:
				currentLine = append(currentLine, &TextBox{
					F:      h.F,
					Glyphs: h.glyphs,
				})
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
			e.VList = append(e.VList, penalty)
		}

		lineBox := HBoxTo(e.TextWidth, currentLine...)
		e.VAddBox(lineBox)

		curBreak = e2.To(curBreak, pos)
	}

	e.HList = e.HList[:0]
}

func (g *Engine) getRelStretch(v *breakNode, e int) float64 {
	width := &GlueBox{}
	width.Add(g.LeftSkip)
	for pos := v.pos; pos < e; pos++ {
		switch h := g.HList[pos].(type) {
		case *hModeGlue:
			width.Add(&h.GlueBox)
		case *hModeText:
			width.Length += h.width
		default:
			panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
		}
	}
	width.Add(g.RightSkip)

	absStretch := g.TextWidth - width.Length

	var relStretch float64
	if absStretch >= 0 {
		if width.Plus.Order > 0 {
			absStretch = g.TextWidth
		}
		relStretch = absStretch / width.Plus.Val
	} else {
		if width.Minus.Order > 0 {
			panic("infinite shrinkage")
		}
		relStretch = absStretch / width.Minus.Val
	}
	return relStretch
}

type badnessClass int

const (
	badnessVeryLoose badnessClass = 2
	badnessLoose     badnessClass = 1
	badnessDecent    badnessClass = 0
	badnessTight     badnessClass = -1
)

func getBadnessClass(relStretch float64) badnessClass {
	switch {
	case relStretch >= 1:
		return badnessVeryLoose
	case relStretch >= 0.5:
		return badnessLoose
	case relStretch > -0.5:
		return badnessDecent
	default:
		return badnessTight
	}
}

type breakNode struct {
	pos         int
	lineNo      int
	prevBadness badnessClass
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

type lineBreaker struct {
	*Engine
}

// Edge returns the outgoing edges of the given vertex.
func (g lineBreaker) AppendEdges(res []int, v *breakNode) []int {
	totalWidth := g.LeftSkip.minLength() + g.RightSkip.minLength()
	glyphsSeen := false
	for pos := v.pos + 1; ; pos++ {
		if pos == len(g.HList) {
			res = append(res, pos)
			break
		}
		switch h := g.HList[pos].(type) {
		case *hModeGlue:
			if glyphsSeen && !h.NoBreak {
				res = append(res, pos)
				glyphsSeen = false
			}
			totalWidth += h.Length - h.Minus.Val
		case *hModeText:
			glyphsSeen = true
			totalWidth += h.width
		default:
			panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
		}
		if totalWidth > g.TextWidth && len(res) > 0 {
			break
		}
	}

	return res
}

// Length returns the "cost" of adding a line break at e.
func (e lineBreaker) Length(v *breakNode, pos int) float64 {
	q := e.getRelStretch(v, pos)

	cost := 0.0
	if q < -1 {
		cost += 1000
	} else {
		cost += 100 * q * q
	}
	thisBadness := getBadnessClass(q)
	if v.lineNo > 0 && abs(thisBadness-v.prevBadness) > 1 {
		cost += 10
	}
	return cost * cost
}

func abs(x badnessClass) badnessClass {
	if x < 0 {
		return -x
	}
	return x
}

// To returns the endpoint of a edge e starting at vertex v.
func (g lineBreaker) To(v *breakNode, pos int) *breakNode {
	pos0 := pos
	for pos < len(g.HList) && hDiscardible(g.HList[pos]) {
		pos++
	}
	return &breakNode{
		lineNo:      v.lineNo + 1,
		pos:         pos,
		prevBadness: getBadnessClass(g.getRelStretch(v, pos0)),
	}
}

func hDiscardible(h interface{}) bool {
	switch h.(type) {
	case *hModeGlue:
		return true
	case *hModeText:
		return false
	default:
		panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
	}
}
