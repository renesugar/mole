package ui

import (
	"fmt"

	"github.com/frankbraun/mole/config"
	"github.com/frankbraun/mole/util/log"
	"github.com/frankbraun/mole/xmpp"
	"github.com/rivo/tview"
)

func (s *state) accountAdd(xmppDebug bool) {
	log.Println("accountAdd()")
	var account config.Account
	form := tview.NewForm().
		AddInputField("Username", "user@example.com", 0, nil, func(text string) {
			account.Username = text
		}).
		// TODO: make into password fix (after pasting problem has been fixed)
		AddInputField("Password", "", 0, nil, func(text string) {
			account.Password = text
		}).
		AddInputField("Contact", "contact@example.com", 0, nil, func(text string) {
			account.Contact = text
		})

	formFrame := tview.NewFrame(form).SetBorders(0, 1, 0, 0, 0, 0)
	formFrame.AddText(mole, true, tview.AlignCenter,
		tview.Styles.TertiaryTextColor)
	formFrame.AddText("", false, tview.AlignLeft,
		tview.Styles.SecondaryTextColor)

	form.AddButton("Save", func() {
		// TODO: check account.Username
		if account.Password == "" {
			formFrame.Clear()
			formFrame.AddText(mole, true, tview.AlignCenter,
				tview.Styles.TertiaryTextColor)
			log.Println("passphrase is empty")
			formFrame.AddText("passphrase is empty", false,
				tview.AlignLeft, tview.Styles.SecondaryTextColor)
			s.app.Draw()
			return
		}
		s.hill.Accounts = append(s.hill.Accounts, account)
		if err := s.state.Save(s.hill.Marshal()); err != nil {
			s.fatal(err)
		}
		// TODO: move somewhere else
		send := make(chan string)
		recv := make(chan string)
		status := fmt.Sprintf("opening XMPP connection for '%s'...", account.Username)
		formFrame.Clear()
		formFrame.AddText(mole, true, tview.AlignCenter,
			tview.Styles.TertiaryTextColor)
		log.Println(status)
		formFrame.AddText(status, false,
			tview.AlignLeft, tview.Styles.SecondaryTextColor)
		s.app.Draw()
		if err := xmpp.Start(&account, send, recv, xmppDebug); err != nil {
			s.fatal(err)
		}
		log.Println("established.")
		s.main(send, recv)
	}).
		AddButton("Quit", func() {
			s.app.Stop()
		}).
		SetBorder(true).
		SetTitle("Add account").SetTitleAlign(tview.AlignLeft)

	s.app.SetRoot(formFrame, true).Draw()
}
