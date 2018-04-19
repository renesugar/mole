package ui

import (
	"fmt"
	"os"

	"github.com/frankbraun/mole/config"
	"github.com/frankbraun/mole/storage"
	"github.com/frankbraun/mole/util/log"
	"github.com/frankbraun/mole/xmpp"
	"github.com/rivo/tview"
)

func (s *state) startup(hillFile string, create, dump, xmppDebug bool) tview.Primitive {
	logoWidth, logoHeight := logoSize()
	logoBox := tview.NewTextView().
		SetTextColor(tview.Styles.TertiaryTextColor)
	logoBox.SetChangedFunc(func() {
		s.app.Draw()
	})

	// create a frame for the subtitle and navigation infos
	frame := tview.NewFrame(tview.NewBox()).
		SetBorders(0, 0, 0, 0, 0, 0)

	go func() {
		s.logoDraw(logoBox)
		frame.AddText(subtitle, true, tview.AlignCenter,
			tview.Styles.SecondaryTextColor)
		s.app.Draw()
	}()

	var (
		passphrase  string
		passphrase2 string
	)
	form := tview.NewForm().
		AddPasswordField("Passphrase", "", 0, '*', func(text string) {
			passphrase = text
		})
	if create {
		form.AddPasswordField("Repeat", "", 0, '*', func(text string) {
			passphrase2 = text
		})
	}

	formFrame := tview.NewFrame(form).SetBorders(0, 1, 0, 0, 0, 0)
	formFrame.AddText("", false,
		tview.AlignLeft, tview.Styles.SecondaryTextColor)

	var (
		openString    string
		confirmString string
		confirmFunc   func()
	)
	if create {
		openString = "Create"
		confirmString = "Save"
		confirmFunc = func() {
			if passphrase != passphrase2 {
				formFrame.Clear()
				log.Println("passphrases do not match")
				formFrame.AddText("passphrases do not match", false,
					tview.AlignLeft, tview.Styles.SecondaryTextColor)
				s.app.Draw()
			} else {
				var err error
				s.hill, err = config.NewHill()
				if err != nil {
					s.fatal(err)
				}
				s.state, err = storage.Create(hillFile, passphrase,
					s.hill.Marshal())
				if err != nil {
					s.fatal(err)
				}
				s.accountAdd(xmppDebug)
			}
		}
	} else {
		openString = "Open"
		confirmString = "Login"
		confirmFunc = func() {
			var (
				data []byte
				err  error
			)
			s.state, data, err = storage.Open(hillFile, passphrase)
			if err != nil {
				formFrame.Clear()
				log.Println(err)
				formFrame.AddText(err.Error(), false, tview.AlignLeft,
					tview.Styles.SecondaryTextColor)
				s.app.Draw()
				return
			}
			s.hill, err = config.Unmarshal(data)
			if err != nil {
				s.fatal(err) // should never happen
			}
			if dump {
				s.app.Suspend(func() {
					fmt.Println(string(s.hill.MarshalIndent()))
					os.Exit(0)
				})
			}
			account := s.hill.LastAccount()
			if account == nil {
				s.accountAdd(xmppDebug)
				return
			}
			// TODO: move somewhere else
			send := make(chan string)
			recv := make(chan string)
			status := fmt.Sprintf("opening XMPP connection for '%s'...", account.Username)
			formFrame.Clear()
			log.Println(status)
			formFrame.AddText(status, false,
				tview.AlignLeft, tview.Styles.SecondaryTextColor)
			s.app.Draw()
			if err := xmpp.Start(account, send, recv, xmppDebug); err != nil {
				s.fatal(err)
			}
			log.Println("established.")
			s.main(send, recv)
		}
	}

	form.AddButton(confirmString, confirmFunc).
		AddButton("Abort", func() {
			s.app.Stop()
		}).
		SetBorder(true).SetTitle(openString + " " + hillFile).
		SetTitleAlign(tview.AlignLeft)

	// create a flex layout that centers the logo and subtitle
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 2, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(logoBox, logoWidth, 1, false).
			AddItem(tview.NewBox(), 0, 1, false), logoHeight, 0, false).
		AddItem(frame, 0, 3, false).
		AddItem(formFrame, 11, 0, true)

	return flex
}

func (s *state) setup(hillFile string, xmppDebug bool) tview.Primitive {
	log.Println("setup()")
	return s.startup(hillFile, true, false, xmppDebug)
}

func (s *state) login(hillFile string, dump, xmppDebug bool) tview.Primitive {
	log.Println("login()")
	return s.startup(hillFile, false, dump, xmppDebug)
}
