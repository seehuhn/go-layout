package layout

import (
	"fmt"
	"math"
)

type lineBreakGraph struct {
	hlist     []interface{}
	textWidth float64
	leftSkip  *glue
	rightSkip *glue
}

type breakNode struct {
	lineNo         int
	pos            int
	prevRelStretch float64
}

// Edge returns the outgoing edges of the given vertex.
func (g *lineBreakGraph) Edges(v *breakNode) []int {
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

func (g *lineBreakGraph) getRelStretch(v *breakNode, e int) float64 {
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
func (g *lineBreakGraph) Length(v *breakNode, e int) float64 {
	q := g.getRelStretch(v, e)

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
func (g *lineBreakGraph) To(v *breakNode, e int) *breakNode {
	pos := e
	for pos < len(g.hlist) && discardible(g.hlist[pos]) {
		pos++
	}
	return &breakNode{
		lineNo:         v.lineNo + 1,
		pos:            pos,
		prevRelStretch: g.getRelStretch(v, e),
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
