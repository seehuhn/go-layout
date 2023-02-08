package layout

import (
	"math"
	"strings"

	"seehuhn.de/go/pdf/color"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/pdf/graphics"
	"seehuhn.de/go/sfnt/funit"
	"seehuhn.de/go/sfnt/glyph"
)

type Engine struct {
	// The list of HList in the horizontal mode.
	// Elements can be of the following types:
	//   *hModeGlue
	//   *hModeText
	HList []interface{}

	VList     []Box
	PrevDepth float64

	TextWidth float64
	LeftSkip  *GlueBox
	RightSkip *GlueBox

	TopSkip      float64
	BottomGlue   *GlueBox
	BaseLineSkip float64
	ParSkip      *GlueBox // TODO(voss)

	InterLinePenalty Penalty
	ClubPenalty      Penalty
	WidowPenalty     Penalty

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
		GlueBox: GlueBox{
			Length: pdfSpaceWidth,
			Plus:   stretchAmount{Val: pdfSpaceWidth / 2},
			Minus:  stretchAmount{Val: pdfSpaceWidth / 3},
		},
		Text: " ",
	}
	xSpaceGlue := &hModeGlue{
		GlueBox: GlueBox{
			Length: 1.5 * pdfSpaceWidth,
			Plus:   stretchAmount{Val: pdfSpaceWidth * 1.5},
			Minus:  stretchAmount{Val: pdfSpaceWidth},
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
		e.HList = append(e.HList, &hModeText{
			F:      F,
			glyphs: gg,
			width:  F.Font.ToPDF(F.Size, gg.AdvanceWidth()),
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

type hModeText struct {
	F      *FontInfo
	glyphs glyph.Seq
	width  float64
}

type hModeGlue struct {
	GlueBox
	Text    string
	NoBreak bool
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

var (
	PenaltyPreventBreak = Penalty(math.Inf(+1))
	PenaltyForceBreak   = Penalty(math.Inf(-1))
)
