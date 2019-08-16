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
	"strconv"
	"unicode"
)

// unvisParser stores the current state of the token parser.
type unvisParser struct {
	tokens []rune
	idx    int
	flag   VisFlag
}

// Next moves the index to the next character.
func (p *unvisParser) Next() {
	p.idx++
}

// Peek gets the current token.
func (p *unvisParser) Peek() (rune, error) {
	if p.idx >= len(p.tokens) {
		return unicode.ReplacementChar, fmt.Errorf("tried to read past end of token list")
	}
	return p.tokens[p.idx], nil
}

// End returns whether all of the tokens have been consumed.
func (p *unvisParser) End() bool {
	return p.idx >= len(p.tokens)
}

func newParser(input string, flag VisFlag) *unvisParser {
	return &unvisParser{
		tokens: []rune(input),
		idx:    0,
		flag:   flag,
	}
}

// While a recursive descent parser is overkill for parsing simple escape
// codes, this is IMO much easier to read than the ugly 80s coroutine code used
// by the original unvis(3) parser. Here's the EBNF for an unvis sequence:
//
// <input>           ::= (<rune>)*
// <rune>            ::= ("\" <escape-sequence>) | ("%" <escape-hex>) | <plain-rune>
// <plain-rune>      ::= any rune
// <escape-sequence> ::= ("x" <escape-hex>) | ("M" <escape-meta>) | ("^" <escape-ctrl) | <escape-cstyle> | <escape-octal>
// <escape-meta>     ::= ("-" <escape-meta1>) | ("^" <escape-ctrl>)
// <escape-meta1>    ::= any rune
// <escape-ctrl>     ::= "?" | any rune
// <escape-cstyle>   ::= "\" | "n" | "r" | "b" | "a" | "v" | "t" | "f"
// <escape-hex>      ::= [0-9a-f] [0-9a-f]
// <escape-octal>    ::= [0-7] ([0-7] ([0-7])?)?

func unvisPlainRune(p *unvisParser) ([]byte, error) {
	ch, err := p.Peek()
	if err != nil {
		return nil, fmt.Errorf("plain rune: %c", ch)
	}
	p.Next()

	// XXX: Maybe we should not be converting to runes and then back to strings
	//      here. Are we sure that the byte-for-byte representation is the
	//      same? If the bytes change, then using these strings for paths will
	//      break...

	str := string(ch)
	return []byte(str), nil
}

func unvisEscapeCStyle(p *unvisParser) ([]byte, error) {
	ch, err := p.Peek()
	if err != nil {
		return nil, fmt.Errorf("escape hex: %s", err)
	}

	output := ""
	switch ch {
	case 'n':
		output = "\n"
	case 'r':
		output = "\r"
	case 'b':
		output = "\b"
	case 'a':
		output = "\x07"
	case 'v':
		output = "\v"
	case 't':
		output = "\t"
	case 'f':
		output = "\f"
	case 's':
		output = " "
	case 'E':
		output = "\x1b"
	case '\n':
		// Hidden newline.
	case '$':
		// Hidden marker.
	default:
		// XXX: We should probably allow falling through and return "\" here...
		return nil, fmt.Errorf("escape cstyle: unknown escape character: %q", ch)
	}

	p.Next()
	return []byte(output), nil
}

func unvisEscapeDigits(p *unvisParser, base int, force bool) ([]byte, error) {
	var code int

	for i := int(0xFF); i > 0; i /= base {
		ch, err := p.Peek()
		if err != nil {
			if !force && i != 0xFF {
				break
			}
			return nil, fmt.Errorf("escape base %d: %s", base, err)
		}

		digit, err := strconv.ParseInt(string(ch), base, 8)
		if err != nil {
			if !force && i != 0xFF {
				break
			}
			return nil, fmt.Errorf("escape base %d: could not parse digit: %s", base, err)
		}

		code = (code * base) + int(digit)
		p.Next()
	}

	if code > unicode.MaxLatin1 {
		return nil, fmt.Errorf("escape base %d: code %q outside latin-1 encoding", base, code)
	}

	char := byte(code & 0xFF)
	return []byte{char}, nil
}

func unvisEscapeCtrl(p *unvisParser, mask byte) ([]byte, error) {
	ch, err := p.Peek()
	if err != nil {
		return nil, fmt.Errorf("escape ctrl: %s", err)
	}
	if ch > unicode.MaxLatin1 {
		return nil, fmt.Errorf("escape ctrl: code %q outside latin-1 encoding", ch)
	}

	char := byte(ch) & 0x1f
	if ch == '?' {
		char = 0x7f
	}

	p.Next()
	return []byte{mask | char}, nil
}

func unvisEscapeMeta(p *unvisParser) ([]byte, error) {
	ch, err := p.Peek()
	if err != nil {
		return nil, fmt.Errorf("escape meta: %s", err)
	}

	mask := byte(0x80)

	switch ch {
	case '^':
		// The same as "\^..." except we apply a mask.
		p.Next()
		return unvisEscapeCtrl(p, mask)

	case '-':
		p.Next()

		ch, err := p.Peek()
		if err != nil {
			return nil, fmt.Errorf("escape meta1: %s", err)
		}
		if ch > unicode.MaxLatin1 {
			return nil, fmt.Errorf("escape meta1: code %q outside latin-1 encoding", ch)
		}

		// Add mask to character.
		p.Next()
		return []byte{mask | byte(ch)}, nil
	}

	return nil, fmt.Errorf("escape meta: unknown escape char: %s", err)
}

func unvisEscapeSequence(p *unvisParser) ([]byte, error) {
	ch, err := p.Peek()
	if err != nil {
		return nil, fmt.Errorf("escape sequence: %s", err)
	}

	switch ch {
	case '\\':
		p.Next()
		return []byte("\\"), nil

	case '0', '1', '2', '3', '4', '5', '6', '7':
		return unvisEscapeDigits(p, 8, false)

	case 'x':
		p.Next()
		return unvisEscapeDigits(p, 16, true)

	case '^':
		p.Next()
		return unvisEscapeCtrl(p, 0x00)

	case 'M':
		p.Next()
		return unvisEscapeMeta(p)

	default:
		return unvisEscapeCStyle(p)
	}
}

func unvisRune(p *unvisParser) ([]byte, error) {
	ch, err := p.Peek()
	if err != nil {
		return nil, fmt.Errorf("rune: %s", err)
	}

	switch ch {
	case '\\':
		p.Next()
		return unvisEscapeSequence(p)

	case '%':
		// % HEX HEX only applies to HTTPStyle encodings.
		if p.flag&VisHTTPStyle == VisHTTPStyle {
			p.Next()
			return unvisEscapeDigits(p, 16, true)
		}
		fallthrough

	default:
		return unvisPlainRune(p)
	}
}

func unvis(p *unvisParser) (string, error) {
	var output []byte
	for !p.End() {
		ch, err := unvisRune(p)
		if err != nil {
			return "", fmt.Errorf("input: %s", err)
		}
		output = append(output, ch...)
	}
	return string(output), nil
}

// Unvis takes a string formatted with the given Vis flags (though only the
// VisHTTPStyle flag is checked) and output the un-encoded version of the
// encoded string. An error is returned if any escape sequences in the input
// string were invalid.
func Unvis(input string, flag VisFlag) (string, error) {
	// TODO: Check all of the VisFlag bits.
	p := newParser(input, flag)
	output, err := unvis(p)
	if err != nil {
		return "", fmt.Errorf("unvis: %s", err)
	}
	if !p.End() {
		return "", fmt.Errorf("unvis: trailing characters at end of input")
	}
	return output, nil
}
