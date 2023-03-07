package layout

import (
	"unicode"

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
	HList      []interface{}
	AfterPunct bool
	AfterSpace bool

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

func (e *Engine) HAddText(F *FontInfo, text string) {
	geom := F.Font.GetGeometry()
	space := F.Font.Layout(" ", F.Size)
	var spaceWidth funit.Int
	if len(space) == 1 && space[0].Gid != 0 {
		spaceWidth = funit.Int(space[0].Advance)
	} else {
		space = nil
		spaceWidth = funit.Int(geom.UnitsPerEm / 4)
	}
	pdfSpaceWidth := geom.ToPDF(F.Size, spaceWidth)

	spaceGlue := &hModeGlue{
		Skip: Skip{
			Length:  pdfSpaceWidth,
			Stretch: glueAmount{Val: pdfSpaceWidth / 2},
			Shrink:  glueAmount{Val: pdfSpaceWidth / 3},
		},
		Text: " ",
	}
	xSpaceGlue := &hModeGlue{
		Skip: Skip{
			Length:  1.5 * pdfSpaceWidth,
			Stretch: glueAmount{Val: pdfSpaceWidth * 1.5},
			Shrink:  glueAmount{Val: pdfSpaceWidth},
		},
		Text: " ",
	}
	addSpace := func() {
		if e.AfterPunct {
			e.HList = append(e.HList, xSpaceGlue)
		} else {
			e.HList = append(e.HList, spaceGlue)
		}
	}
	addRunes := func(rr []rune) {
		gg := F.Font.Layout(string(rr), F.Size)
		box := &TextBox{
			F:      F,
			Glyphs: gg,
		}
		e.HList = append(e.HList, &hModeBox{
			Box:   box,
			width: geom.ToPDF(F.Size, gg.AdvanceWidth()),
		})
	}

	var run []rune
	for _, r := range text {
		if r == 0x200B { // ZERO WIDTH SPACE
			if len(run) > 0 {
				addRunes(run)
				run = run[:0]
			}
			e.HList = append(e.HList, &hModePenalty{})
		} else if unicode.IsSpace(r) &&
			r != 0x00A0 && // NO-BREAK SPACE
			r != 0x2007 && // FIGURE SPACE
			r != 0x202F { // NARROW NO-BREAK SPACE
			if len(run) > 0 {
				addRunes(run)
				run = run[:0]
			}
			if !e.AfterSpace {
				addSpace()
			}
			e.AfterSpace = true
			e.AfterPunct = false
		} else {
			run = append(run, r)
			e.AfterSpace = false
			e.AfterPunct = r == '.' || r == '!' || r == '?'
		}
	}
	if len(run) > 0 {
		addRunes(run)
	}
}

// HAddGlue adds a glue item to the horizontal mode list.
func (e *Engine) HAddGlue(g *Skip) {
	e.HList = append(e.HList, &hModeGlue{Skip: *g})
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

type Penalty float64

func (obj Penalty) Extent() *BoxExtent {
	return &BoxExtent{
		WhiteSpaceOnly: true,
	}
}

func (obj Penalty) Draw(page *graphics.Page, xPos, yPos float64) {
	// pass
}
