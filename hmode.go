package layout

import (
	"strings"

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
	vlist []Box

	textWidth float64
	leftSkip  *glue
	rightSkip *glue

	baseLineSkip float64
}

func (e *Engine) TokenizeParagraph(par string, F *fontInfo) {
	space := F.font.Layout([]rune{' '})
	var spaceWidth funit.Int
	if len(space) == 1 && space[0].Gid != 0 {
		spaceWidth = funit.Int(space[0].Advance)
	} else {
		space = nil
		spaceWidth = funit.Int(F.font.UnitsPerEm / 4)
	}
	pdfSpaceWidth := F.font.ToPDF(F.size, spaceWidth)

	spaceGlue := &hModeGlue{
		glue: glue{
			Length: pdfSpaceWidth,
			Plus:   stretchAmount{Val: pdfSpaceWidth / 2},
			Minus:  stretchAmount{Val: pdfSpaceWidth / 3},
		},
		Text: " ",
	}
	xSpaceGlue := &hModeGlue{
		glue: glue{
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
		gg := F.font.Typeset(f, F.size)
		e.hlist = append(e.hlist, &hModeText{
			glyphs:   gg,
			font:     F.font,
			fontSize: F.size,
			width:    F.font.ToPDF(F.size, gg.AdvanceWidth()),
		})
	}
}

type fontInfo struct {
	font *font.Font
	size float64
}

type hModeText struct {
	glyphs   glyph.Seq
	font     *font.Font
	fontSize float64
	width    float64
}

type hModeGlue struct {
	glue
	Text    string
	NoBreak bool
}

type glue struct {
	Length float64
	Plus   stretchAmount
	Minus  stretchAmount
}

func (g *glue) minWidth() float64 {
	if g == nil {
		return 0
	}
	return g.Length - g.Minus.Val
}
