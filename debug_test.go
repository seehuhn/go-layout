// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

import "testing"

func TestFormatS(t *testing.T) {
	cases := []struct {
		in   GlueAmount
		want string
	}{
		{GlueAmount{}, "0"},
		{GlueAmount{2}, "2"},
		{GlueAmount{0, 1}, "1fil"},
		{GlueAmount{2, 1}, "2 + 1fil"},
		{GlueAmount{2, -1}, "2 - 1fil"},
		{GlueAmount{-2, 0, 1.5}, "-2 + 1.500fill"},
		{GlueAmount{0, -1, 0, 1}, "-1fil + 1filll"},
	}
	for _, c := range cases {
		if got := formatS(c.in); got != c.want {
			t.Errorf("formatS(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
