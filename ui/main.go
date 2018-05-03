package ui

import (
	"io"

	"github.com/frankbraun/codechain/util/log"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// TODO: take care of syncing/mutexes!
// TODO: check security implications of DynamicColors!

func (s *state) main(send, recv chan string) {
	log.Println("main()")
	account := s.hill.LastAccount()

	contactList := tview.NewList().
		ShowSecondaryText(false).
		AddItem(account.Contact, "", 0, nil)
	contactList.SetBorder(true)

	chatRecord := tview.NewTextView()
	chatRecord.SetBorder(true)
	chatRecord.SetDynamicColors(true)
	chatRecord.SetChangedFunc(func() {
		s.app.Draw()
	})
	first := true
	go func() {
		for {
			msg := <-recv
			if !first {
				msg = "\n" + msg
			} else {
				first = false
			}
			_, err := io.WriteString(chatRecord, msg)
			if err != nil {
				s.fatal(err)
			}
		}
	}()

	innerFlex := tview.NewFlex().
		AddItem(contactList, 0, 2, false).
		AddItem(chatRecord, 0, 8, false)

	frame := tview.NewFrame(innerFlex).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(mole, true, tview.AlignCenter, tview.Styles.TertiaryTextColor).
		AddText(account.Username, true, tview.AlignLeft, tview.Styles.SecondaryTextColor).
		AddText("", false, tview.AlignLeft, tview.Styles.SecondaryTextColor)

	inputField := tview.NewInputField()
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			msg := inputField.GetText()
			if msg != "" {
				send <- msg
				inputField.SetText("")
				msg = "[blue]" + msg + "[-]"
				if !first {
					msg = "\n" + msg
				} else {
					first = false
				}
				_, err := io.WriteString(chatRecord, msg)
				if err != nil {
					s.fatal(err)
				}
			}
		}
	})

	outerFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(frame, 0, 1, false).
		AddItem(inputField, 1, 0, true)

	s.app.SetRoot(outerFlex, true).Draw()
}
