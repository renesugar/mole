// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run makexml.go -output xml.go

// Package cldr provides a parser for LDML and related XML formats.
// This package is intended to be used by the table generation tools
// for the various internationalization-related packages.
// As the XML types are generated from the CLDR DTD, and as the CLDR standard
// is periodically amended, this package may change considerably over time.
// This mostly means that data may appear and disappear between versions.
// That is, old code should keep compiling for newer versions, but data
// may have moved or changed.
// CLDR version 22 is the first version supported by this package.
// Older versions may not work.
package cldr // import "golang.org/x/text/unicode/cldr"
