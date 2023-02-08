package layout

import (
	"fmt"
	"math"
	"strings"

	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/color"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/pdf/pages"
)

func (e *Engine) VisualisePageBreak(tree *pages.Tree, F *font.Font, height float64) error {
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
		geomColor  = color.RGB(0, 0, 0.9)
		breakColor = color.RGB(0.9, 0, 0)
	)

	cand := e.vGetCandidates(height)
	if len(cand) == 0 {
		return nil
	}

	numBoxes := cand[len(cand)-1].pos + 3
	for numBoxes < len(e.VList) && vDiscardible(e.VList[numBoxes]) {
		numBoxes++
	}
	numBoxes++ // include one non-discardible box
	if numBoxes > len(e.VList) {
		numBoxes = len(e.VList)
	}

	vPos := []float64{0}
	width := 100.0 // start with a minimum width
	for i := 0; i < numBoxes; i++ {
		ext := e.VList[i].Extent()
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

	page, err := graphics.NewPage(tree.Out)
	if err != nil {
		return err
	}

	yTop := bottomMargin + visualHeight
	target := height
	for i := 0; i < numBoxes; i++ {
		yMid := yTop - (vPos[i+1]+vPos[i])/2

		box := e.VList[i]
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
			page.SetFillColor(color.Gray(0.9))
			page.Rectangle(leftMargin-1, yDescent-1, ext.Width+2, yAscent-yDescent+2)
			page.Fill()
			page.PopGraphicsState()

			box.Draw(page, leftMargin, yBase)
		}

		// show the box types
		page.BeginText()
		page.SetFont(F, 7)
		if ext.WhiteSpaceOnly {
			page.StartLine(leftMargin+2, yMid-2)
		} else {
			page.StartLine(leftMargin+2, yAscent-1)
		}
		var label string
		switch obj := box.(type) {
		case Penalty:
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
		case *GlueBox:
			if obj == e.ParSkip {
				label = "glue (parskip)"
			} else {
				label = "glue"
			}
		default:
			label = fmt.Sprintf("%T", box)
		}
		page.ShowText(label)
		page.EndText()

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
			label = label + fmt.Sprintf(" plus %s", formatS(str.Stretch()))
		}
		if isShrink {
			label = label + fmt.Sprintf(" minus %s", formatS(shr.Shrink()))
		}
		if !ext.WhiteSpaceOnly || label != "0" {
			page.BeginText()
			page.SetFillColor(geomColor)
			page.SetFont(F, 7)
			page.StartLine(leftMargin+width+15, yMid-2)
			page.ShowText(label)
			page.EndText()
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
		page.BeginText()
		page.SetFont(F, 7)
		page.StartLine(x, y-3)
		page.ShowTextAligned(fmt.Sprintf("— b=%s, p=%s —", format(c.badness), format(float64(c.penalty))), 0, 0.5)
		page.EndText()
	}
	page.PopGraphicsState()

	// mark the accumulated page height, minus the required total
	total := &GlueBox{
		Length: -height,
	}

	ext0 := e.VList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}
	total.Length += topSkip

	if topSkip > 0 {
		page.BeginText()
		page.SetFillColor(breakColor)
		page.SetFont(F, 7)
		page.StartLine(leftMargin+width+15, yTop+5)
		page.ShowText(fmt.Sprintf("%s (topskip)", format(topSkip)))
		page.EndText()
	}
	prevDept := 0.0
	for i := 0; i < bestPos; i++ {
		box := e.VList[i]
		if _, isPenalty := box.(Penalty); isPenalty {
			continue
		}

		ext := box.Extent()
		total.Length += ext.Height + prevDept
		prevDept = ext.Depth

		if stretch, ok := box.(stretcher); ok {
			total.Plus.Add(stretch.Stretch())
		}
		if shrink, ok := box.(shrinker); ok {
			total.Minus.Add(shrink.Shrink())
		}

		page.BeginText()
		page.SetFillColor(breakColor)
		page.SetFont(F, 7)
		page.StartLine(leftMargin+width-30, yTop-vPos[i+1]-2)
		page.ShowText(fmt.Sprintf("%s plus %s minus %s",
			format(total.Length), formatS(total.Plus), formatS(total.Minus)))
		page.EndText()
	}
	y := yTop - vPos[bestPos] - 12
	if b := e.BottomGlue; b != nil {
		page.BeginText()
		page.SetFillColor(breakColor)
		page.SetFont(F, 7)
		page.StartLine(leftMargin+width+15, y)
		page.ShowText(fmt.Sprintf("%s plus %s minus %s (bottomglue)",
			format(b.Length), formatS(b.Plus), formatS(b.Minus)))
		page.EndText()
		total.Add(b)
		y -= 10

		page.BeginText()
		page.SetFillColor(breakColor)
		page.SetFont(F, 7)
		page.StartLine(leftMargin+width-30, y)
		page.ShowText(fmt.Sprintf("%s plus %s minus %s",
			format(total.Length), formatS(total.Plus), formatS(total.Minus)))
		page.EndText()
		y -= 10
	}

	next := bestPos + 1
	if next < len(e.VList) && vDiscardible(e.VList[next]) {
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
	if next < len(e.VList) {
		page.MoveTo(leftMargin-5, bottomMargin)
		page.LineTo(leftMargin-5, yTop-vPos[next]+2)
		page.LineTo(leftMargin+30, yTop-vPos[next]+2)
	}
	page.Stroke()
	page.PopGraphicsState()

	// add the page to the page tree
	dict, err := page.Close()
	if err != nil {
		return err
	}
	dict["MediaBox"] = &pdf.Rectangle{
		LLx: 0,
		LLy: 0,
		URx: leftMargin + width + rigthMargin,
		URy: topMargin + visualHeight + bottomMargin,
	}
	_, err = tree.AppendPage(dict)
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

func formatS(x stretchAmount) string {
	unit := ""
	if x.Order > 0 {
		unit = "fi" + strings.Repeat("l", x.Order)
	}
	return format(x.Val) + unit
}
