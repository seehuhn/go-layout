package layout

import (
	"fmt"
	"math"

	"golang.org/x/exp/slices"
)

type knuthPlassLineBreaker struct {
	α float64 // extra demerits for consecutive flagged breaks
	γ float64 // extra demerits for badness classes that are more than 1 apart
	ρ float64 // upper bound on the adjustment ratios
	q int     // looseness parameter

	lineWidth func(lineNo int) float64

	hList []interface{}

	active []*knuthNode
	total  *GlueBox
}

type knuthNode struct {
	pos           int
	line          int
	fitness       fitnessClass
	total         *GlueBox
	totalDemerits float64
	previous      *knuthNode
}

func (br *knuthPlassLineBreaker) Run() []int {
	br.active = append(br.active[:0], &knuthNode{
		total: &GlueBox{},
	})
	br.total = &GlueBox{}

	var feasibleBreaks []int
	for b := 0; b < len(br.hList); b++ {
		if br.IsValidBreakpoint(b) {
			feasibleBreaks = feasibleBreaks[:0]

			pb := br.Penalty(b)

			aIdx := 0
			for aIdx < len(br.active) { // loop over all line numbers
				var Ac [4]*knuthNode
				Dc := [4]float64{math.Inf(+1), math.Inf(+1), math.Inf(+1), math.Inf(+1)}
				D := math.Inf(+1)
				for { // loop over all nodes for a given line number
					a := br.active[aIdx]

					r := br.AdjustmentRatio(a, b)
					if r < -1 || pb == PenaltyForceBreak {
						// deactivate node a
						copy(br.active[aIdx:], br.active[aIdx+1:])
						br.active[len(br.active)-1] = nil
						br.active = br.active[:len(br.active)-1]
					} else {
						aIdx++
					}

					if r >= -1 && r <= br.ρ {
						c := getFitnessClass(r)
						d := br.computeDemerits(r, pb, a, b, c)

						if d < Dc[c+1] {
							Ac[c+1] = a
							Dc[c+1] = d
							if d < D {
								D = d
							}
						}
					}

					if aIdx >= len(br.active) || br.active[aIdx].line > a.line {
						break
					}
				}

				if D < math.Inf(+1) {
					// insert new active nodes for breaks from Ac to b
					afterBTotal := br.total.Clone()
				afterBLoop:
					for i := b; i < len(br.hList); i++ {
						switch h := br.hList[i].(type) {
						case *hModeBox:
							break afterBLoop
						case *hModeGlue:
							afterBTotal.Add(&h.GlueBox)
						case *hModePenalty:
							if i > b && h.Penalty == PenaltyForceBreak {
								break afterBLoop
							}
						}
					}

					for c := fitnessClass(-1); c <= 2; c++ {
						if Dc[c+1] > D+br.γ {
							continue
						}
						s := &knuthNode{
							pos:           b,
							line:          Ac[c+1].line + 1,
							fitness:       c,
							total:         afterBTotal,
							totalDemerits: Dc[c+1],
							previous:      Ac[c+1],
						}
						br.active = slices.Insert(br.active, aIdx, s)
						aIdx++
					}
				}
			}
			if len(br.active) == 0 {
				panic("no feasible solution")
			}
		}

		switch h := br.hList[b].(type) {
		case *hModeBox:
			br.total.Length += h.width
		case *hModeGlue:
			br.total.Add(&h.GlueBox)
		}
	}

	// Choose the active node with the fewest total demerits.
	bestA := 0
	for a := 1; a < len(br.active); a++ {
		if br.active[a].totalDemerits < br.active[bestA].totalDemerits {
			bestA = a
		}
	}
	k := br.active[bestA].line

	if br.q != 0 { // choose the appropriate active node
		s := 0
		var d float64
		for aIdx := 0; aIdx < len(br.active); aIdx++ {
			a := br.active[aIdx]
			delta := a.line - k
			if br.q <= delta && delta < s || s < delta && delta <= br.q {
				s = delta
				d = a.totalDemerits
				bestA = aIdx
			} else if delta == s && a.totalDemerits < d {
				d = a.totalDemerits
				bestA = aIdx
			}
		}
		k = br.active[bestA].line
	}

	// use the chosen node to determine the optimal breakpoint sequence
	breaks := make([]int, k)
	for a := br.active[bestA]; a.line > 0; a = a.previous {
		breaks[a.line-1] = a.pos
	}
	return breaks
}

func (br *knuthPlassLineBreaker) computeDemerits(r float64, pb float64, a *knuthNode, b int, c fitnessClass) float64 {
	var d float64
	r3 := 1 + 100*math.Pow(math.Abs(r), 3)
	if pb >= 0 {
		d = math.Pow(r3+float64(pb), 2)
	} else if pb != PenaltyForceBreak {
		d = math.Pow(r3, 2) - math.Pow(float64(pb), 2)
	} else {
		d = math.Pow(r3, 2)
	}
	if br.IsFlagged(a.pos) && br.IsFlagged(b) {
		d += br.α
	}
	if abs(c-a.fitness) > 1 {
		d += br.γ
	}
	return d
}

func (br *knuthPlassLineBreaker) IsValidBreakpoint(pos int) bool {
	switch h := br.hList[pos].(type) {
	case *hModePenalty:
		return h.Penalty < PenaltyPreventBreak
	case *hModeGlue:
		_, prevIsBox := br.hList[pos-1].(*hModeBox)
		return prevIsBox
	default:
		return false
	}
}

func (br *knuthPlassLineBreaker) Penalty(pos int) float64 {
	if p, isPenalty := br.hList[pos].(*hModePenalty); isPenalty {
		return p.Penalty
	}
	return 0
}

func (br *knuthPlassLineBreaker) IsFlagged(pos int) bool {
	if p, isPenalty := br.hList[pos].(*hModePenalty); isPenalty {
		return p.flagged
	}
	return false
}

func (br *knuthPlassLineBreaker) AdjustmentRatio(a *knuthNode, b int) float64 {
	L := br.total.Length - a.total.Length
	if p, isPenalty := br.hList[b].(*hModePenalty); isPenalty {
		L += p.width
	}
	available := br.lineWidth(a.line)
	if L < available { // loose line
		stretch := br.total.Plus.Minus(a.total.Plus)
		if stretch.Order > 0 {
			return 0
		}
		if stretch.Val > 0 {
			return (available - L) / stretch.Val
		}
		return math.Inf(+1)
	} else if L > available { // tight line
		shrink := br.total.Minus.Minus(a.total.Minus)
		if shrink.Order > 0 {
			// TODO(voss): what to do here?
			return 0
		}
		if shrink.Val > 0 {
			return (available - L) / shrink.Val
		}
		return math.Inf(+1)
	}
	return 0
}

type fitnessClass int

const (
	badnessVeryLoose fitnessClass = 2
	badnessLoose     fitnessClass = 1
	badnessDecent    fitnessClass = 0
	badnessTight     fitnessClass = -1
)

func (b fitnessClass) String() string {
	switch b {
	case badnessVeryLoose:
		return "very loose"
	case badnessLoose:
		return "loose"
	case badnessDecent:
		return "decent"
	case badnessTight:
		return "tight"
	default:
		return fmt.Sprintf("badnessClass(%d)", b)
	}
}

func getFitnessClass(r float64) fitnessClass {
	var c fitnessClass
	if r < -0.5 {
		c = badnessTight
	} else if r <= 0.5 {
		c = badnessDecent
	} else if r <= 1.0 {
		c = badnessLoose
	} else {
		c = badnessVeryLoose
	}
	return c
}

func abs(x fitnessClass) fitnessClass {
	if x < 0 {
		return -x
	}
	return x
}

var (
	PenaltyPreventBreak = math.Inf(+1)
	PenaltyForceBreak   = math.Inf(-1)
)
