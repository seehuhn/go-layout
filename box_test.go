// Copyright 2021 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package layout

// compile-time test: we implement the Box interface
var _ Box = &ruleBox{}
var _ Box = &vBox{}
var _ Box = Kern(0)
var _ Box = &Glue{}
var _ Box = &TextBox{}
