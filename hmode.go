package layout

import (
	"strings"

	"seehuhn.de/go/pdf/font"
	"seehuhn.de/go/sfnt/funit"
	"seehuhn.de/go/sfnt/glyph"
)

type hModeList struct {
	// The list of tokens in the horizontal mode.
	// Elements can be of the following types:
	//   *hModeGlue
	//   *hModeText
	tokens []interface{}
}

func TokenizeParagraph(par string, F *fontInfo) *hModeList {
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
	parFillSkip := &hModeGlue{
		glue: glue{
			Plus: stretchAmount{Val: 1, Level: 1},
		},
		Text:    "\n",
		NoBreak: true,
	}

	var tokens []interface{}
	endOfSentence := false
	for i, f := range strings.Fields(par) {
		if i > 0 {
			if endOfSentence {
				tokens = append(tokens, xSpaceGlue)
				endOfSentence = false
			} else {
				tokens = append(tokens, spaceGlue)
			}
		}
		gg := F.font.Typeset(f, F.size)
		tokens = append(tokens, &hModeText{
			glyphs:   gg,
			font:     F.font,
			fontSize: F.size,
			width:    F.font.ToPDF(F.size, gg.AdvanceWidth()),
		})
	}
	// TODO(voss): add an infinite penalty before the ParFillSkip glue
	tokens = append(tokens, parFillSkip)

	return &hModeList{tokens: tokens}
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
