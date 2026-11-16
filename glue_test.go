// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGlueAmountAdd(t *testing.T) {
	cases := []struct {
		a, b, want GlueAmount
	}{
		{GlueAmount{1}, GlueAmount{2}, GlueAmount{3}},
		{GlueAmount{0, 1}, GlueAmount{0, 2}, GlueAmount{0, 3}},
		{GlueAmount{1}, GlueAmount{0, 2}, GlueAmount{1, 2}},
		{GlueAmount{2, 1}, GlueAmount{0, -1}, GlueAmount{2, 0}},
	}
	for _, c := range cases {
		got := c.a
		got.Add(c.b)
		if got != c.want {
			t.Errorf("%v + %v = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestGlueAmountMinus(t *testing.T) {
	cases := []struct {
		a, b, want GlueAmount
	}{
		{GlueAmount{3}, GlueAmount{1}, GlueAmount{2}},
		{GlueAmount{0, 3}, GlueAmount{0, 1}, GlueAmount{0, 2}},
		{GlueAmount{3}, GlueAmount{0, 1}, GlueAmount{3, -1}},
		{GlueAmount{5, 1}, GlueAmount{2, 1}, GlueAmount{3, 0}},
	}
	for _, c := range cases {
		got := c.a.Minus(c.b)
		if got != c.want {
			t.Errorf("%v - %v = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestGlueAmountDominant(t *testing.T) {
	cases := []struct {
		a     GlueAmount
		val   float64
		order int
	}{
		{GlueAmount{}, 0, 0},
		{GlueAmount{2}, 2, 0},
		{GlueAmount{2, 1}, 1, 1},
		{GlueAmount{2, 0, 0, 3}, 3, 3},
		{GlueAmount{2, -1}, -1, 1},
	}
	for _, c := range cases {
		val, order := c.a.Dominant()
		if val != c.val || order != c.order {
			t.Errorf("%v: Dominant() = %g, %d, want %g, %d", c.a, val, order, c.val, c.order)
		}
	}
}

func TestSkip(t *testing.T) {
	got := Skip(10, 2, 1, 3, 0)
	want := &Glue{Length: 10, Stretch: GlueAmount{0, 2}, Shrink: GlueAmount{3}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong glue (-want +got):\n%s", diff)
	}
}

func TestGlueAdd(t *testing.T) {
	g := Skip(10, 2, 0, 1, 0)
	g.Add(Skip(5, 3, 1, 1, 0))
	want := &Glue{Length: 15, Stretch: GlueAmount{2, 3}, Shrink: GlueAmount{2}}
	if diff := cmp.Diff(want, g); diff != "" {
		t.Errorf("add failed (-want +got):\n%s", diff)
	}

	g.Add(nil)
	if diff := cmp.Diff(want, g); diff != "" {
		t.Errorf("adding nil changed glue (-want +got):\n%s", diff)
	}
}

func TestGlueMinus(t *testing.T) {
	a := Skip(10, 2, 0, 1, 1)
	b := Skip(4, 3, 1, 1, 0)
	want := &Glue{Length: 6, Stretch: GlueAmount{2, -3}, Shrink: GlueAmount{-1, 1}}

	got := a.Minus(b)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("subtraction failed (-want +got):\n%s", diff)
	}

	var set Glue
	set.SetMinus(a, b)
	if diff := cmp.Diff(want, &set); diff != "" {
		t.Errorf("subtraction into receiver failed (-want +got):\n%s", diff)
	}

	// the receiver may be the same as either argument
	alias := a.Clone()
	alias.SetMinus(alias, b)
	if diff := cmp.Diff(want, alias); diff != "" {
		t.Errorf("subtraction into first argument failed (-want +got):\n%s", diff)
	}
	alias = b.Clone()
	alias.SetMinus(a, alias)
	if diff := cmp.Diff(want, alias); diff != "" {
		t.Errorf("subtraction into second argument failed (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(a, a.Minus(nil)); diff != "" {
		t.Errorf("subtracting nil changed glue (-want +got):\n%s", diff)
	}
}

func TestGlueClone(t *testing.T) {
	a := Skip(10, 2, 0, 1, 1)
	b := a.Clone()
	if diff := cmp.Diff(a, b); diff != "" {
		t.Errorf("clone differs (-want +got):\n%s", diff)
	}
	b.Length = 0
	if a.Length != 10 {
		t.Error("clone shares storage with original")
	}
}

func TestGlueMinMaxLength(t *testing.T) {
	cases := []struct {
		g        *Glue
		min, max float64
	}{
		{Skip(10, 2, 0, 3, 0), 7, 12},
		{Skip(10, 2, 1, 3, 0), 7, math.Inf(+1)},
		{Skip(10, 2, 0, 3, 1), math.Inf(-1), 12},
		{&Glue{Length: 10, Stretch: GlueAmount{2, 1, 0, -1}}, 10, math.Inf(+1)},
	}
	for _, c := range cases {
		if got := c.g.minLength(); got != c.min {
			t.Errorf("%v: minLength = %g, want %g", c.g, got, c.min)
		}
		if got := c.g.maxLength(); got != c.max {
			t.Errorf("%v: maxLength = %g, want %g", c.g, got, c.max)
		}
	}
}

func TestGlueBox(t *testing.T) {
	g := Skip(10, 2, 0, 3, 0)
	want := &BoxExtent{Width: 10, Height: 10, WhiteSpaceOnly: true}
	if diff := cmp.Diff(want, g.Extent()); diff != "" {
		t.Errorf("extent differs (-want +got):\n%s", diff)
	}
	if got := g.GetStretch(); got != (GlueAmount{2}) {
		t.Errorf("stretch = %v, want [2 0 0 0]", got)
	}
	if got := g.GetShrink(); got != (GlueAmount{3}) {
		t.Errorf("shrink = %v, want [3 0 0 0]", got)
	}
}

func TestTotalGlue(t *testing.T) {
	boxes := []Box{
		&ruleBox{BoxExtent{Width: 5, Height: 2, Depth: 1}},
		Skip(3, 1, 0, 1, 0),
		Skip(4, 1, 1, 0, 0),
	}

	want := &Glue{Length: 12, Stretch: GlueAmount{1, 1}, Shrink: GlueAmount{1}}
	if diff := cmp.Diff(want, totalWidthAndGlue(boxes)); diff != "" {
		t.Errorf("width total differs (-want +got):\n%s", diff)
	}

	want = &Glue{Length: 3 + 3 + 4, Stretch: GlueAmount{1, 1}, Shrink: GlueAmount{1}}
	if diff := cmp.Diff(want, totalHeightAndGlue(boxes)); diff != "" {
		t.Errorf("height total differs (-want +got):\n%s", diff)
	}
}

// TestCancelledInfiniteGlue checks that finite stretch takes effect when the
// infinite stretch in a list cancels out.
func TestCancelledInfiniteGlue(t *testing.T) {
	boxes := []Box{
		&ruleBox{BoxExtent{Width: 10, Height: 10}},
		Skip(0, 1, 1, 0, 0),
		Skip(0, -1, 1, 0, 0),
		Skip(0, 10, 0, 0, 0),
		&ruleBox{BoxExtent{Width: 10, Height: 10}},
	}

	xx := horizontalLayout(0, 30, boxes...)
	if xx[4] != 20 {
		t.Errorf("last box at x=%g, want 20", xx[4])
	}

	yy := verticalLayout(30, 30, boxes...)
	if yy[4] != 0 {
		t.Errorf("last box at y=%g, want 0", yy[4])
	}
}

// TestInexactInfiniteGlue checks that infinite stretch cancels even when the
// amounts have no exact binary representation.
func TestInexactInfiniteGlue(t *testing.T) {
	var total GlueAmount
	for _, x := range []float64{0.1, 0.2, -0.3} {
		total.Add(GlueAmount{0, x})
	}
	if val, order := total.Dominant(); val != 0 || order != 0 {
		t.Errorf("Dominant() = %g, %d, want 0, 0", val, order)
	}

	diff := GlueAmount{0, 0.1 + 0.2}.Minus(GlueAmount{0, 0.3})
	if val, order := diff.Dominant(); val != 0 || order != 0 {
		t.Errorf("Dominant() of difference = %g, %d, want 0, 0", val, order)
	}

	boxes := []Box{
		&ruleBox{BoxExtent{Width: 10, Height: 10}},
		Skip(0, 0.1, 1, 0, 0),
		Skip(0, 0.2, 1, 0, 0),
		Skip(0, -0.3, 1, 0, 0),
		Skip(0, 10, 0, 0, 0),
		&ruleBox{BoxExtent{Width: 10, Height: 10}},
	}
	xx := horizontalLayout(0, 30, boxes...)
	if xx[5] != 20 {
		t.Errorf("last box at x=%g, want 20", xx[5])
	}
	yy := verticalLayout(30, 30, boxes...)
	if yy[5] != 0 {
		t.Errorf("last box at y=%g, want 0", yy[5])
	}
}

// TestNegativeInfiniteGlue checks that glue with negative infinite stretch
// fills a box on its own.
func TestNegativeInfiniteGlue(t *testing.T) {
	boxes := []Box{
		&ruleBox{BoxExtent{Width: 10}},
		Skip(0, -1, 1, 0, 0),
		&ruleBox{BoxExtent{Width: 10}},
	}
	xx := horizontalLayout(0, 30, boxes...)
	if xx[2] != 20 {
		t.Errorf("last box at x=%g, want 20", xx[2])
	}
}
