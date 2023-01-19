package layout

import (
	"strings"

	"seehuhn.de/go/pdf/color"
	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/sfnt/funit"
	"seehuhn.de/go/sfnt/glyph"
)

type Engine struct {
	// The list of hlist in the horizontal mode.
	// Elements can be of the following types:
	//   *hModeGlue
	//   *hModeText
	hlist []interface{}

	VList     []Box
	PrevDepth float64

	TextWidth    float64
	LeftSkip     *GlueBox
	RightSkip    *GlueBox
	BaseLineSkip float64
}

func (e *Engine) AddText(F *FontInfo, par string) {
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
				e.hlist = append(e.hlist, xSpaceGlue)
				endOfSentence = false
			} else {
				e.hlist = append(e.hlist, spaceGlue)
			}
		}
		gg := F.Font.Typeset(f, F.Size)
		e.hlist = append(e.hlist, &hModeText{
			F:      F,
			glyphs: gg,
			width:  F.Font.ToPDF(F.Size, gg.AdvanceWidth()),
		})
	}
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

type GlueBox struct {
	Length float64
	Plus   stretchAmount
	Minus  stretchAmount
}

func (g *GlueBox) minWidth() float64 {
	if g == nil {
		return 0
	}
	return g.Length - g.Minus.Val
}
