/*
 * govis: unicode aware vis(3) encoding implementation
 * Copyright (C) 2017 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package govis

import (
	"fmt"
	"unicode"
)

func isunsafe(ch rune) bool {
	return ch == '\b' || ch == '\007' || ch == '\r'
}

func isglob(ch rune) bool {
	return ch == '*' || ch == '?' || ch == '[' || ch == '#'
}

// ishttp is defined by RFC 1808.
func ishttp(ch rune) bool {
	// RFC1808 does not really consider characters outside of ASCII, so just to
	// be safe always treat characters outside the ASCII character set as "not
	// HTTP".
	if ch > unicode.MaxASCII {
		return false
	}

	return unicode.IsDigit(ch) || unicode.IsLetter(ch) ||
		// Safe characters.
		ch == '$' || ch == '-' || ch == '_' || ch == '.' || ch == '+' ||
		// Extra characters.
		ch == '!' || ch == '*' || ch == '\'' || ch == '(' ||
		ch == ')' || ch == ','
}

func isgraph(ch rune) bool {
	return unicode.IsGraphic(ch) && !unicode.IsSpace(ch) && ch <= unicode.MaxASCII
}

// vis converts a single *byte* into its encoding. While Go supports the
// concept of runes (and thus native utf-8 parsing), in order to make sure that
// the bit-stream will be completely maintained through an Unvis(Vis(...))
// round-trip. The downside is that Vis() will never output unicode -- but on
// the plus side this is actually a benefit on the encoding side (it will
// always work with the simple unvis(3) implementation). It also means that we
// don't have to worry about different multi-byte encodings.
func vis(b byte, flag VisFlag) (string, error) {
	// Treat the single-byte character as a rune.
	ch := rune(b)

	// XXX: This is quite a horrible thing to support.
	if flag&VisHTTPStyle == VisHTTPStyle {
		if !ishttp(ch) {
			return "%" + fmt.Sprintf("%.2X", ch), nil
		}
	}

	// Figure out if the character doesn't need to be encoded. Effectively, we
	// encode most "normal" (graphical) characters as themselves unless we have
	// been specifically asked not to. Note though that we *ALWAYS* encode
	// everything outside ASCII.
	// TODO: Switch this to much more logical code.

	if ch > unicode.MaxASCII {
		/* ... */
	} else if flag&VisGlob == VisGlob && isglob(ch) {
		/* ... */
	} else if isgraph(ch) ||
		(flag&VisSpace != VisSpace && ch == ' ') ||
		(flag&VisTab != VisTab && ch == '\t') ||
		(flag&VisNewline != VisNewline && ch == '\n') ||
		(flag&VisSafe != 0 && isunsafe(ch)) {

		encoded := string(ch)
		if ch == '\\' && flag&VisNoSlash == 0 {
			encoded += "\\"
		}
		return encoded, nil
	}

	// Try to use C-style escapes first.
	if flag&VisCStyle == VisCStyle {
		switch ch {
		case ' ':
			return "\\s", nil
		case '\n':
			return "\\n", nil
		case '\r':
			return "\\r", nil
		case '\b':
			return "\\b", nil
		case '\a':
			return "\\a", nil
		case '\v':
			return "\\v", nil
		case '\t':
			return "\\t", nil
		case '\f':
			return "\\f", nil
		case '\x00':
			// Output octal just to be safe.
			return "\\000", nil
		}
	}

	// For graphical characters we generate octal output (and also if it's
	// being forced by the caller's flags). Also spaces should always be
	// encoded as octal.
	if flag&VisOctal == VisOctal || isgraph(ch) || ch&0x7f == ' ' {
		// Always output three-character octal just to be safe.
		return fmt.Sprintf("\\%.3o", ch), nil
	}

	// Now we have to output meta or ctrl escapes. As far as I can tell, this
	// is not actually defined by any standard -- so this logic is basically
	// copied from the original vis(3) implementation. Hopefully nobody
	// actually relies on this (octal and hex are better).

	encoded := ""
	if flag&VisNoSlash == 0 {
		encoded += "\\"
	}

	// Meta characters have 0x80 set, but are otherwise identical to control
	// characters.
	if b&0x80 != 0 {
		b &= 0x7f
		encoded += "M"
	}

	if unicode.IsControl(rune(b)) {
		encoded += "^"
		if b == 0x7f {
			encoded += "?"
		} else {
			encoded += fmt.Sprintf("%c", b+'@')
		}
	} else {
		encoded += fmt.Sprintf("-%c", b)
	}

	return encoded, nil
}

// Vis encodes the provided string to a BSD-compatible encoding using BSD's
// vis() flags. However, it will correctly handle multi-byte encoding (which is
// not done properly by BSD's vis implementation).
func Vis(src string, flag VisFlag) (string, error) {
	if flag&visMask != flag {
		return "", fmt.Errorf("vis: flag %q contains unknown or unsupported flags", flag)
	}

	output := ""
	for _, ch := range []byte(src) {
		encodedCh, err := vis(ch, flag)
		if err != nil {
			return "", err
		}
		output += encodedCh
	}

	return output, nil
}
