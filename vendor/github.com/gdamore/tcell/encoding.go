// Copyright 2015 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"strings"
	"sync"

	"golang.org/x/text/encoding"

	gencoding "github.com/gdamore/encoding"
)

var encodings map[string]encoding.Encoding
var encodingLk sync.Mutex
var encodingFallback EncodingFallback = EncodingFallbackFail

// EncodingFallback describes how the system behavees when the locale
// requires a character set that we do not support.  The system always
// supports UTF-8 and US-ASCII. On Windows consoles, UTF-16LE is also
// supported automatically.  Other character sets must be added using the
// RegisterEncoding API.  (A large group of nearly all of them can be
// added using the RegisterAll function in the encoding sub package.)
type EncodingFallback int

const (
	// EncodingFallbackFail behavior causes GetEncoding to fail
	// when it cannot find an encoding.
	EncodingFallbackFail = iota

	// EncodingFallbackASCII behaviore causes GetEncoding to fall back
	// to a 7-bit ASCII encoding, if no other encoding can be found.
	EncodingFallbackASCII

	// EncodingFallbackUTF8 behavior causes GetEncoding to assume
	// UTF8 can pass unmodified upon failure.  Note that this behavior
	// is not recommended, unless you are sure your terminal can cope
	// with real UTF8 sequences.
	EncodingFallbackUTF8
)

// GetEncoding is used by Screen implementors who want to locate an encoding
// for the given character set name.  Note that this will return nil for
// either the Unicode (UTF-8) or ASCII encodings, since we don't use
// encodings for them but instead have our own native methods.
func GetEncoding(charset string) encoding.Encoding {
	charset = strings.ToLower(charset)
	encodingLk.Lock()
	defer encodingLk.Unlock()
	if enc, ok := encodings[charset]; ok {
		return enc
	}
	switch encodingFallback {
	case EncodingFallbackASCII:
		return gencoding.ASCII
	case EncodingFallbackUTF8:
		return encoding.Nop
	}
	return nil
}

func init() {
	// We always support UTF-8 and ASCII.
	encodings = make(map[string]encoding.Encoding)
	encodings["utf-8"] = gencoding.UTF8
	encodings["utf8"] = gencoding.UTF8
	encodings["us-ascii"] = gencoding.ASCII
	encodings["ascii"] = gencoding.ASCII
	encodings["iso646"] = gencoding.ASCII
}
