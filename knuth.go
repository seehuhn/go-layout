package layout

import (
	"fmt"
	"math"
)

type knuthPlassLineBreaker struct {
	α float64 // extra demerits for consecutive flagged breaks
	γ float64 // extra demerits for badness classes that are more than 1 apart
	ρ float64 // upper bound on the adjustment ratios
	q int     // looseness parameter (try to in-/decrease number of lines by q)

	lineWidth func(lineNo int) *Skip

	hList []interface{}

	active []*knuthPlassNode
	total  *Skip

	scratch *Skip
}

type knuthPlassNode struct {
	pos           int
	line          int
	fitness       fitnessClass
	total         *Skip
	totalDemerits float64
	previous      *knuthPlassNode
}

func (br *knuthPlassLineBreaker) Run() []int {
	start := &knuthPlassNode{total: &Skip{}}
	br.active = append(br.active[:0], start)
	br.total = &Skip{}
	br.scratch = &Skip{}

	for b := 0; b < len(br.hList); b++ {
		if isValidBreakpoint(br.hList, b) {
			pb := br.Penalty(b)

			aIdx := 0
			for aIdx < len(br.active) { // loop over all line numbers
				var Ac [4]*knuthPlassNode
				Dc := [4]float64{math.Inf(+1), math.Inf(+1), math.Inf(+1), math.Inf(+1)}
				D := math.Inf(+1)

				// loop over all active nodes which share the same line number
				for {
					a := br.active[aIdx]

					r := br.AdjustmentRatio(a, b)
					if r < -1 || pb == PenaltyForceBreak {
						// remove a from the active list
						copy(br.active[aIdx:], br.active[aIdx+1:])
						br.active[len(br.active)-1] = nil
						br.active = br.active[:len(br.active)-1]
					} else {
						// leave a in the active list, skip to next node
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
					totalAfterB := br.total.Clone()
				afterBLoop:
					for i := b; i < len(br.hList); i++ {
						switch h := br.hList[i].(type) {
						case *hModeBox:
							break afterBLoop
						case *hModeGlue:
							totalAfterB.Add(&h.Skip)
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
						s := &knuthPlassNode{
							pos:           b,
							line:          Ac[c+1].line + 1,
							fitness:       c,
							total:         totalAfterB,
							totalDemerits: Dc[c+1],
							previous:      Ac[c+1],
						}
						// br.active = slices.Insert(br.active, aIdx, s)
						br.active = append(br.active, nil)
						copy(br.active[aIdx+1:], br.active[aIdx:])
						br.active[aIdx] = s
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
			br.total.Add(&h.Skip)
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

func (br *knuthPlassLineBreaker) computeDemerits(r float64, pb float64, a *knuthPlassNode, b int, c fitnessClass) float64 {
	var d float64
	r3 := 1 + 100*pow3(math.Abs(r))
	if pb >= 0 {
		d = pow2(r3 + float64(pb))
	} else if pb != PenaltyForceBreak {
		d = pow2(r3) - pow2(float64(pb))
	} else {
		d = pow2(r3)
	}
	if br.IsFlagged(a.pos) && br.IsFlagged(b) {
		d += br.α
	}
	if abs(c-a.fitness) > 1 {
		d += br.γ
	}
	d += a.totalDemerits
	return d
}

func pow2(x float64) float64 {
	return x * x
}

func pow3(x float64) float64 {
	return x * x * x
}

func isValidBreakpoint(hList []interface{}, pos int) bool {
	switch h := hList[pos].(type) {
	case *hModePenalty:
		return h.Penalty < PenaltyPreventBreak
	case *hModeGlue:
		_, prevIsBox := hList[pos-1].(*hModeBox)
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

func (br *knuthPlassLineBreaker) AdjustmentRatio(a *knuthPlassNode, b int) float64 {
	br.scratch.SetMinus(br.total, a.total)
	if p, isPenalty := br.hList[b].(*hModePenalty); isPenalty {
		br.scratch.Length += p.width
	}
	available := br.lineWidth(a.line)
	br.scratch.SetMinus(br.scratch, available)
	if br.scratch.Length < -1e-3 { // loose line
		stretch := br.scratch.Stretch
		if stretch.Order > 0 {
			return 0
		}
		if stretch.Val > 0 {
			return -br.scratch.Length / stretch.Val
		}
		return math.Inf(+1)
	} else if br.scratch.Length > 1e-3 { // tight line
		shrink := br.scratch.Shrink
		if shrink.Order > 0 {
			return 0
		}
		if shrink.Val > 0 {
			return -br.scratch.Length / shrink.Val
		}
		return math.Inf(+1)
	}
	return 0
}

type fitnessClass int

const (
	fitnessTight     fitnessClass = -1
	fitnessDecent    fitnessClass = 0
	fitnessLoose     fitnessClass = 1
	fitnessVeryLoose fitnessClass = 2
)

func (b fitnessClass) String() string {
	switch b {
	case fitnessVeryLoose:
		return "very loose"
	case fitnessLoose:
		return "loose"
	case fitnessDecent:
		return "decent"
	case fitnessTight:
		return "tight"
	default:
		return fmt.Sprintf("badnessClass(%d)", b)
	}
}

func getFitnessClass(r float64) fitnessClass {
	var c fitnessClass
	if r < -0.5 {
		c = fitnessTight
	} else if r <= 0.5 {
		c = fitnessDecent
	} else if r <= 1.0 {
		c = fitnessLoose
	} else {
		c = fitnessVeryLoose
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
