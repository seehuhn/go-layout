package layout

import (
	"fmt"
	"math"

	"seehuhn.de/go/dijkstra"
)

func (e *Engine) EndParagraph() {
	// TODO(voss): check that no node has infinite shrinkability (since
	// otherwise the whole paragraph would fit into a single line)

	parFillSkip := &hModeGlue{
		glue: glue{
			Plus: stretchAmount{Val: 1, Level: 1},
		},
		Text:    "\n",
		NoBreak: true,
	}
	e.hlist = append(e.hlist, parFillSkip)

	findPath := dijkstra.ShortestPathSet[*breakNode, int, float64]
	start := &breakNode{}
	e2 := lineBreaker{e}
	breaks, err := findPath(e2, start, func(v *breakNode) bool {
		return v.pos == len(e.hlist)
	})
	if err != nil {
		panic(err) // unreachable
	}

	first := true
	var prevDepth float64

	curBreak := &breakNode{}
	for _, pos := range breaks {
		var lineBoxes []Box
		if e.leftSkip != nil {
			leftSkip := glueBox(*e.leftSkip)
			lineBoxes = append(lineBoxes, &leftSkip)
		}
		for _, item := range e.hlist[curBreak.pos:pos] {
			switch h := item.(type) {
			case *hModeGlue:
				glue := glueBox(h.glue)
				lineBoxes = append(lineBoxes, &glue)
			case *hModeText:
				lineBoxes = append(lineBoxes, &TextBox{
					Font:     h.font,
					FontSize: h.fontSize,
					Glyphs:   h.glyphs,
				})
			default:
				panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
			}
		}
		if e.rightSkip != nil {
			rightSkip := glueBox(*e.rightSkip)
			lineBoxes = append(lineBoxes, &rightSkip)
		}
		line := HBoxTo(e.textWidth, lineBoxes...)
		ext := line.Extent()
		if first {
			first = false
		} else {
			gap := ext.Height + prevDepth
			if gap+0.1 < e.baseLineSkip {
				e.vlist = append(e.vlist, Kern(e.baseLineSkip-gap))
			}
		}
		prevDepth = ext.Depth

		e.vlist = append(e.vlist, line)

		curBreak = e2.To(curBreak, pos)
	}

	e.hlist = e.hlist[:0]
}

type breakNode struct {
	lineNo         int
	pos            int
	prevRelStretch float64
}

type lineBreaker struct {
	*Engine
}

// Edge returns the outgoing edges of the given vertex.
func (g lineBreaker) Edges(v *breakNode) []int {
	var res []int

	totalWidth := g.leftSkip.minWidth() + g.rightSkip.minWidth()
	glyphsSeen := false
	for pos := v.pos + 1; ; pos++ {
		if pos == len(g.hlist) {
			res = append(res, pos)
			break
		}
		switch h := g.hlist[pos].(type) {
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
		if totalWidth > g.textWidth && len(res) > 0 {
			break
		}
	}

	return res
}

func (g *Engine) getRelStretch(v *breakNode, e int) float64 {
	width := &glue{}
	width = width.Add(g.leftSkip)
	for pos := v.pos; pos < e; pos++ {
		switch h := g.hlist[pos].(type) {
		case *hModeGlue:
			width = width.Add(&h.glue)
		case *hModeText:
			width.Length += h.width
		default:
			panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
		}
	}
	width = width.Add(g.rightSkip)

	absStretch := g.textWidth - width.Length

	var relStretch float64
	if absStretch >= 0 {
		if width.Plus.Level == 0 {
			relStretch = absStretch / width.Plus.Val
		}
	} else {
		if width.Minus.Level > 0 {
			panic("infinite shrinkage")
		}
		relStretch = absStretch / width.Minus.Val
	}
	return relStretch
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
	if v.lineNo > 0 && math.Abs(q-v.prevRelStretch) > 0.1 {
		cost += 10
	}
	return cost * cost
}

// To returns the endpoint of a edge e starting at vertex v.
func (g lineBreaker) To(v *breakNode, pos int) *breakNode {
	pos0 := pos
	for pos < len(g.hlist) && discardible(g.hlist[pos]) {
		pos++
	}
	return &breakNode{
		lineNo:         v.lineNo + 1,
		pos:            pos,
		prevRelStretch: g.getRelStretch(v, pos0),
	}
}

func discardible(h interface{}) bool {
	switch h.(type) {
	case *hModeGlue:
		return true
	case *hModeText:
		return false
	default:
		panic(fmt.Sprintf("unexpected type %T in horizontal mode list", h))
	}
}
