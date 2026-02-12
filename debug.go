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
	"fmt"
	"math"
	"strings"

	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/font/standard"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/pdf/graphics/color"
	"seehuhn.de/go/pdf/graphics/content"
	"seehuhn.de/go/pdf/graphics/content/builder"
	"seehuhn.de/go/pdf/graphics/extgstate"
	"seehuhn.de/go/pdf/page"
	"seehuhn.de/go/pdf/pagetree"
)

// DebugPageBreak creates a PDF page which explains the page break decisions.
func (e *Engine) DebugPageBreak(tree *pagetree.Writer, rm *pdf.ResourceManager) error {
	const (
		overshot     = 1.4
		glueHeight   = 12
		visualGap    = 8
		bottomMargin = 36
		leftMargin   = 48
		topMargin    = 36
		rightMargin  = 120
	)
	var (
		geomColor  = color.DeviceRGB{0, 0, 0.9}
		breakColor = color.DeviceRGB{0.9, 0, 0}
	)

	F := standard.TimesRoman.New()

	height := e.TextHeight
	cand := e.vGetCandidates(height)
	if len(cand) == 0 {
		return nil
	}

	numBoxes := cand[len(cand)-1].pos + 6
	for numBoxes < len(e.vList) && vDiscardible(e.vList[numBoxes]) {
		numBoxes++
	}
	numBoxes++ // include one non-discardible box
	if numBoxes > len(e.vList) {
		numBoxes = len(e.vList)
	}

	vPos := []float64{0}
	width := 100.0 // start with a minimum width
	for i := 0; i < numBoxes; i++ {
		ext := e.vList[i].Extent()
		naturalHeight := ext.Height + ext.Depth

		if ext.Width > width {
			width = ext.Width
		}

		dy := naturalHeight
		if naturalHeight < glueHeight || ext.WhiteSpaceOnly {
			dy = glueHeight
		}
		vPos = append(vPos, vPos[len(vPos)-1]+dy+visualGap)
	}
	visualHeight := vPos[len(vPos)-1]

	// Create a builder to accumulate drawing operations
	b := builder.New(content.Page, nil)

	yTop := bottomMargin + visualHeight
	target := height
	for i := 0; i < numBoxes; i++ {
		yMid := yTop - (vPos[i+1]+vPos[i])/2

		box := e.vList[i]
		ext := box.Extent()
		var yBase, yAscent, yDescent float64
		if ext.WhiteSpaceOnly {
			yBase = yMid
			yAscent = yTop - vPos[i] - 0.5*visualGap
			yDescent = yTop - vPos[i+1] + 0.5*visualGap
		} else {
			dy := ext.Height + ext.Depth
			yBase = yMid - 0.5*dy + ext.Depth
			yAscent = yBase + ext.Height
			yDescent = yBase - ext.Depth
		}

		str, isStretch := box.(stretcher)
		shr, isShrink := box.(shrinker)

		// draw the box contents
		if !ext.WhiteSpaceOnly {
			b.PushGraphicsState()
			b.SetFillColor(color.DeviceGray(0.9))
			b.Rectangle(leftMargin-1, yDescent-1, ext.Width+2, yAscent-yDescent+2)
			b.Fill()
			b.PopGraphicsState()

			box.Draw(b, leftMargin, yBase)
		}

		// show the box types
		b.TextBegin()
		b.TextSetFont(F, 7)
		if ext.WhiteSpaceOnly {
			b.TextFirstLine(leftMargin+2, yMid-2)
		} else {
			b.TextFirstLine(leftMargin+2, yAscent-1)
		}
		var label string
		switch obj := box.(type) {
		case penalty:
			pVal := float64(obj)
			if math.IsInf(pVal, +1) {
				label = "penalty (no break)"
			} else if math.IsInf(pVal, -1) {
				label = "penalty (force break)"
			} else {
				label = fmt.Sprintf("penalty %s", format(pVal))
			}
		case *hBox:
			label = "hbox"
		case *Glue:
			if obj == e.ParSkip {
				label = "glue (parskip)"
			} else {
				label = "glue"
			}
		default:
			label = fmt.Sprintf("%T", box)
		}
		b.TextShow(label)
		b.TextEnd()

		// draw the geometry annotations
		b.PushGraphicsState()
		b.SetStrokeColor(geomColor)
		b.SetLineCap(graphics.LineCapRound)
		b.SetLineWidth(0.5)
		x := leftMargin + width + 10.0
		if isStretch || isShrink {
			h := yAscent - yDescent
			numWiggles := max(int(math.Round(h)/3), 4)
			dh := h / float64(numWiggles)
			y := yAscent
			xw := x - 2
			b.MoveTo(xw, y)
			for i := 1; i <= numWiggles; i++ {
				oldX := xw
				oldY := y
				xw = 2*x - xw
				y -= dh
				b.CurveTo(oldX, oldY-2, xw, y+2, xw, y)
			}
			b.Stroke()
		} else if !ext.WhiteSpaceOnly {
			b.MoveTo(x, yAscent)
			b.LineTo(x, yDescent)
			b.MoveTo(x-2, yAscent)
			b.LineTo(x+2, yAscent)
			b.MoveTo(x-2, yDescent)
			b.LineTo(x+2, yDescent)
			b.Stroke()
		}

		// show vertial sizes in the right margin
		label = format(ext.Height + ext.Depth)
		if isStretch {
			label = label + fmt.Sprintf(" plus %s", formatS(str.GetStretch()))
		}
		if isShrink {
			label = label + fmt.Sprintf(" minus %s", formatS(shr.GetShrink()))
		}
		if !ext.WhiteSpaceOnly || label != "0" {
			b.TextBegin()
			b.SetFillColor(geomColor)
			b.TextSetFont(F, 7)
			b.TextFirstLine(leftMargin+width+15, yMid-2)
			b.TextShow(label)
			b.TextEnd()
		}

		// draw a mark to indicate the target height
		newTarget := target - ext.Height - ext.Depth
		if target > 0 && newTarget <= 0 {
			y := yAscent - target
			b.SetFillColor(geomColor)
			b.MoveTo(leftMargin-8, y)
			b.LineTo(leftMargin-18, y+4)
			b.LineTo(leftMargin-18, y-4)
			b.Fill()
		}
		target = newTarget
		b.PopGraphicsState()
	}

	// mark the line break candidates
	b.PushGraphicsState()
	b.SetFillColor(breakColor)
	x := leftMargin + 0.5*width
	bestPos := -1
	bestCost := math.Inf(+1)
	for _, c := range cand {
		cost := c.badness + float64(c.penalty)
		if cost <= bestCost {
			bestCost = cost
			bestPos = c.pos
		}

		y := yTop - vPos[c.pos]
		b.TextBegin()
		b.TextSetFont(F, 7)
		b.TextFirstLine(x, y-3)
		b.TextShowAligned(fmt.Sprintf("— b=%s, p=%s —", format(c.badness), format(float64(c.penalty))), 0, 0.5)
		b.TextEnd()
	}
	b.PopGraphicsState()

	// mark the accumulated page height, minus the required total
	total := &Glue{
		Length: -height,
	}

	ext0 := e.vList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}
	total.Length += topSkip

	if topSkip > 0 {
		b.TextBegin()
		b.SetFillColor(breakColor)
		b.TextSetFont(F, 7)
		b.TextFirstLine(leftMargin+width+15, yTop+5)
		b.TextShow(fmt.Sprintf("%s (topskip)", format(topSkip)))
		b.TextEnd()
	}
	prevDepth := 0.0
	for i := 0; i < bestPos; i++ {
		box := e.vList[i]
		if _, isPenalty := box.(penalty); isPenalty {
			continue
		}

		ext := box.Extent()
		total.Length += ext.Height + prevDepth
		prevDepth = ext.Depth

		if stretch, ok := box.(stretcher); ok {
			total.Stretch.IncrementBy(stretch.GetStretch())
		}
		if shrink, ok := box.(shrinker); ok {
			total.Shrink.IncrementBy(shrink.GetShrink())
		}

		b.TextBegin()
		b.SetFillColor(breakColor)
		b.TextSetFont(F, 7)
		b.TextFirstLine(leftMargin+width-30, yTop-vPos[i+1]-2)
		b.TextShow(fmt.Sprintf("%s plus %s minus %s",
			format(total.Length), formatS(total.Stretch), formatS(total.Shrink)))
		b.TextEnd()
	}
	y := yTop - vPos[bestPos] - 12
	if bg := e.BottomGlue; bg != nil {
		b.TextBegin()
		b.SetFillColor(breakColor)
		b.TextSetFont(F, 7)
		b.TextFirstLine(leftMargin+width+15, y)
		b.TextShow(fmt.Sprintf("%s plus %s minus %s (bottomglue)",
			format(bg.Length), formatS(bg.Stretch), formatS(bg.Shrink)))
		b.TextEnd()
		total.Add(bg)
		y -= 10

		b.TextBegin()
		b.SetFillColor(breakColor)
		b.TextSetFont(F, 7)
		b.TextFirstLine(leftMargin+width-30, y)
		b.TextShow(fmt.Sprintf("%s plus %s minus %s",
			format(total.Length), formatS(total.Stretch), formatS(total.Shrink)))
		b.TextEnd()
		y -= 10
	}

	next := bestPos + 1
	if next < len(e.vList) && vDiscardible(e.vList[next]) {
		next++
	}

	// draw the final page outlines
	b.PushGraphicsState()
	b.SetStrokeColor(breakColor)
	b.SetLineWidth(2)
	b.MoveTo(leftMargin+30, yTop-vPos[bestPos]-2)
	b.LineTo(leftMargin-5, yTop-vPos[bestPos]-2)
	b.LineTo(leftMargin-5, yTop+2)
	b.LineTo(leftMargin+30, yTop+2)
	if next < len(e.vList) {
		b.MoveTo(leftMargin-5, bottomMargin)
		b.LineTo(leftMargin-5, yTop-vPos[next]+2)
		b.LineTo(leftMargin+30, yTop-vPos[next]+2)
	}
	b.Stroke()
	b.PopGraphicsState()

	// Create page object
	p := &page.Page{
		MediaBox: &pdf.Rectangle{
			LLx: 0,
			LLy: 0,
			URx: leftMargin + width + rightMargin,
			URy: topMargin + visualHeight + bottomMargin,
		},
		Resources: b.Resources,
		Contents:  []*page.Content{{Operators: b.Stream}},
	}
	if err := tree.AppendPage(p); err != nil {
		return err
	}

	return nil
}

// DebugLineBreaks creates a PDF page which explains the line break decisions.
func (e *Engine) DebugLineBreaks(tree *pagetree.Writer, rm *pdf.ResourceManager, F font.Instance) error {
	// This must match the code in [Engine.EndParagraph]

	const (
		bottomMargin = 36
		leftMargin   = 48
		topMargin    = 36
		rightMargin  = 240
	)
	var (
		// geomColor  = color.DeviceRGB{0, 0, 0.9}
		breakColor      = color.DeviceRGB{0.9, 0, 0}
		annotationColor = color.DeviceRGB{0, 0.7, 0}
	)

	hList := e.hList
	if e.ParFillSkip != nil {
		hList = append(hList, &hModePenalty{Penalty: PenaltyPreventBreak})
		hList = append(hList, e.ParFillSkip)
	}
	hList = append(hList, &hModePenalty{Penalty: PenaltyForceBreak})

	lineWidth := &Glue{Length: e.TextWidth}
	lineWidth = lineWidth.Minus(e.LeftSkip).Minus(e.RightSkip)

	br := &knuthPlassLineBreaker{
		α: 100,
		γ: 100,
		ρ: 1000,
		q: 0,
		lineWidth: func(lineNo int) *Glue {
			return lineWidth
		},
		hList: hList,
	}
	breaks := br.Run()

	var startPos []int
	var hLists [][]any
	var lineContents [][]Box
	var lineBoxes []Box
	var xxx [][]float64

	prevPos := 0
	for _, pos := range breaks {
		startPos = append(startPos, prevPos)
		var currentLine []Box
		if e.LeftSkip != nil {
			currentLine = append(currentLine, e.LeftSkip)
		}
		for _, item := range hList[prevPos:pos] {
			switch h := item.(type) {
			case *Glue:
				currentLine = append(currentLine, h)
			case *hModeBox:
				currentLine = append(currentLine, h.Box)
			}
		}
		hLists = append(hLists, hList[prevPos:pos])
		if e.RightSkip != nil {
			currentLine = append(currentLine, e.RightSkip)
		}
		xx := horizontalLayout(leftMargin, e.TextWidth, currentLine...)
		if e.LeftSkip != nil {
			xx = xx[1:]
		}
		if e.RightSkip == nil {
			xx = append(xx, e.TextWidth)
		}
		xxx = append(xxx, xx)

	skipDiscardible:
		for prevPos = pos; prevPos < len(hList); prevPos++ {
			switch h := br.hList[prevPos].(type) {
			case *hModeBox:
				break skipDiscardible
			case *hModePenalty:
				if prevPos > pos && h.Penalty == PenaltyForceBreak {
					break skipDiscardible
				}
			}
		}

		lineContents = append(lineContents, currentLine)
		lineBox := HBoxTo(e.TextWidth, currentLine...)
		lineBoxes = append(lineBoxes, lineBox)
	}

	// Now we have gathered all the lines.
	// Create a page which shows the line breaks.

	// Create a builder to accumulate drawing operations
	b := builder.New(content.Page, nil)

	gs := &extgstate.ExtGState{
		FillAlpha: 0.75,
		Set:       graphics.StateFillAlpha,
	}

	visualHeight := 0.0
	for _, box := range lineBoxes {
		ext := box.Extent()
		visualHeight += ext.Depth + ext.Height
		visualHeight += 10
	}

	// Show the text width
	b.PushGraphicsState()
	b.SetStrokeColor(breakColor)
	b.SetLineWidth(0.5)
	b.MoveTo(leftMargin, 0)
	b.LineTo(leftMargin, bottomMargin+visualHeight+topMargin)
	b.MoveTo(leftMargin+e.TextWidth, 0)
	b.LineTo(leftMargin+e.TextWidth, bottomMargin+visualHeight+topMargin)
	b.Stroke()
	b.PopGraphicsState()

	x := float64(leftMargin)
	y := bottomMargin + visualHeight
	for i, box := range lineBoxes {
		ext := box.Extent()
		y -= ext.Height

		// draw the line
		box.Draw(b, x, y)

		// draw the first few tokens after the linebreak, to illustrate
		// the linebreak decision
		xx := xxx[i]
		xEnd := xx[len(xx)-1]
		xx = xx[:len(xx)-1]
		x := xEnd
		var extra []Box
		pos := startPos[i] + len(xx)
		for pos < len(hList) {
			xx = append(xx, x)
			switch h := hList[pos].(type) {
			case *Glue:
				extra = append(extra, h)
				x += h.Length
			case *hModeBox:
				extra = append(extra, h.Box)
				x += h.width
			}
			if x >= leftMargin+e.TextWidth+72 {
				break
			}
			pos++
		}
		b.PushGraphicsState()
		overflow := HBox(extra...)
		ext = overflow.Extent()
		b.Rectangle(xEnd, y-ext.Depth, leftMargin+e.TextWidth+72-xEnd, ext.Height+ext.Depth)
		b.ClipNonZero()
		b.EndPath()
		overflow.Draw(b, xEnd, y)
		b.SetExtGState(gs)
		b.SetFillColor(color.DeviceRGB{1, 1, 1})
		b.Rectangle(xEnd, y-ext.Depth, leftMargin+e.TextWidth+72-xEnd, ext.Height+ext.Depth)
		b.Fill()
		b.PopGraphicsState()

		// draw a little triangle for each potential breakpoint
		b.PushGraphicsState()
		b.SetFillColor(breakColor)
		for j, x := range xx {
			if !isValidBreakpoint(hList, startPos[i]+j) {
				continue
			}
			b.MoveTo(x, y-1)
			b.LineTo(x-1, y-4)
			b.LineTo(x+1, y-4)
		}
		b.Fill()
		b.PopGraphicsState()

		// add the annotations
		b.PushGraphicsState()
		b.SetLineWidth(3.5)
		b.SetStrokeColor(color.DeviceGray(0.9))
		b.MoveTo(leftMargin+e.TextWidth+72+1.5, 0)
		b.LineTo(leftMargin+e.TextWidth+72+1.5, bottomMargin+visualHeight+topMargin)
		b.Stroke()
		b.SetLineWidth(1.5)
		b.SetStrokeColor(color.DeviceGray(0.8))
		b.MoveTo(leftMargin+e.TextWidth+72+0.5, 0)
		b.LineTo(leftMargin+e.TextWidth+72+0.5, bottomMargin+visualHeight+topMargin)
		b.Stroke()
		b.SetLineWidth(0.5)
		b.SetStrokeColor(color.DeviceGray(0.6))
		b.MoveTo(leftMargin+e.TextWidth+72, 0)
		b.LineTo(leftMargin+e.TextWidth+72, bottomMargin+visualHeight+topMargin)
		b.Stroke()
		b.PopGraphicsState()

		b.TextBegin()
		b.TextSetFont(F, 6)
		b.SetFillColor(annotationColor)
		b.TextFirstLine(leftMargin+e.TextWidth+72+10, y+4)
		total := totalWidthAndGlue(lineContents[i])
		b.TextShow(fmt.Sprintf("%+.1f", e.TextWidth-total.Length))
		var r float64
		if total.Length > e.TextWidth+0.05 {
			r = (e.TextWidth - total.Length) / total.Shrink.Val
			label := fmt.Sprintf(" / %.1f (%.0f%%)", total.Shrink.Val, -100*r)
			if total.Stretch.Order > 0 {
				r = 0
				label = " / inf"
			}
			b.TextShow(label)
		} else if total.Length < e.TextWidth-0.05 {
			r = (e.TextWidth - total.Length) / total.Stretch.Val
			label := fmt.Sprintf(" / %.1f (%.0f%%)", total.Stretch.Val, 100*r)
			if total.Stretch.Order > 0 {
				r = 0
				label = " / inf"
			}
			b.TextShow(label)
		}
		b.TextSecondLine(0, -7)
		b.TextShowAligned(fmt.Sprintf(" r = %.2f", r), 30, 0)
		c := getFitnessClass(r)
		if c != fitnessDecent {
			b.TextShowAligned(c.String(), 25, 1)
		}
		b.TextEnd()

		y -= ext.Depth
		y -= 10
	}

	// Create page object
	p := &page.Page{
		MediaBox: &pdf.Rectangle{
			LLx: 0,
			LLy: 0,
			URx: leftMargin + e.TextWidth + rightMargin,
			URy: topMargin + visualHeight + bottomMargin,
		},
		Resources: b.Resources,
		Contents:  []*page.Content{{Operators: b.Stream}},
	}
	if err := tree.AppendPage(p); err != nil {
		return err
	}

	return nil
}

func format(x float64) string {
	xInt := int(math.Round(x))
	if math.Abs(x-float64(xInt)) < 1e-6 {
		return fmt.Sprintf("%d", xInt)
	}
	if math.Abs(x) >= 1e7 {
		return fmt.Sprintf("%.6g", x)
	}
	return fmt.Sprintf("%.3f", x)
}

func formatS(x glueAmount) string {
	unit := ""
	if x.Order > 0 {
		unit = "fi" + strings.Repeat("l", x.Order)
	}
	return format(x.Val) + unit
}
