// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gen.go

// Package identifier defines the contract between implementations of Encoding
// and Index by defining identifiers that uniquely identify standardized coded
// character sets (CCS) and character encoding schemes (CES), which we will
// together refer to as encodings, for which Encoding implementations provide
// converters to and from UTF-8. This package is typically only of concern to
// implementers of Indexes and Encodings.
//
// One part of the identifier is the MIB code, which is defined by IANA and
// uniquely identifies a CCS or CES. Each code is associated with data that
// references authorities, official documentation as well as aliases and MIME
// names.
//
// Not all CESs are covered by the IANA registry. The "other" string that is
// returned by ID can be used to identify other character sets or versions of
// existing ones.
//
// It is recommended that each package that provides a set of Encodings provide
// the All and Common variables to reference all supported encodings and
// commonly used subset. This allows Index implementations to include all
// available encodings without explicitly referencing or knowing about them.
package identifier
