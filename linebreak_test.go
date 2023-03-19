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
	"math"
	"testing"

	"golang.org/x/text/language"
	"seehuhn.de/go/pdf/document"
	"seehuhn.de/go/pdf/font/simple"
	"seehuhn.de/go/pdf/pages"
)

func TestLineBreaks(t *testing.T) {
	paper := pages.A4
	hSize := math.Round(15 / 2.54 * 72)
	const fontSize = 10

	doc, err := document.CreateSinglePage("test_LineBreaks.pdf", paper.URx, paper.URy)
	if err != nil {
		t.Fatal(err)
	}

	F1, err := simple.EmbedFile(doc.Out, "../otf/SourceSerif4-Regular.otf", "F1", language.BritishEnglish)
	if err != nil {
		t.Fatal(err)
	}
	geom := F1.GetGeometry()

	e := &Engine{
		TextWidth:    hSize,
		RightSkip:    &Glue{Stretch: glueAmount{Val: 36, Order: 0}},
		ParFillSkip:  Skip(0, 1, 1, 0, 0),
		BaseLineSkip: geom.ToPDF16(fontSize, geom.BaseLineSkip),
	}

	e.HAddText(&FontInfo{Font: F1, Size: 10}, testText)
	e.EndParagraph()

	paragraph := VTop(e.vList...)

	paragraph.Draw(doc.Page, 72, 25/2.54*72)

	err = doc.Close()
	if err != nil {
		t.Fatal(err)
	}
}

const testText = `Call me Ishmael. Some years ago—never mind how long precisely—having little or no money in my purse, and nothing particular to interest me on shore, I thought I would sail about a little and see the watery part of the world. It is a way I have of driving off the spleen and regulating the circulation. Whenever I find myself growing grim about the mouth; whenever it is a damp, drizzly November in my soul; whenever I find myself involuntarily pausing before coffin warehouses, and bringing up the rear of every funeral I meet; and especially whenever my hypos get such an upper hand of me, that it requires a strong moral principle to prevent me from deliberately stepping into the street, and methodically knocking people’s hats off—then, I account it high time to get to sea as soon as I can. This is my substitute for pistol and ball. With a philosophical flourish Cato throws himself upon his sword; I quietly take to the ship. There is nothing surprising in this. If they but knew it, almost all men in their degree, some time or other, cherish very nearly the same feelings towards the ocean with me.`
