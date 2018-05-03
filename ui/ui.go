// Package ui implements the text user interface of Mole.
package ui

import (
	"os"

	"github.com/frankbraun/codechain/util/log"
	"github.com/frankbraun/mole/config"
	"github.com/frankbraun/mole/storage"
	"github.com/frankbraun/mole/util"
	"github.com/rivo/tview"
)

// state of UI.
type state struct {
	app   *tview.Application // the "application"
	state *storage.State     // state of storage backend
	hill  *config.Hill       // entire date of running Mole instance
}

func newState() *state {
	return &state{app: tview.NewApplication()}
}

// fatal function to abort running application.
func (s *state) fatal(err error) {
	log.Printf("fatal(): %v", err)
	if !s.app.Suspend(func() {
		// app suspended -> exit
		util.Fatal(err)
	}) {
		// app was already suspended -> exit
		util.Fatal(err)
	}
}

// Run user interface on hillFile.
func Run(hillFile string, dump, xmppDebug bool) error {
	s := newState()
	var root tview.Primitive
	if _, err := os.Stat(hillFile); err != nil {
		root = s.setup(hillFile, xmppDebug) // create .hill file
	} else {
		root = s.login(hillFile, dump, xmppDebug) // open .hill file
	}
	return s.app.SetRoot(root, true).Run()
}
