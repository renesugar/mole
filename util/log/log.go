// Package log implements the Mole logging framework.
package log

import (
	"fmt"
	"log"
)

// Std is the standard logger. The default is nil (nothing is logged).
var Std *log.Logger

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	if Std == nil {
		return
	}
	Std.Output(2, fmt.Sprintf(format, v...))
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Println.
func Println(v ...interface{}) {
	if Std == nil {
		return
	}
	Std.Output(2, fmt.Sprintln(v...))
}
