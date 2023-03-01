package layout

import (
	"strings"

	"seehuhn.de/go/pdf/color"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/sfnt/funit"
)

// A list of horizontal mode items can contain the following types:
//  - *hModeBox: a box which is not affected by line breaking.
//        The only property relevant for line breaking is the width.
//  - *hModeGlue:
//  - *hModePenalty: an optional breakpoint

type hModeBox struct {
	Box
	width float64
}

type hModeGlue struct {
	Skip
	Text string
}

type hModePenalty struct {
	Penalty float64
	width   float64
	flagged bool
}

type Engine struct {
	// The list of HList in the horizontal mode.
	// Elements can be of the following types:
	//   *hModeGlue
	//   *hModeText
	HList []interface{}

	VList     []Box
	PrevDepth float64

	TextWidth   float64
	LeftSkip    *Skip
	RightSkip   *Skip
	ParFillSkip *Skip

	TopSkip      float64
	BottomGlue   *Skip
	BaseLineSkip float64
	ParSkip      *Skip // TODO(voss)

	InterLinePenalty float64
	ClubPenalty      float64
	WidowPenalty     float64

	PageNumber int

	AfterPageFunc func(*Engine, *graphics.Page) error
}

func (e *Engine) HAddText(F *FontInfo, par string) {
	space := F.Font.Layout([]rune{' '})
	var spaceWidth funit.Int
	if len(space) == 1 && space[0].Gid != 0 {
		spaceWidth = funit.Int(space[0].Advance)
	} else {
		space = nil
		spaceWidth = funit.Int(F.Font.UnitsPerEm / 4)
	}
	pdfSpaceWidth := F.Font.ToPDF(F.Size, spaceWidth)

	spaceGlue := &hModeGlue{
		Skip: Skip{
			Length:  pdfSpaceWidth,
			Stretch: stretchAmount{Val: pdfSpaceWidth / 2},
			Shrink:  stretchAmount{Val: pdfSpaceWidth / 3},
		},
		Text: " ",
	}
	xSpaceGlue := &hModeGlue{
		Skip: Skip{
			Length:  1.5 * pdfSpaceWidth,
			Stretch: stretchAmount{Val: pdfSpaceWidth * 1.5},
			Shrink:  stretchAmount{Val: pdfSpaceWidth},
		},
		Text: " ",
	}

	endOfSentence := false
	for i, f := range strings.Fields(par) {
		if i > 0 {
			if endOfSentence {
				e.HList = append(e.HList, xSpaceGlue)
				endOfSentence = false
			} else {
				e.HList = append(e.HList, spaceGlue)
			}
		}
		gg := F.Font.Typeset(f, F.Size)
		box := &TextBox{
			F:      F,
			Glyphs: gg,
		}
		e.HList = append(e.HList, &hModeBox{
			Box:   box,
			width: F.Font.ToPDF(F.Size, gg.AdvanceWidth()),
		})
	}
}

func (e *Engine) VAddBox(b Box) {
	ext := b.Extent()
	if len(e.VList) > 0 {
		gap := ext.Height + e.PrevDepth
		if gap+0.001 < e.BaseLineSkip {
			e.VList = append(e.VList, Kern(e.BaseLineSkip-gap))
		}
	}
	e.VList = append(e.VList, b)
	e.PrevDepth = ext.Depth
}

type FontInfo struct {
	Font  *font.Font
	Size  float64
	Color color.Color
}

type Penalty float64

func (obj Penalty) Extent() *BoxExtent {
	return &BoxExtent{
		WhiteSpaceOnly: true,
	}
}

func (obj Penalty) Draw(page *graphics.Page, xPos, yPos float64) {
	// pass
}
