package ui

import (
	"io"
	"strings"
	"time"
)

const logo = `
                 _
 _ __ ___   ___ | | ___
| '_ ' _ \ / _ \| |/ _ \
| | | | | | (_) | |  __/
|_| |_| |_|\___/|_|\___|

`

const (
	subtitle = "Mole — secure XMPP client for the terminal"
	mole     = "Mole"
)

func logoSize() (int, int) {
	lines := strings.Split(logo, "\n")
	w := 0
	h := len(lines)
	for _, line := range lines {
		if len(line) > w {
			w = len(line)
		}
	}
	return w, h
}

func (s *state) logoDraw(w io.Writer) {
	var ba [1]byte
	for _, b := range []byte(logo) {
		if b == '\n' {
			time.Sleep(80 * time.Millisecond)
		} else {
			time.Sleep(8 * time.Millisecond)
		}
		ba[0] = b
		if _, err := w.Write(ba[:]); err != nil {
			s.fatal(err)
		}
	}
}
