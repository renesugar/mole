// +build windows

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
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// NewConsoleScreen returns a Screen for the Windows console associated
// with the current process.  The Screen makes use of the Windows Console
// API to display content and read events.
func NewConsoleScreen() (Screen, error) {
	return &cScreen{}, nil
}

func (s *cScreen) Init() error {
	s.evch = make(chan Event, 10)
	s.quit = make(chan struct{})
	s.scandone = make(chan struct{})

	in, e := syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if e != nil {
		return e
	}
	s.in = in
	out, e := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if e != nil {
		syscall.Close(s.in)
		return e
	}
	s.out = out

	cf, _, e := procCreateEvent.Call(
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(0))
	if cf == uintptr(0) {
		return e
	}
	s.cancelflag = syscall.Handle(cf)

	s.Lock()

	s.curx = -1
	s.cury = -1
	s.style = StyleDefault
	s.getCursorInfo(&s.ocursor)
	s.getConsoleInfo(&s.oscreen)
	s.getOutMode(&s.oomode)
	s.getInMode(&s.oimode)
	s.resize()

	s.fini = false
	s.setInMode(modeResizeEn)
	s.setOutMode(0)
	s.clearScreen(s.style)
	s.hideCursor()
	s.Unlock()
	go s.scanInput()

	return nil
}

func (s *cScreen) CharacterSet() string {
	// We are always UTF-16LE on Windows
	return "UTF-16LE"
}

func (s *cScreen) EnableMouse() {
	s.setInMode(modeResizeEn | modeMouseEn)
}

func (s *cScreen) DisableMouse() {
	s.setInMode(modeResizeEn)
}

func (s *cScreen) Fini() {
	s.Lock()
	s.style = StyleDefault
	s.curx = -1
	s.cury = -1
	s.fini = true
	s.Unlock()

	s.setCursorInfo(&s.ocursor)
	s.setInMode(s.oimode)
	s.setOutMode(s.oomode)
	s.setBufferSize(int(s.oscreen.size.x), int(s.oscreen.size.y))
	s.clearScreen(StyleDefault)
	s.setCursorPos(0, 0)
	procSetConsoleTextAttribute.Call(
		uintptr(s.out),
		uintptr(s.mapStyle(StyleDefault)))

	close(s.quit)
	procSetEvent.Call(uintptr(s.cancelflag))
	// Block until scanInput returns; this prevents a race condition on Win 8+
	// which causes syscall.Close to block until another keypress is read.
	<-s.scandone
	syscall.Close(s.in)
	syscall.Close(s.out)
}

func (s *cScreen) PostEventWait(ev Event) {
	s.evch <- ev
}

func (s *cScreen) PostEvent(ev Event) error {
	select {
	case s.evch <- ev:
		return nil
	default:
		return ErrEventQFull
	}
}

func (s *cScreen) PollEvent() Event {
	select {
	case <-s.quit:
		return nil
	case ev := <-s.evch:
		return ev
	}
}

func (s *cScreen) showCursor() {
	s.setCursorInfo(&cursorInfo{size: 100, visible: 1})
}

func (s *cScreen) hideCursor() {
	s.setCursorInfo(&cursorInfo{size: 1, visible: 0})
}

func (s *cScreen) ShowCursor(x, y int) {
	s.Lock()
	if !s.fini {
		s.curx = x
		s.cury = y
	}
	s.doCursor()
	s.Unlock()
}

func (s *cScreen) HideCursor() {
	s.ShowCursor(-1, -1)
}

func (s *cScreen) scanInput() {
	for {
		if e := s.getConsoleInput(); e != nil {
			close(s.scandone)
			return
		}
	}
}

// Windows console can display 8 characters, in either low or high intensity
func (s *cScreen) Colors() int {
	return 16
}

func (s *cScreen) SetCell(x, y int, style Style, ch ...rune) {
	if len(ch) > 0 {
		s.SetContent(x, y, ch[0], ch[1:], style)
	} else {
		s.SetContent(x, y, ' ', nil, style)
	}
}

func (s *cScreen) SetContent(x, y int, mainc rune, combc []rune, style Style) {
	s.Lock()
	if !s.fini {
		s.cells.SetContent(x, y, mainc, combc, style)
	}
	s.Unlock()
}

func (s *cScreen) GetContent(x, y int) (rune, []rune, Style, int) {
	s.Lock()
	mainc, combc, style, width := s.cells.GetContent(x, y)
	s.Unlock()
	return mainc, combc, style, width
}

func (s *cScreen) draw() {
	// allocate a scratch line bit enough for no combining chars.
	// if you have combining characters, you may pay for extra allocs.
	if s.clear {
		s.clearScreen(s.style)
		s.clear = false
		s.cells.Invalidate()
	}
	buf := make([]uint16, 0, s.w)
	wcs := buf[:]
	lstyle := Style(-1) // invalid attribute

	lx, ly := -1, -1
	ra := make([]rune, 1)

	for y := 0; y < int(s.h); y++ {
		for x := 0; x < int(s.w); x++ {
			mainc, combc, style, width := s.cells.GetContent(x, y)
			dirty := s.cells.Dirty(x, y)
			if style == StyleDefault {
				style = s.style
			}

			if !dirty || style != lstyle {
				// write out any data queued thus far
				// because we are going to skip over some
				// cells, or because we need to change styles
				s.writeString(lx, ly, lstyle, wcs)
				wcs = buf[0:0]
				lstyle = Style(-1)
				if !dirty {
					continue
				}
			}
			if x > s.w-width {
				mainc = ' '
				combc = nil
				width = 1
			}
			if len(wcs) == 0 {
				lstyle = style
				lx = x
				ly = y
			}
			ra[0] = mainc
			wcs = append(wcs, utf16.Encode(ra)...)
			if len(combc) != 0 {
				wcs = append(wcs, utf16.Encode(combc)...)
			}
			s.cells.SetDirty(x, y, false)
			x += width - 1
		}
		s.writeString(lx, ly, lstyle, wcs)
		wcs = buf[0:0]
		lstyle = Style(-1)
	}
}

func (s *cScreen) Show() {
	s.Lock()
	if !s.fini {
		s.hideCursor()
		s.resize()
		s.draw()
		s.doCursor()
	}
	s.Unlock()
}

func (s *cScreen) Sync() {
	s.Lock()
	if !s.fini {
		s.cells.Invalidate()
		s.hideCursor()
		s.resize()
		s.draw()
		s.doCursor()
	}
	s.Unlock()
}

func (s *cScreen) Size() (int, int) {
	s.Lock()
	w, h := s.w, s.h
	s.Unlock()

	return w, h
}

func (s *cScreen) resize() {
	info := consoleInfo{}
	s.getConsoleInfo(&info)

	w := int((info.win.right - info.win.left) + 1)
	h := int((info.win.bottom - info.win.top) + 1)

	if s.w == w && s.h == h {
		return
	}

	s.cells.Resize(w, h)
	s.w = w
	s.h = h

	r := rect{0, 0, int16(w - 1), int16(h - 1)}
	procSetConsoleWindowInfo.Call(
		uintptr(s.out),
		uintptr(1),
		uintptr(unsafe.Pointer(&r)))

	s.setBufferSize(w, h)

	s.PostEvent(NewEventResize(w, h))
}

func (s *cScreen) Clear() {
	s.Fill(' ', s.style)
}

func (s *cScreen) Fill(r rune, style Style) {
	s.Lock()
	if !s.fini {
		s.cells.Fill(r, style)
		s.clear = true
	}
	s.Unlock()
}

func (s *cScreen) clearScreen(style Style) {
	pos := coord{0, 0}
	attr := s.mapStyle(style)
	x, y := s.w, s.h
	scratch := uint32(0)
	count := uint32(x * y)

	procFillConsoleOutputAttribute.Call(
		uintptr(s.out),
		uintptr(attr),
		uintptr(count),
		pos.uintptr(),
		uintptr(unsafe.Pointer(&scratch)))
	procFillConsoleOutputCharacter.Call(
		uintptr(s.out),
		uintptr(' '),
		uintptr(count),
		pos.uintptr(),
		uintptr(unsafe.Pointer(&scratch)))
}

func (s *cScreen) SetStyle(style Style) {
	s.Lock()
	s.style = style
	s.Unlock()
}

// No fallback rune support, since we have Unicode.  Yay!

func (s *cScreen) RegisterRuneFallback(r rune, subst string) {
}

func (s *cScreen) UnregisterRuneFallback(r rune) {
}

func (s *cScreen) CanDisplay(r rune, checkFallbacks bool) bool {
	// We presume we can display anything -- we're Unicode.
	// (Sadly this not precisely true.  Combinings are especially
	// poorly supported under Windows.)
	return true
}

func (s *cScreen) HasMouse() bool {
	return true
}

func (s *cScreen) Resize(int, int, int, int) {}

func (s *cScreen) HasKey(k Key) bool {
	// Microsoft has codes for some keys, but they are unusual,
	// so we don't include them.  We include all the typical
	// 101, 105 key layout keys.
	valid := map[Key]bool{
		KeyBackspace: true,
		KeyTab:       true,
		KeyEscape:    true,
		KeyPause:     true,
		KeyPrint:     true,
		KeyPgUp:      true,
		KeyPgDn:      true,
		KeyEnter:     true,
		KeyEnd:       true,
		KeyHome:      true,
		KeyLeft:      true,
		KeyUp:        true,
		KeyRight:     true,
		KeyDown:      true,
		KeyInsert:    true,
		KeyDelete:    true,
		KeyF1:        true,
		KeyF2:        true,
		KeyF3:        true,
		KeyF4:        true,
		KeyF5:        true,
		KeyF6:        true,
		KeyF7:        true,
		KeyF8:        true,
		KeyF9:        true,
		KeyF10:       true,
		KeyF11:       true,
		KeyF12:       true,
		KeyRune:      true,
	}

	return valid[k]
}
