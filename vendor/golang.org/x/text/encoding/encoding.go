// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package encoding defines an interface for character encodings, such as Shift
// JIS and Windows 1252, that can convert to and from UTF-8.
//
// Encoding implementations are provided in other packages, such as
// golang.org/x/text/encoding/charmap and
// golang.org/x/text/encoding/japanese.
package encoding // import "golang.org/x/text/encoding"

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

// TODO:
// - There seems to be some inconsistency in when decoders return errors
//   and when not. Also documentation seems to suggest they shouldn't return
//   errors at all (except for UTF-16).
// - Encoders seem to rely on or at least benefit from the input being in NFC
//   normal form. Perhaps add an example how users could prepare their output.

// Encoding is a character set encoding that can be transformed to and from
// UTF-8.
type Encoding interface {
	// NewDecoder returns a Decoder.
	NewDecoder() *Decoder

	// NewEncoder returns an Encoder.
	NewEncoder() *Encoder
}

// A Decoder converts bytes to UTF-8. It implements transform.Transformer.
//
// Transforming source bytes that are not of that encoding will not result in an
// error per se. Each byte that cannot be transcoded will be represented in the
// output by the UTF-8 encoding of '\uFFFD', the replacement rune.
type Decoder struct {
	transform.Transformer

	// This forces external creators of Decoders to use names in struct
	// initializers, allowing for future extendibility without having to break
	// code.
	_ struct{}
}

// An Encoder converts bytes from UTF-8. It implements transform.Transformer.
//
// Each rune that cannot be transcoded will result in an error. In this case,
// the transform will consume all source byte up to, not including the offending
// rune. Transforming source bytes that are not valid UTF-8 will be replaced by
// `\uFFFD`. To return early with an error instead, use transform.Chain to
// preprocess the data with a UTF8Validator.
type Encoder struct {
	transform.Transformer

	// This forces external creators of Encoders to use names in struct
	// initializers, allowing for future extendibility without having to break
	// code.
	_ struct{}
}

// Nop is the nop encoding. Its transformed bytes are the same as the source
// bytes; it does not replace invalid UTF-8 sequences.
var Nop Encoding = nop{}

type nop struct{}

func (nop) NewDecoder() *Decoder {
	return &Decoder{Transformer: transform.Nop}
}
func (nop) NewEncoder() *Encoder {
	return &Encoder{Transformer: transform.Nop}
}

type replacement struct{}

func (replacement) NewDecoder() *Decoder {
	return &Decoder{Transformer: replacementDecoder{}}
}

func (replacement) NewEncoder() *Encoder {
	return &Encoder{Transformer: replacementEncoder{}}
}

type replacementDecoder struct{ transform.NopResetter }

func (replacementDecoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if len(dst) < 3 {
		return 0, 0, transform.ErrShortDst
	}
	if atEOF {
		const fffd = "\ufffd"
		dst[0] = fffd[0]
		dst[1] = fffd[1]
		dst[2] = fffd[2]
		nDst = 3
	}
	return nDst, len(src), nil
}

type replacementEncoder struct{ transform.NopResetter }

func (replacementEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	r, size := rune(0), 0

	for ; nSrc < len(src); nSrc += size {
		r = rune(src[nSrc])

		// Decode a 1-byte rune.
		if r < utf8.RuneSelf {
			size = 1

		} else {
			// Decode a multi-byte rune.
			r, size = utf8.DecodeRune(src[nSrc:])
			if size == 1 {
				// All valid runes of size 1 (those below utf8.RuneSelf) were
				// handled above. We have invalid UTF-8 or we haven't seen the
				// full character yet.
				if !atEOF && !utf8.FullRune(src[nSrc:]) {
					err = transform.ErrShortSrc
					break
				}
				r = '\ufffd'
			}
		}

		if nDst+utf8.RuneLen(r) > len(dst) {
			err = transform.ErrShortDst
			break
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
	}
	return nDst, nSrc, err
}

type errorHandler struct {
	*Encoder
	handler func(dst []byte, r rune, err repertoireError) (n int, ok bool)
}

// TODO: consider making this error public in some form.
type repertoireError interface {
	Replacement() byte
}

func (h errorHandler) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	nDst, nSrc, err = h.Transformer.Transform(dst, src, atEOF)
	for err != nil {
		rerr, ok := err.(repertoireError)
		if !ok {
			return nDst, nSrc, err
		}
		r, sz := utf8.DecodeRune(src[nSrc:])
		n, ok := h.handler(dst[nDst:], r, rerr)
		if !ok {
			return nDst, nSrc, transform.ErrShortDst
		}
		err = nil
		nDst += n
		if nSrc += sz; nSrc < len(src) {
			var dn, sn int
			dn, sn, err = h.Transformer.Transform(dst[nDst:], src[nSrc:], atEOF)
			nDst += dn
			nSrc += sn
		}
	}
	return nDst, nSrc, err
}

// ErrInvalidUTF8 means that a transformer encountered invalid UTF-8.
var ErrInvalidUTF8 = errors.New("encoding: invalid UTF-8")

// UTF8Validator is a transformer that returns ErrInvalidUTF8 on the first
// input byte that is not valid UTF-8.
var UTF8Validator transform.Transformer = utf8Validator{}

type utf8Validator struct{ transform.NopResetter }

func (utf8Validator) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	for i := 0; i < n; {
		if c := src[i]; c < utf8.RuneSelf {
			dst[i] = c
			i++
			continue
		}
		_, size := utf8.DecodeRune(src[i:])
		if size == 1 {
			// All valid runes of size 1 (those below utf8.RuneSelf) were
			// handled above. We have invalid UTF-8 or we haven't seen the
			// full character yet.
			err = ErrInvalidUTF8
			if !atEOF && !utf8.FullRune(src[i:]) {
				err = transform.ErrShortSrc
			}
			return i, i, err
		}
		if i+size > len(dst) {
			return i, i, transform.ErrShortDst
		}
		for ; size > 0; size-- {
			dst[i] = src[i]
			i++
		}
	}
	if len(src) > len(dst) {
		err = transform.ErrShortDst
	}
	return n, n, err
}
