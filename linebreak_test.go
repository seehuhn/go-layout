package layout

import (
	"math"
	"testing"

	"golang.org/x/text/language"
	"seehuhn.de/go/pdf"
	"seehuhn.de/go/pdf/font/simple"
	"seehuhn.de/go/pdf/pages"
)

func TestLineBreaks(t *testing.T) {
	const fontSize = 10
	hSize := math.Round(15 / 2.54 * 72)

	out, err := pdf.Create("test_LineBreaks.pdf")
	if err != nil {
		t.Fatal(err)
	}

	FF1, err := simple.LoadFont("../otf/SourceSerif4-Regular.otf",
		language.BritishEnglish)
	if err != nil {
		t.Fatal(err)
	}
	F1, err := FF1.Embed(out, "F1")
	if err != nil {
		t.Fatal(err)
	}
	geom := F1.GetGeometry()

	e := &Engine{
		TextWidth:    hSize,
		RightSkip:    &Skip{Stretch: glueAmount{Val: 36, Order: 0}},
		ParFillSkip:  Glue(0, 1, 1, 0, 0),
		BaseLineSkip: geom.ToPDF16(fontSize, geom.BaseLineSkip),
	}

	e.HAddText(&FontInfo{Font: F1, Size: 10}, testText)
	e.EndParagraph()

	for _, box := range e.VList {
		t.Logf("%T: %v", box, box)
	}

	pageTree := pages.InstallTree(out, &pages.InheritableAttributes{
		MediaBox: pages.A4,
	})

	page, err := pages.AppendPage(pageTree)
	if err != nil {
		t.Fatal(err)
	}

	paragraph := VTop(e.VList)

	paragraph.Draw(page.Page, 72, 25/2.54*72)

	_, err = page.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = out.Close()
	if err != nil {
		t.Error(err)
	}
}

const testText = `Call me Ishmael. Some years ago—never mind how long precisely—having little or no money in my purse, and nothing particular to interest me on shore, I thought I would sail about a little and see the watery part of the world. It is a way I have of driving off the spleen and regulating the circulation. Whenever I find myself growing grim about the mouth; whenever it is a damp, drizzly November in my soul; whenever I find myself involuntarily pausing before coffin warehouses, and bringing up the rear of every funeral I meet; and especially whenever my hypos get such an upper hand of me, that it requires a strong moral principle to prevent me from deliberately stepping into the street, and methodically knocking people’s hats off—then, I account it high time to get to sea as soon as I can. This is my substitute for pistol and ball. With a philosophical flourish Cato throws himself upon his sword; I quietly take to the ship. There is nothing surprising in this. If they but knew it, almost all men in their degree, some time or other, cherish very nearly the same feelings towards the ocean with me.`
