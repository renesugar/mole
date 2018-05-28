// Copyright 2016 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"sync"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

// SimCell represents a simulated screen cell.  The purpose of this
// is to track on screen content.
type SimCell struct {
	// Bytes is the actual character bytes.  Normally this is
	// rune data, but it could be be data in another encoding system.
	Bytes []byte

	// Style is the style used to display the data.
	Style Style

	// Runes is the list of runes, unadulterated, in UTF-8.
	Runes []rune
}

type simscreen struct {
	physw int
	physh int
	fini  bool
	style Style
	evch  chan Event
	quit  chan struct{}

	front     []SimCell
	back      CellBuffer
	clear     bool
	cursorx   int
	cursory   int
	cursorvis bool
	mouse     bool
	charset   string
	encoder   transform.Transformer
	decoder   transform.Transformer
	fillchar  rune
	fillstyle Style
	fallback  map[rune]string

	sync.Mutex
}

func (s *simscreen) Init() error {
	s.evch = make(chan Event, 10)
	s.fillchar = 'X'
	s.fillstyle = StyleDefault
	s.mouse = false
	s.physw = 80
	s.physh = 25
	s.cursorx = -1
	s.cursory = -1
	s.style = StyleDefault

	if enc := GetEncoding(s.charset); enc != nil {
		s.encoder = enc.NewEncoder()
		s.decoder = enc.NewDecoder()
	} else {
		return ErrNoCharset
	}

	s.front = make([]SimCell, s.physw*s.physh)
	s.back.Resize(80, 25)

	// default fallbacks
	s.fallback = make(map[rune]string)
	for k, v := range RuneFallbacks {
		s.fallback[k] = v
	}
	return nil
}

func (s *simscreen) Fini() {
	s.Lock()
	s.fini = true
	s.back.Resize(0, 0)
	s.Unlock()
	if s.quit != nil {
		close(s.quit)
	}
	s.physw = 0
	s.physh = 0
	s.front = nil
}

func (s *simscreen) SetStyle(style Style) {
	s.Lock()
	s.style = style
	s.Unlock()
}

func (s *simscreen) Clear() {
	s.Fill(' ', s.style)
}

func (s *simscreen) Fill(r rune, style Style) {
	s.Lock()
	s.back.Fill(r, style)
	s.Unlock()
}

func (s *simscreen) SetCell(x, y int, style Style, ch ...rune) {

	if len(ch) > 0 {
		s.SetContent(x, y, ch[0], ch[1:], style)
	} else {
		s.SetContent(x, y, ' ', nil, style)
	}
}

func (s *simscreen) SetContent(x, y int, mainc rune, combc []rune, st Style) {

	s.Lock()
	s.back.SetContent(x, y, mainc, combc, st)
	s.Unlock()
}

func (s *simscreen) GetContent(x, y int) (rune, []rune, Style, int) {
	var mainc rune
	var combc []rune
	var style Style
	var width int
	s.Lock()
	mainc, combc, style, width = s.back.GetContent(x, y)
	s.Unlock()
	return mainc, combc, style, width
}

func (s *simscreen) drawCell(x, y int) int {

	mainc, combc, style, width := s.back.GetContent(x, y)
	if !s.back.Dirty(x, y) {
		return width
	}
	if x >= s.physw || y >= s.physh || x < 0 || y < 0 {
		return width
	}
	simc := &s.front[(y*s.physw)+x]

	if style == StyleDefault {
		style = s.style
	}
	simc.Style = style
	simc.Runes = append([]rune{mainc}, combc...)

	// now emit runes - taking care to not overrun width with a
	// wide character, and to ensure that we emit exactly one regular
	// character followed up by any residual combing characters

	simc.Bytes = nil

	if x > s.physw-width {
		simc.Runes = []rune{' '}
		simc.Bytes = []byte{' '}
		return width
	}

	lbuf := make([]byte, 12)
	ubuf := make([]byte, 12)
	nout := 0

	for _, r := range simc.Runes {

		l := utf8.EncodeRune(ubuf, r)

		nout, _, _ = s.encoder.Transform(lbuf, ubuf[:l], true)

		if nout == 0 || lbuf[0] == '\x1a' {

			// skip combining

			if subst, ok := s.fallback[r]; ok {
				simc.Bytes = append(simc.Bytes,
					[]byte(subst)...)

			} else if r >= ' ' && r <= '~' {
				simc.Bytes = append(simc.Bytes, byte(r))

			} else if simc.Bytes == nil {
				simc.Bytes = append(simc.Bytes, '?')
			}
		} else {
			simc.Bytes = append(simc.Bytes, lbuf[:nout]...)
		}
	}
	s.back.SetDirty(x, y, false)
	return width
}

func (s *simscreen) ShowCursor(x, y int) {
	s.Lock()
	s.cursorx, s.cursory = x, y
	s.showCursor()
	s.Unlock()
}

func (s *simscreen) HideCursor() {
	s.ShowCursor(-1, -1)
}

func (s *simscreen) showCursor() {

	x, y := s.cursorx, s.cursory
	if x < 0 || y < 0 || x >= s.physw || y >= s.physh {
		s.cursorvis = false
	} else {
		s.cursorvis = true
	}
}

func (s *simscreen) hideCursor() {
	// does not update cursor position
	s.cursorvis = false
}

func (s *simscreen) Show() {
	s.Lock()
	s.resize()
	s.draw()
	s.Unlock()
}

func (s *simscreen) clearScreen() {
	// We emulate a hardware clear by filling with a specific pattern
	for i := range s.front {
		s.front[i].Style = s.fillstyle
		s.front[i].Runes = []rune{s.fillchar}
		s.front[i].Bytes = []byte{byte(s.fillchar)}
	}
	s.clear = false
}

func (s *simscreen) draw() {
	s.hideCursor()
	if s.clear {
		s.clearScreen()
	}

	w, h := s.back.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			width := s.drawCell(x, y)
			x += width - 1
		}
	}
	s.showCursor()
}

func (s *simscreen) EnableMouse() {
	s.mouse = true
}

func (s *simscreen) DisableMouse() {
	s.mouse = false
}

func (s *simscreen) Size() (int, int) {
	s.Lock()
	w, h := s.back.Size()
	s.Unlock()
	return w, h
}

func (s *simscreen) resize() {
	w, h := s.physw, s.physh
	ow, oh := s.back.Size()
	if w != ow || h != oh {
		s.back.Resize(w, h)
		ev := NewEventResize(w, h)
		s.PostEvent(ev)
	}
}

func (s *simscreen) Colors() int {
	return 256
}

func (s *simscreen) PollEvent() Event {
	select {
	case <-s.quit:
		return nil
	case ev := <-s.evch:
		return ev
	}
}

func (s *simscreen) PostEventWait(ev Event) {
	s.evch <- ev
}

func (s *simscreen) PostEvent(ev Event) error {
	select {
	case s.evch <- ev:
		return nil
	default:
		return ErrEventQFull
	}
}

func (s *simscreen) Sync() {
	s.Lock()
	s.clear = true
	s.resize()
	s.back.Invalidate()
	s.draw()
	s.Unlock()
}

func (s *simscreen) CharacterSet() string {
	return s.charset
}

func (s *simscreen) RegisterRuneFallback(r rune, subst string) {
	s.Lock()
	s.fallback[r] = subst
	s.Unlock()
}

func (s *simscreen) UnregisterRuneFallback(r rune) {
	s.Lock()
	delete(s.fallback, r)
	s.Unlock()
}

func (s *simscreen) CanDisplay(r rune, checkFallbacks bool) bool {

	if enc := s.encoder; enc != nil {
		nb := make([]byte, 6)
		ob := make([]byte, 6)
		num := utf8.EncodeRune(ob, r)

		enc.Reset()
		dst, _, err := enc.Transform(nb, ob[:num], true)
		if dst != 0 && err == nil && nb[0] != '\x1A' {
			return true
		}
	}
	if !checkFallbacks {
		return false
	}
	if _, ok := s.fallback[r]; ok {
		return true
	}
	return false
}

func (s *simscreen) HasMouse() bool {
	return false
}

func (s *simscreen) Resize(int, int, int, int) {}

func (s *simscreen) HasKey(Key) bool {
	return true
}
