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
	"seehuhn.de/go/pdf/font/type1"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/pdf/graphics/color"
	"seehuhn.de/go/pdf/pagetree"
)

// DebugPageBreak creates a PDF page which explains the page break decisions.
func (e *Engine) DebugPageBreak(tree *pagetree.Writer) error {
	const (
		overshot     = 1.4
		glueHeight   = 12
		visualGap    = 8
		bottomMargin = 36
		leftMargin   = 48
		topMargin    = 36
		rigthMargin  = 120
	)
	var (
		geomColor  = color.DeviceRGB.New(0, 0, 0.9)
		breakColor = color.DeviceRGB.New(0.9, 0, 0)
	)

	F, err := type1.TimesRoman.Embed(tree.Out, &font.Options{ResName: "Q"})
	if err != nil {
		return err
	}

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

	contentRef := tree.Out.Alloc()
	stream, err := tree.Out.OpenStream(contentRef, nil, pdf.FilterCompress{})
	if err != nil {
		return err
	}
	page := graphics.NewWriter(stream, pdf.GetVersion(tree.Out))

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
			page.PushGraphicsState()
			page.SetFillColor(color.DeviceGray.New(0.9))
			page.Rectangle(leftMargin-1, yDescent-1, ext.Width+2, yAscent-yDescent+2)
			page.Fill()
			page.PopGraphicsState()

			box.Draw(page, leftMargin, yBase)
		}

		// show the box types
		page.TextStart()
		page.TextSetFont(F, 7)
		if ext.WhiteSpaceOnly {
			page.TextFirstLine(leftMargin+2, yMid-2)
		} else {
			page.TextFirstLine(leftMargin+2, yAscent-1)
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
		page.TextShow(label)
		page.TextEnd()

		// draw the geometry annotations
		page.PushGraphicsState()
		page.SetStrokeColor(geomColor)
		page.SetLineCap(graphics.LineCapRound)
		page.SetLineWidth(0.5)
		x := leftMargin + width + 10.0
		if isStretch || isShrink {
			h := yAscent - yDescent
			numWiggles := int(math.Round(h) / 3)
			if numWiggles < 4 {
				numWiggles = 4
			}
			dh := h / float64(numWiggles)
			y := yAscent
			xw := x - 2
			page.MoveTo(xw, y)
			for i := 1; i <= numWiggles; i++ {
				oldX := xw
				oldY := y
				xw = 2*x - xw
				y -= dh
				page.CurveTo(oldX, oldY-2, xw, y+2, xw, y)
			}
			page.Stroke()
		} else if !ext.WhiteSpaceOnly {
			page.MoveTo(x, yAscent)
			page.LineTo(x, yDescent)
			page.MoveTo(x-2, yAscent)
			page.LineTo(x+2, yAscent)
			page.MoveTo(x-2, yDescent)
			page.LineTo(x+2, yDescent)
			page.Stroke()
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
			page.TextStart()
			page.SetFillColor(geomColor)
			page.TextSetFont(F, 7)
			page.TextFirstLine(leftMargin+width+15, yMid-2)
			page.TextShow(label)
			page.TextEnd()
		}

		// draw a mark to indicate the target height
		newTarget := target - ext.Height - ext.Depth
		if target > 0 && newTarget <= 0 {
			y := yAscent - target
			page.SetFillColor(geomColor)
			page.MoveTo(leftMargin-8, y)
			page.LineTo(leftMargin-18, y+4)
			page.LineTo(leftMargin-18, y-4)
			page.Fill()
		}
		target = newTarget
		page.PopGraphicsState()
	}

	// mark the line break candidates
	page.PushGraphicsState()
	page.SetFillColor(breakColor)
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
		page.TextStart()
		page.TextSetFont(F, 7)
		page.TextFirstLine(x, y-3)
		page.TextShowAligned(fmt.Sprintf("— b=%s, p=%s —", format(c.badness), format(float64(c.penalty))), 0, 0.5)
		page.TextEnd()
	}
	page.PopGraphicsState()

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
		page.TextStart()
		page.SetFillColor(breakColor)
		page.TextSetFont(F, 7)
		page.TextFirstLine(leftMargin+width+15, yTop+5)
		page.TextShow(fmt.Sprintf("%s (topskip)", format(topSkip)))
		page.TextEnd()
	}
	prevDept := 0.0
	for i := 0; i < bestPos; i++ {
		box := e.vList[i]
		if _, isPenalty := box.(penalty); isPenalty {
			continue
		}

		ext := box.Extent()
		total.Length += ext.Height + prevDept
		prevDept = ext.Depth

		if stretch, ok := box.(stretcher); ok {
			total.Stretch.IncrementBy(stretch.GetStretch())
		}
		if shrink, ok := box.(shrinker); ok {
			total.Shrink.IncrementBy(shrink.GetShrink())
		}

		page.TextStart()
		page.SetFillColor(breakColor)
		page.TextSetFont(F, 7)
		page.TextFirstLine(leftMargin+width-30, yTop-vPos[i+1]-2)
		page.TextShow(fmt.Sprintf("%s plus %s minus %s",
			format(total.Length), formatS(total.Stretch), formatS(total.Shrink)))
		page.TextEnd()
	}
	y := yTop - vPos[bestPos] - 12
	if b := e.BottomGlue; b != nil {
		page.TextStart()
		page.SetFillColor(breakColor)
		page.TextSetFont(F, 7)
		page.TextFirstLine(leftMargin+width+15, y)
		page.TextShow(fmt.Sprintf("%s plus %s minus %s (bottomglue)",
			format(b.Length), formatS(b.Stretch), formatS(b.Shrink)))
		page.TextEnd()
		total.Add(b)
		y -= 10

		page.TextStart()
		page.SetFillColor(breakColor)
		page.TextSetFont(F, 7)
		page.TextFirstLine(leftMargin+width-30, y)
		page.TextShow(fmt.Sprintf("%s plus %s minus %s",
			format(total.Length), formatS(total.Stretch), formatS(total.Shrink)))
		page.TextEnd()
		y -= 10
	}

	next := bestPos + 1
	if next < len(e.vList) && vDiscardible(e.vList[next]) {
		next++
	}

	// draw the final page outlines
	page.PushGraphicsState()
	page.SetStrokeColor(breakColor)
	page.SetLineWidth(2)
	page.MoveTo(leftMargin+30, yTop-vPos[bestPos]-2)
	page.LineTo(leftMargin-5, yTop-vPos[bestPos]-2)
	page.LineTo(leftMargin-5, yTop+2)
	page.LineTo(leftMargin+30, yTop+2)
	if next < len(e.vList) {
		page.MoveTo(leftMargin-5, bottomMargin)
		page.LineTo(leftMargin-5, yTop-vPos[next]+2)
		page.LineTo(leftMargin+30, yTop-vPos[next]+2)
	}
	page.Stroke()
	page.PopGraphicsState()

	// add the page to the page tree
	err = stream.Close()
	if err != nil {
		return err
	}
	dict := pdf.Dict{
		"Type":     pdf.Name("Page"),
		"Contents": contentRef,
		"MediaBox": &pdf.Rectangle{
			LLx: 0,
			LLy: 0,
			URx: leftMargin + width + rigthMargin,
			URy: topMargin + visualHeight + bottomMargin,
		},
	}
	if page.Resources != nil {
		dict["Resources"] = pdf.AsDict(page.Resources)
	}
	err = tree.AppendPage(dict)
	if err != nil {
		return err
	}

	return nil
}

// DebugLineBreaks creates a PDF page which explains the line break decisions.
func (e *Engine) DebugLineBreaks(tree *pagetree.Writer, F font.Embedded) error {
	// This must match the code in [Engine.EndParagraph]

	const (
		bottomMargin = 36
		leftMargin   = 48
		topMargin    = 36
		rigthMargin  = 240
	)
	var (
		// geomColor  = color.DeviceRGB.New(0, 0, 0.9)
		breakColor      = color.DeviceRGB.New(0.9, 0, 0)
		annotationColor = color.DeviceRGB.New(0, 0.7, 0)
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
	var hLists [][]interface{}
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

	contentRef := tree.Out.Alloc()
	stream, err := tree.Out.OpenStream(contentRef, nil, pdf.FilterCompress{})
	if err != nil {
		return err
	}
	page := graphics.NewWriter(stream, pdf.GetVersion(tree.Out))

	state := graphics.State{}
	state.FillAlpha = 0.75
	state.Set = graphics.StateFillAlpha
	gs, err := graphics.NewExtGState(state, "gs:t")
	if err != nil {
		return err
	}

	visualHeight := 0.0
	for _, box := range lineBoxes {
		ext := box.Extent()
		visualHeight += ext.Depth + ext.Height
		visualHeight += 10
	}

	// Show the text width
	page.PushGraphicsState()
	page.SetStrokeColor(breakColor)
	page.SetLineWidth(0.5)
	page.MoveTo(leftMargin, 0)
	page.LineTo(leftMargin, bottomMargin+visualHeight+topMargin)
	page.MoveTo(leftMargin+e.TextWidth, 0)
	page.LineTo(leftMargin+e.TextWidth, bottomMargin+visualHeight+topMargin)
	page.Stroke()
	page.PopGraphicsState()

	x := float64(leftMargin)
	y := bottomMargin + visualHeight
	for i, box := range lineBoxes {
		ext := box.Extent()
		y -= ext.Height

		// draw the line
		box.Draw(page, x, y)

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
		page.PushGraphicsState()
		overflow := HBox(extra...)
		ext = overflow.Extent()
		page.Rectangle(xEnd, y-ext.Depth, leftMargin+e.TextWidth+72-xEnd, ext.Height+ext.Depth)
		page.ClipNonZero()
		page.EndPath()
		overflow.Draw(page, xEnd, y)
		page.SetExtGState(gs)
		page.SetFillColor(color.DeviceRGB.New(1, 1, 1))
		page.Rectangle(xEnd, y-ext.Depth, leftMargin+e.TextWidth+72-xEnd, ext.Height+ext.Depth)
		page.Fill()
		page.PopGraphicsState()

		// draw a little triangle for each potential breakpoint
		page.PushGraphicsState()
		page.SetFillColor(breakColor)
		for j, x := range xx {
			if !isValidBreakpoint(hList, startPos[i]+j) {
				continue
			}
			page.MoveTo(x, y-1)
			page.LineTo(x-1, y-4)
			page.LineTo(x+1, y-4)
		}
		page.Fill()
		page.PopGraphicsState()

		// add the annotations
		page.PushGraphicsState()
		page.SetLineWidth(3.5)
		page.SetStrokeColor(color.DeviceGray.New(0.9))
		page.MoveTo(leftMargin+e.TextWidth+72+1.5, 0)
		page.LineTo(leftMargin+e.TextWidth+72+1.5, bottomMargin+visualHeight+topMargin)
		page.Stroke()
		page.SetLineWidth(1.5)
		page.SetStrokeColor(color.DeviceGray.New(0.8))
		page.MoveTo(leftMargin+e.TextWidth+72+0.5, 0)
		page.LineTo(leftMargin+e.TextWidth+72+0.5, bottomMargin+visualHeight+topMargin)
		page.Stroke()
		page.SetLineWidth(0.5)
		page.SetStrokeColor(color.DeviceGray.New(0.6))
		page.MoveTo(leftMargin+e.TextWidth+72, 0)
		page.LineTo(leftMargin+e.TextWidth+72, bottomMargin+visualHeight+topMargin)
		page.Stroke()
		page.PopGraphicsState()

		page.TextStart()
		page.TextSetFont(F, 6)
		page.SetFillColor(annotationColor)
		page.TextFirstLine(leftMargin+e.TextWidth+72+10, y+4)
		total := totalWidthAndGlue(lineContents[i])
		page.TextShow(fmt.Sprintf("%+.1f", e.TextWidth-total.Length))
		var r float64
		if total.Length > e.TextWidth+0.05 {
			r = (e.TextWidth - total.Length) / total.Shrink.Val
			label := fmt.Sprintf(" / %.1f (%.0f%%)", total.Shrink.Val, -100*r)
			if total.Stretch.Order > 0 {
				r = 0
				label = " / inf"
			}
			page.TextShow(label)
		} else if total.Length < e.TextWidth-0.05 {
			r = (e.TextWidth - total.Length) / total.Stretch.Val
			label := fmt.Sprintf(" / %.1f (%.0f%%)", total.Stretch.Val, 100*r)
			if total.Stretch.Order > 0 {
				r = 0
				label = " / inf"
			}
			page.TextShow(label)
		}
		page.TextSecondLine(0, -7)
		page.TextShowAligned(fmt.Sprintf(" r = %.2f", r), 30, 0)
		c := getFitnessClass(r)
		if c != fitnessDecent {
			page.TextShowAligned(c.String(), 25, 1)
		}
		page.TextEnd()

		y -= ext.Depth
		y -= 10
	}

	err = stream.Close()
	if err != nil {
		return err
	}
	dict := pdf.Dict{
		"Type":     pdf.Name("Page"),
		"Contents": contentRef,
		"MediaBox": &pdf.Rectangle{
			LLx: 0,
			LLy: 0,
			URx: leftMargin + e.TextWidth + rigthMargin,
			URy: topMargin + visualHeight + bottomMargin,
		},
	}
	if page.Resources != nil {
		dict["Resources"] = pdf.AsDict(page.Resources)
	}
	err = tree.AppendPage(dict)
	if err != nil {
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
