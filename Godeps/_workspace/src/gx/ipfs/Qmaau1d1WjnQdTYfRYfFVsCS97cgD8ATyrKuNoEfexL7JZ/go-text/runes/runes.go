// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package runes provide transforms for UTF-8 encoded text.
package runes

import (
	"unicode"
	"unicode/utf8"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmaau1d1WjnQdTYfRYfFVsCS97cgD8ATyrKuNoEfexL7JZ/go-text/transform"
)

// A Set is a collection of runes.
type Set interface {
	// Contains returns true if r is contained in the set.
	Contains(r rune) bool
}

type setFunc func(rune) bool

func (s setFunc) Contains(r rune) bool {
	return s(r)
}

// Note: using funcs here instead of wrapping types result in cleaner
// documentation and a smaller API.

// In creates a Set with a Contains method that returns true for all runes in
// the given RangeTable.
func In(rt *unicode.RangeTable) Set {
	return setFunc(func(r rune) bool { return unicode.Is(rt, r) })
}

// In creates a Set with a Contains method that returns true for all runes not
// in the given RangeTable.
func NotIn(rt *unicode.RangeTable) Set {
	return setFunc(func(r rune) bool { return !unicode.Is(rt, r) })
}

// Predicate creates a Set with a Contains method that returns f(r).
func Predicate(f func(rune) bool) Set {
	return setFunc(f)
}

// Transformer implements the transform.Transformer interface.
type Transformer struct {
	transform.Transformer
}

// Bytes returns a new byte slice with the result of converting b using t.  It
// calls Reset on t. It returns nil if any error was found. This can only happen
// if an error-producing Transformer is passed to If.
func (t Transformer) Bytes(b []byte) []byte {
	b, _, err := transform.Bytes(t, b)
	if err != nil {
		return nil
	}
	return b
}

// String returns a string with the result of converting s using t. It calls
// Reset on t. It returns the empty string if any error was found. This can only
// happen if an error-producing Transformer is passed to If.
func (t Transformer) String(s string) string {
	s, _, err := transform.String(t, s)
	if err != nil {
		return ""
	}
	return s
}

// TODO:
// - Copy: copying strings and bytes in whole-rune units.
// - Validation (maybe)
// - Well-formed-ness (maybe)

const runeErrorString = string(utf8.RuneError)

// Remove returns a Transformer that removes runes r for which s.Contains(r).
// Illegal input bytes are replaced by RuneError before being passed to f.
func Remove(s Set) Transformer {
	if f, ok := s.(setFunc); ok {
		// This little trick cuts the running time of BenchmarkRemove for sets
		// created by Predicate roughly in half.
		// TODO: special-case RangeTables as well.
		return Transformer{remove(f)}
	}
	return Transformer{remove(s.Contains)}
}

// TODO: remove transform.RemoveFunc.

type remove func(r rune) bool

func (remove) Reset() {}

// Transform implements transform.Transformer.
func (t remove) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for r, size := rune(0), 0; nSrc < len(src); {
		if r = rune(src[nSrc]); r < utf8.RuneSelf {
			size = 1
		} else {
			r, size = utf8.DecodeRune(src[nSrc:])

			if size == 1 {
				// Invalid rune.
				if !atEOF && !utf8.FullRune(src[nSrc:]) {
					err = transform.ErrShortSrc
					break
				}
				// We replace illegal bytes with RuneError. Not doing so might
				// otherwise turn a sequence of invalid UTF-8 into valid UTF-8.
				// The resulting byte sequence may subsequently contain runes
				// for which t(r) is true that were passed unnoticed.
				if !t(utf8.RuneError) {
					if nDst+3 > len(dst) {
						err = transform.ErrShortDst
						break
					}
					dst[nDst+0] = runeErrorString[0]
					dst[nDst+1] = runeErrorString[1]
					dst[nDst+2] = runeErrorString[2]
					nDst += 3
				}
				nSrc++
				continue
			}
		}

		if t(r) {
			nSrc += size
			continue
		}
		if nDst+size > len(dst) {
			err = transform.ErrShortDst
			break
		}
		for i := 0; i < size; i++ {
			dst[nDst] = src[nSrc]
			nDst++
			nSrc++
		}
	}
	return
}

// Map returns a Transformer that maps the runes in the input using the given
// mapping. Illegal bytes in the input are converted to utf8.RuneError before
// being passed to the mapping func.
func Map(mapping func(rune) rune) Transformer {
	return Transformer{mapper(mapping)}
}

type mapper func(rune) rune

func (mapper) Reset() {}

// Transform implements transform.Transformer.
func (t mapper) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	var replacement rune
	var b [utf8.UTFMax]byte

	for r, size := rune(0), 0; nSrc < len(src); {
		if r = rune(src[nSrc]); r < utf8.RuneSelf {
			if replacement = t(r); replacement < utf8.RuneSelf {
				if nDst == len(dst) {
					err = transform.ErrShortDst
					break
				}
				dst[nDst] = byte(replacement)
				nDst++
				nSrc++
				continue
			}
			size = 1
		} else if r, size = utf8.DecodeRune(src[nSrc:]); size == 1 {
			// Invalid rune.
			if !atEOF && !utf8.FullRune(src[nSrc:]) {
				err = transform.ErrShortSrc
				break
			}

			if replacement = t(utf8.RuneError); replacement == utf8.RuneError {
				if nDst+3 > len(dst) {
					err = transform.ErrShortDst
					break
				}
				dst[nDst+0] = runeErrorString[0]
				dst[nDst+1] = runeErrorString[1]
				dst[nDst+2] = runeErrorString[2]
				nDst += 3
				nSrc++
				continue
			}
		} else if replacement = t(r); replacement == r {
			if nDst+size > len(dst) {
				err = transform.ErrShortDst
				break
			}
			for i := 0; i < size; i++ {
				dst[nDst] = src[nSrc]
				nDst++
				nSrc++
			}
			continue
		}

		n := utf8.EncodeRune(b[:], replacement)

		if nDst+n > len(dst) {
			err = transform.ErrShortDst
			break
		}
		for i := 0; i < n; i++ {
			dst[nDst] = b[i]
			nDst++
		}
		nSrc += size
	}
	return
}
