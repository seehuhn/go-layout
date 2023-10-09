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
	"errors"
	"math"
)

type texLineBreaker struct {
	passive *passiveNode // most recent node on passive list

	activeWidth    deltaLike // distance from first active node to cur p
	curActiveWidth deltaLike // distance from current active node
	background     deltaLike // length of an “empty” line
	breakWidth     deltaLike // length being computed after current break

	curP       int
	secondPass bool
	finalPass  bool
	threshold  float64
}

func (e *Engine) newTexLineBreaker() (*texLineBreaker, error) {
	q := e.LeftSkip
	if q.isInfiniteShrink() {
		return nil, errors.New("infinitly shrinkable left skip")
	}
	r := e.RightSkip
	if r.isInfiniteShrink() {
		return nil, errors.New("infinitly shrinkable right skip")
	}

	br := &texLineBreaker{}
	br.background.width = q.Length + r.Length
	br.background.stretch[q.Stretch.Order] += q.Stretch.Val
	br.background.stretch[r.Stretch.Order] += r.Stretch.Val
	br.background.shrink = q.Shrink.Val + r.Shrink.Val

	return br, nil
}

// TryBreak tests if the current breakpoint curP is feasible, by running
// through the active list to see what lines of text can be made from active
// nodes to curP.  If feasible breaks are possible, new break nodes are
// created.  If curP is too far from an active node, that node is deactivated.
//
// The parameter pi to try break is the penalty associated with a break at cur
// p; we have pi = ejectPenalty if the break is forced, and pi = infPenalty if
// the break is illegal.
//
// Hyphenated is set depending on whether or not the current break is at a
// discNode. The end of a paragraph is also regarded as "hyphenated"; this case
// is distinguishable by the condition curP = null.
func (e *texLineBreaker) tryBreak(pi float64, hyphenated bool) {
	if math.IsInf(pi, +1) {
		// this breakpoint is inhibited by infinite penalty
		return
	}
	// noBreakYet := true
	// prevR := ...
}

// texBreakNode describes break that is feasible, in the sense that there is a
// way to end a line at the given place without requiring any line to stretch
// more than a given tolerance.
type texBreakNode struct {
	breakPos uint32       // index into hlist (glue, math, penalty, disc)
	lineNo   uint16       // ordinal number of the line that will follow this breakpoint
	fitness  fitnessClass // classification of the line ending at this breakpoint
}

// An active node for a given breakpoint contains six fields:
type activeNode struct {
	link          *deltaNode   // next node in the list of active nodes; the last active node has link = lastActive
	breakNode     *passiveNode // the passive node associated with this breakpoint
	lineNo        uint16       // the number of the line that follows this breakpoint
	fitness       fitnessClass // classification of the line ending at this breakpoint
	hyphenated    bool         // whether this breakpoint is a disc node
	totalDemerits float64      // minimum possible sum of demerits over all lines leading from the beginning of the paragraph to this breakpoint
}

type deltaNode struct {
	link *activeNode
	deltaLike
}

type deltaLike struct {
	width   float64
	stretch [4]float64 // units of pt, fil, fill, and filll
	shrink  float64
}

func (d *deltaLike) add(other deltaLike) {
	d.width += other.width
	for i, x := range other.stretch {
		d.stretch[i] += x
	}
	d.shrink += other.shrink
}

type passiveNode struct {
	link      *passiveNode // the passive node created just before this one (nil for the first passive node)
	curBreak  uint32       // the position of this breakpoint in the horizontal list
	prevBreak *passiveNode //the passive node that precedes this one in an optimal path to this breakpoint
}
