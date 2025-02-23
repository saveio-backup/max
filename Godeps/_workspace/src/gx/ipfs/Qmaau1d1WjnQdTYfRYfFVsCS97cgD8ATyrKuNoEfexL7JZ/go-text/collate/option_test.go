// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package collate

import (
	"reflect"
	"testing"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ/go-text/collate/colltab"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ/go-text/language"
)

var (
	defaultIgnore = ignore(colltab.Tertiary)
	defaultTable  = colltab.Init(locales[0])
)

func TestOptions(t *testing.T) {
	for i, tt := range []struct {
		in  []Option
		out options
	}{
		0: {
			out: options{
				ignore: defaultIgnore,
			},
		},
		1: {
			in: []Option{IgnoreDiacritics},
			out: options{
				ignore: [colltab.NumLevels]bool{false, true, false, true, true},
			},
		},
		2: {
			in: []Option{IgnoreCase, IgnoreDiacritics},
			out: options{
				ignore: ignore(colltab.Primary),
			},
		},
		3: {
			in: []Option{ignoreDiacritics, IgnoreWidth},
			out: options{
				ignore:    ignore(colltab.Primary),
				caseLevel: true,
			},
		},
		4: {
			in: []Option{IgnoreWidth, ignoreDiacritics},
			out: options{
				ignore:    ignore(colltab.Primary),
				caseLevel: true,
			},
		},
		5: {
			in: []Option{IgnoreCase, IgnoreWidth},
			out: options{
				ignore: ignore(colltab.Secondary),
			},
		},
		6: {
			in: []Option{IgnoreCase, IgnoreWidth, Loose},
			out: options{
				ignore: ignore(colltab.Primary),
			},
		},
		7: {
			in: []Option{Force, IgnoreCase, IgnoreWidth, Loose},
			out: options{
				ignore: [colltab.NumLevels]bool{false, true, true, true, false},
			},
		},
		8: {
			in: []Option{IgnoreDiacritics, IgnoreCase},
			out: options{
				ignore: ignore(colltab.Primary),
			},
		},
		9: {
			in: []Option{Numeric},
			out: options{
				ignore:  defaultIgnore,
				numeric: true,
			},
		},
		10: {
			in: []Option{OptionsFromTag(language.MustParse("und-u-ks-level1"))},
			out: options{
				ignore: ignore(colltab.Primary),
			},
		},
		11: {
			in: []Option{OptionsFromTag(language.MustParse("und-u-ks-level4"))},
			out: options{
				ignore: ignore(colltab.Quaternary),
			},
		},
		12: {
			in:  []Option{OptionsFromTag(language.MustParse("und-u-ks-identic"))},
			out: options{},
		},
		13: {
			in: []Option{
				OptionsFromTag(language.MustParse("und-u-kn-true-kb-true-kc-true")),
			},
			out: options{
				ignore:    defaultIgnore,
				caseLevel: true,
				backwards: true,
				numeric:   true,
			},
		},
		14: {
			in: []Option{
				OptionsFromTag(language.MustParse("und-u-kn-true-kb-true-kc-true")),
				OptionsFromTag(language.MustParse("und-u-kn-false-kb-false-kc-false")),
			},
			out: options{
				ignore: defaultIgnore,
			},
		},
		15: {
			in: []Option{
				OptionsFromTag(language.MustParse("und-u-kn-true-kb-true-kc-true")),
				OptionsFromTag(language.MustParse("und-u-kn-foo-kb-foo-kc-foo")),
			},
			out: options{
				ignore:    defaultIgnore,
				caseLevel: true,
				backwards: true,
				numeric:   true,
			},
		},
		16: { // Normal options take precedence over tag options.
			in: []Option{
				Numeric, IgnoreCase,
				OptionsFromTag(language.MustParse("und-u-kn-false-kc-true")),
			},
			out: options{
				ignore:    ignore(colltab.Secondary),
				caseLevel: false,
				numeric:   true,
			},
		},
		17: {
			in: []Option{
				OptionsFromTag(language.MustParse("und-u-ka-shifted")),
			},
			out: options{
				ignore:    defaultIgnore,
				alternate: altShifted,
			},
		},
		18: {
			in: []Option{
				OptionsFromTag(language.MustParse("und-u-ka-blanked")),
			},
			out: options{
				ignore:    defaultIgnore,
				alternate: altBlanked,
			},
		},
		19: {
			in: []Option{
				OptionsFromTag(language.MustParse("und-u-ka-posix")),
			},
			out: options{
				ignore:    defaultIgnore,
				alternate: altShiftTrimmed,
			},
		},
	} {
		c := newCollator(defaultTable)
		c.t = nil
		c.variableTop = 0
		c.f = 0

		c.setOptions(tt.in)
		if !reflect.DeepEqual(c.options, tt.out) {
			t.Errorf("%d: got %v; want %v", i, c.options, tt.out)
		}
	}

}
