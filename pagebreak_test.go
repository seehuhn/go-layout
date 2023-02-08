package layout

import (
	"math"
	"testing"
)

func TestVBreakCandidates1(t *testing.T) {
	vList := []Box{
		&RuleBox{
			BoxExtent: BoxExtent{
				Width:  10,
				Height: 10,
				Depth:  0,
			},
		},
		Penalty(123),
		&RuleBox{
			BoxExtent: BoxExtent{
				Width:  10,
				Height: 10,
				Depth:  0,
			},
		},
		Penalty(456),
		&RuleBox{
			BoxExtent: BoxExtent{
				Width:  10,
				Height: 10,
				Depth:  0,
			},
		},
	}
	e := &Engine{
		VList: vList,
	}

	cand := e.vGetCandidates(20)
	if len(cand) != 2 {
		t.Fatalf("expected 2 breaks, got %d", len(cand))
	}
	if !math.IsInf(cand[0].badness, +1) {
		t.Fatalf("expected break with badness +oo, got %f", cand[0].badness)
	}
	if cand[1].pos != 3 {
		t.Fatalf("expected break at pos 3, got %d", cand[0].pos)
	}
	if cand[1].badness != 0 {
		t.Fatalf("expected break with badness 0, got %f", cand[0].badness)
	}
	if cand[1].penalty != 456 {
		t.Fatalf("expected break with penalty 456, got %f", cand[0].penalty)
	}

	cand = e.vGetCandidates(15)
	if len(cand) != 1 {
		t.Fatalf("expected 1 breaks, got %d", len(cand))
	}
	if cand[0].pos != 1 {
		t.Fatalf("expected break at pos 1, got %d", cand[0].pos)
	}
	if !math.IsInf(cand[0].badness, +1) {
		t.Fatalf("expected break with badness +oo, got %f", cand[0].badness)
	}
	if cand[0].penalty != 123 {
		t.Fatalf("expected break with penalty 123, got %f", cand[0].penalty)
	}
}

func TestVBreakCandidates2(t *testing.T) {
	vList := []Box{
		&RuleBox{
			BoxExtent: BoxExtent{
				Width:  10,
				Height: 10,
				Depth:  5,
			},
		},
		Penalty(1),
		&RuleBox{
			BoxExtent: BoxExtent{
				Width:  10,
				Height: 10,
				Depth:  5,
			},
		},
		Penalty(2),
		&RuleBox{
			BoxExtent: BoxExtent{
				Width:  10,
				Height: 10,
				Depth:  5,
			},
		},
	}
	e := &Engine{
		VList: vList,
	}

	cand := e.vGetCandidates(10)
	if len(cand) != 1 {
		t.Fatalf("expected 1 breaks, got %d", len(cand))
	}
	if cand[0].pos != 1 {
		t.Fatalf("expected break at pos 1, got %d", cand[0].pos)
	}
	if cand[0].badness != 0 {
		t.Fatalf("expected break with badness 0, got %f", cand[0].badness)
	}
	if cand[0].penalty != 1 {
		t.Fatalf("expected break with penalty 456, got %f", cand[0].penalty)
	}
}
