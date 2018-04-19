// Package util contains utility functions for Mole.
package util

import (
	"fmt"
	"os"
)

// Fatal prints err to stderr and exits the process with exit code 1.
func Fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s: error: %s\n", os.Args[0], err)
	os.Exit(1)
}
