// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package transform provides reader and writer wrappers that transform the
// bytes passing through as well as various transformations. Example
// transformations provided by other packages include normalization and
// conversion between character sets.
package transform // import "golang.org/x/text/transform"

import (
	"errors"
	"unicode/utf8"
)

var (
	// ErrShortDst means that the destination buffer was too short to
	// receive all of the transformed bytes.
	ErrShortDst = errors.New("transform: short destination buffer")

	// ErrShortSrc means that the source buffer has insufficient data to
	// complete the transformation.
	ErrShortSrc = errors.New("transform: short source buffer")

	// ErrEndOfSpan means that the input and output (the transformed input)
	// are not identical.
	ErrEndOfSpan = errors.New("transform: input and output are not identical")

	// errInconsistentByteCount means that Transform returned success (nil
	// error) but also returned nSrc inconsistent with the src argument.
	errInconsistentByteCount = errors.New("transform: inconsistent byte count returned")

	// errShortInternal means that an internal buffer is not large enough
	// to make progress and the Transform operation must be aborted.
	errShortInternal = errors.New("transform: short internal buffer")
)

// Transformer transforms bytes.
type Transformer interface {
	// Transform writes to dst the transformed bytes read from src, and
	// returns the number of dst bytes written and src bytes read. The
	// atEOF argument tells whether src represents the last bytes of the
	// input.
	//
	// Callers should always process the nDst bytes produced and account
	// for the nSrc bytes consumed before considering the error err.
	//
	// A nil error means that all of the transformed bytes (whether freshly
	// transformed from src or left over from previous Transform calls)
	// were written to dst. A nil error can be returned regardless of
	// whether atEOF is true. If err is nil then nSrc must equal len(src);
	// the converse is not necessarily true.
	//
	// ErrShortDst means that dst was too short to receive all of the
	// transformed bytes. ErrShortSrc means that src had insufficient data
	// to complete the transformation. If both conditions apply, then
	// either error may be returned. Other than the error conditions listed
	// here, implementations are free to report other errors that arise.
	Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error)

	// Reset resets the state and allows a Transformer to be reused.
	Reset()
}

// SpanningTransformer extends the Transformer interface with a Span method
// that determines how much of the input already conforms to the Transformer.
type SpanningTransformer interface {
	Transformer

	// Span returns a position in src such that transforming src[:n] results in
	// identical output src[:n] for these bytes. It does not necessarily return
	// the largest such n. The atEOF argument tells whether src represents the
	// last bytes of the input.
	//
	// Callers should always account for the n bytes consumed before
	// considering the error err.
	//
	// A nil error means that all input bytes are known to be identical to the
	// output produced by the Transformer. A nil error can be be returned
	// regardless of whether atEOF is true. If err is nil, then then n must
	// equal len(src); the converse is not necessarily true.
	//
	// ErrEndOfSpan means that the Transformer output may differ from the
	// input after n bytes. Note that n may be len(src), meaning that the output
	// would contain additional bytes after otherwise identical output.
	// ErrShortSrc means that src had insufficient data to determine whether the
	// remaining bytes would change. Other than the error conditions listed
	// here, implementations are free to report other errors that arise.
	//
	// Calling Span can modify the Transformer state as a side effect. In
	// effect, it does the transformation just as calling Transform would, only
	// without copying to a destination buffer and only up to a point it can
	// determine the input and output bytes are the same. This is obviously more
	// limited than calling Transform, but can be more efficient in terms of
	// copying and allocating buffers. Calls to Span and Transform may be
	// interleaved.
	Span(src []byte, atEOF bool) (n int, err error)
}

// NopResetter can be embedded by implementations of Transformer to add a nop
// Reset method.
type NopResetter struct{}

// Reset implements the Reset method of the Transformer interface.
func (NopResetter) Reset() {}

type nop struct{ NopResetter }

func (nop) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	n := copy(dst, src)
	if n < len(src) {
		err = ErrShortDst
	}
	return n, n, err
}

func (nop) Span(src []byte, atEOF bool) (n int, err error) {
	return len(src), nil
}

type discard struct{ NopResetter }

func (discard) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	return 0, len(src), nil
}

var (
	// Discard is a Transformer for which all Transform calls succeed
	// by consuming all bytes and writing nothing.
	Discard Transformer = discard{}

	// Nop is a SpanningTransformer that copies src to dst.
	Nop SpanningTransformer = nop{}
)

// chain is a sequence of links. A chain with N Transformers has N+1 links and
// N+1 buffers. Of those N+1 buffers, the first and last are the src and dst
// buffers given to chain.Transform and the middle N-1 buffers are intermediate
// buffers owned by the chain. The i'th link transforms bytes from the i'th
// buffer chain.link[i].b at read offset chain.link[i].p to the i+1'th buffer
// chain.link[i+1].b at write offset chain.link[i+1].n, for i in [0, N).
type chain struct {
	link []link
	err  error
	// errStart is the index at which the error occurred plus 1. Processing
	// errStart at this level at the next call to Transform. As long as
	// errStart > 0, chain will not consume any more source bytes.
	errStart int
}

func (c *chain) fatalError(errIndex int, err error) {
	if i := errIndex + 1; i > c.errStart {
		c.errStart = i
		c.err = err
	}
}

type link struct {
	t Transformer
	// b[p:n] holds the bytes to be transformed by t.
	b []byte
	p int
	n int
}

func (l *link) src() []byte {
	return l.b[l.p:l.n]
}

func (l *link) dst() []byte {
	return l.b[l.n:]
}

// Reset resets the state of Chain. It calls Reset on all the Transformers.
func (c *chain) Reset() {
	for i, l := range c.link {
		if l.t != nil {
			l.t.Reset()
		}
		c.link[i].p, c.link[i].n = 0, 0
	}
}

// TODO: make chain use Span (is going to be fun to implement!)

// Transform applies the transformers of c in sequence.
func (c *chain) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	// Set up src and dst in the chain.
	srcL := &c.link[0]
	dstL := &c.link[len(c.link)-1]
	srcL.b, srcL.p, srcL.n = src, 0, len(src)
	dstL.b, dstL.n = dst, 0
	var lastFull, needProgress bool // for detecting progress

	// i is the index of the next Transformer to apply, for i in [low, high].
	// low is the lowest index for which c.link[low] may still produce bytes.
	// high is the highest index for which c.link[high] has a Transformer.
	// The error returned by Transform determines whether to increase or
	// decrease i. We try to completely fill a buffer before converting it.
	for low, i, high := c.errStart, c.errStart, len(c.link)-2; low <= i && i <= high; {
		in, out := &c.link[i], &c.link[i+1]
		nDst, nSrc, err0 := in.t.Transform(out.dst(), in.src(), atEOF && low == i)
		out.n += nDst
		in.p += nSrc
		if i > 0 && in.p == in.n {
			in.p, in.n = 0, 0
		}
		needProgress, lastFull = lastFull, false
		switch err0 {
		case ErrShortDst:
			// Process the destination buffer next. Return if we are already
			// at the high index.
			if i == high {
				return dstL.n, srcL.p, ErrShortDst
			}
			if out.n != 0 {
				i++
				// If the Transformer at the next index is not able to process any
				// source bytes there is nothing that can be done to make progress
				// and the bytes will remain unprocessed. lastFull is used to
				// detect this and break out of the loop with a fatal error.
				lastFull = true
				continue
			}
			// The destination buffer was too small, but is completely empty.
			// Return a fatal error as this transformation can never complete.
			c.fatalError(i, errShortInternal)
		case ErrShortSrc:
			if i == 0 {
				// Save ErrShortSrc in err. All other errors take precedence.
				err = ErrShortSrc
				break
			}
			// Source bytes were depleted before filling up the destination buffer.
			// Verify we made some progress, move the remaining bytes to the errStart
			// and try to get more source bytes.
			if needProgress && nSrc == 0 || in.n-in.p == len(in.b) {
				// There were not enough source bytes to proceed while the source
				// buffer cannot hold any more bytes. Return a fatal error as this
				// transformation can never complete.
				c.fatalError(i, errShortInternal)
				break
			}
			// in.b is an internal buffer and we can make progress.
			in.p, in.n = 0, copy(in.b, in.src())
			fallthrough
		case nil:
			// if i == low, we have depleted the bytes at index i or any lower levels.
			// In that case we increase low and i. In all other cases we decrease i to
			// fetch more bytes before proceeding to the next index.
			if i > low {
				i--
				continue
			}
		default:
			c.fatalError(i, err0)
		}
		// Exhausted level low or fatal error: increase low and continue
		// to process the bytes accepted so far.
		i++
		low = i
	}

	// If c.errStart > 0, this means we found a fatal error.  We will clear
	// all upstream buffers. At this point, no more progress can be made
	// downstream, as Transform would have bailed while handling ErrShortDst.
	if c.errStart > 0 {
		for i := 1; i < c.errStart; i++ {
			c.link[i].p, c.link[i].n = 0, 0
		}
		err, c.errStart, c.err = c.err, 0, nil
	}
	return dstL.n, srcL.p, err
}

type removeF func(r rune) bool

func (removeF) Reset() {}

// Transform implements the Transformer interface.
func (t removeF) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for r, sz := rune(0), 0; len(src) > 0; src = src[sz:] {

		if r = rune(src[0]); r < utf8.RuneSelf {
			sz = 1
		} else {
			r, sz = utf8.DecodeRune(src)

			if sz == 1 {
				// Invalid rune.
				if !atEOF && !utf8.FullRune(src) {
					err = ErrShortSrc
					break
				}
				// We replace illegal bytes with RuneError. Not doing so might
				// otherwise turn a sequence of invalid UTF-8 into valid UTF-8.
				// The resulting byte sequence may subsequently contain runes
				// for which t(r) is true that were passed unnoticed.
				if !t(r) {
					if nDst+3 > len(dst) {
						err = ErrShortDst
						break
					}
					nDst += copy(dst[nDst:], "\uFFFD")
				}
				nSrc++
				continue
			}
		}

		if !t(r) {
			if nDst+sz > len(dst) {
				err = ErrShortDst
				break
			}
			nDst += copy(dst[nDst:], src[:sz])
		}
		nSrc += sz
	}
	return
}
