// mole is a secure XMPP client for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frankbraun/codechain/util/home"
	"github.com/frankbraun/codechain/util/lockfile"
	"github.com/frankbraun/codechain/util/log"
	"github.com/frankbraun/mole/ui"
	"github.com/frankbraun/mole/util"
	"github.com/mattn/go-xmpp"
)

const (
	defaultHillName = "mole.hill"
)

var (
	defaultHomeDir  = home.AppDataDir("mole", false)
	defaultHillFile = filepath.Join(defaultHomeDir, defaultHillName)
)

func prepareHillDir(hillFile string) error {
	if hillFile == defaultHillFile {
		log.Printf("mkdir -p %s", defaultHomeDir)
		if err := os.MkdirAll(defaultHomeDir, 0700); err != nil {
			return err
		}
	} else {
		dir := filepath.Dir(hillFile)
		fi, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("%s: is not a directory", dir)
		}
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func moleMain() error {
	// option parsing
	dump := flag.Bool("d", false, "dump hill file after decryption")
	hillFile := flag.String("f", defaultHillFile, "set hill file")
	logFile := flag.String("l", "", "set log file (for debugging only, might leak sensitive data!)")
	xmppDebug := flag.Bool("x", false, "enable XMPP debugging")
	flag.Parse()
	if flag.NArg() != 0 {
		usage()
	}
	// initialize logging framework
	if *logFile != "" {
		fp, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer func() {
			log.Printf("logfile '%s' closed.", *logFile)
			fp.Close()
		}()
		log.Std = log.NewStd(fp)
		log.Printf("logging to '%s'...", *logFile)
		// use fp as xmpp.DebugWriter
		xmpp.DebugWriter = fp
	} else if *xmppDebug {
		return fmt.Errorf("option -x requires option -l")
	}
	// prepare directory for .hill file
	if err := prepareHillDir(*hillFile); err != nil {
		return err
	}
	// create lock anchored at .hill file
	lock, err := lockfile.Create(*hillFile)
	if err != nil {
		return err
	}
	defer lock.Release()
	// start UI event loop
	return ui.Run(*hillFile, *dump, *xmppDebug)
}

func main() {
	// work around defer not working after os.Exit()
	if err := moleMain(); err != nil {
		log.Printf("fatal(): %v", err)
		util.Fatal(err)
	}
}
