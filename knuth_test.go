// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import "testing"

// TestAdjustmentRatioAfterInfiniteGlue checks that infinite glue before the
// start of a line does not affect the adjustment ratio of the line.
func TestAdjustmentRatioAfterInfiniteGlue(t *testing.T) {
	hList := []any{
		&hModeBox{width: 10},
		Skip(0, 1, 1, 0, 0),
		&hModeBox{width: 10},
		Skip(10, 10, 0, 0, 0),
		&hModeBox{width: 10},
		&hModePenalty{Penalty: PenaltyForceBreak},
	}
	br := &knuthPlassLineBreaker{
		lineWidth: func(int) *Glue { return &Glue{Length: 40} },
		hList:     hList,
	}

	// totals of the material before a line starting at position 2, and
	// before the break at position 5
	a := &knuthPlassNode{pos: 2, line: 1}
	for _, item := range hList[:2] {
		addToTotal(&a.total, item)
	}
	for _, item := range hList[:5] {
		addToTotal(&br.total, item)
	}

	// the line has natural width 30 and stretch 10, and needs to fill 40
	if r := br.AdjustmentRatio(a, 5); r != 1 {
		t.Errorf("adjustment ratio = %g, want 1", r)
	}
}

func addToTotal(total *Glue, item any) {
	switch h := item.(type) {
	case *hModeBox:
		total.Length += h.width
	case *Glue:
		total.Add(h)
	}
}
