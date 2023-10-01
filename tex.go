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

type passiveNode struct {
	link      *passiveNode // the passive node created just before this one (nil for the first passive node)
	curBreak  uint32       // the position of this breakpoint in the horizontal list
	prevBreak *passiveNode //the passive node that precedes this one in an optimal path to this breakpoint
}

type deltaNode struct {
	link    *activeNode
	width   float64
	stretch [4]float64
	shrink  float64
}
