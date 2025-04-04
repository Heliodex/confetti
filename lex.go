// implementation of the Confetti language

package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var lineTerminators = [...]rune{'\r', '\n', '\u000a', '\u000b', '\u000c', '\u000d', '\u0085', '\u2028', '\u2029'}

func isLineTerminator(r rune) bool {
	return slices.Contains(lineTerminators[:], r)
}

// all unicode chars with whitespace property
func isWhitespace(r rune) bool {
	return !isLineTerminator(r) && unicode.IsSpace(r)
}

// characters not in any Unicode category
func isUnassigned(r rune) bool {
	return r >= 0x40000 && r <= 0xEFFFF
}

// surrogate, private use, unassigned
func isForbidden(r rune) bool {
	return unicode.Is(unicode.Cc, r) && !isLineTerminator(r) && !isWhitespace(r) || unicode.Is(unicode.Cs, r) || r > 0x10FFFF || isUnassigned(r)
}

var reserved = [...]rune{'"', '#', ';', '{', '}'}

func isReserved(r rune) bool {
	return slices.Contains(reserved[:], r)
}

type Stream struct {
	src []rune
	pos int
}

func (s *Stream) reading() bool {
	return s.pos < len(s.src)
}

var (
	errEOF       = errors.New("EOF")
	errForbidden = errors.New("illegal character")
)

func (s *Stream) current() (c rune, err error) {
	if s.pos >= len(s.src) {
		// panic("EOF")
		return 0, errEOF
	}

	c = s.src[s.pos]
	if isForbidden(c) {
		// get illegal character as U+XXXX
		if c < 0x10000 {
			return 0, fmt.Errorf("%w U+%04X", errForbidden, c)
		}
		return 0, fmt.Errorf("%w U+%X", errForbidden, c)
	}

	return
}

func (s *Stream) next(n int) rune {
	if s.pos+n < len(s.src) {
		return s.src[s.pos+n]
	}
	return 0
}

// tokens
type TokenType uint8

const (
	TokArgument TokenType = iota
	TokNewline
	TokLineContinuation
	TokWhitespace
	TokComment
	TokSemicolon
	TokOpenBrace
	TokCloseBrace
)

type Token struct {
	Type    TokenType
	Content string
}

// A directive “argument” shall be a sequence of one or more characters from the argument character set. The argument character set shall consist of any Unicode scalar value excluding characters from the white space, line terminator, reserved punctuator, and forbidden character sets.
func argumentOk(r rune) bool {
	return !isWhitespace(r) && !isLineTerminator(r) && !isReserved(r) && !isForbidden(r)
}

func quotedArgumentOk(r rune) bool {
	return !isLineTerminator(r) && r != '"' && !isForbidden(r)
}

func tripleQuotedArgumentOk(r rune) bool {
	return !isForbidden(r) && r != '"'
}

var (
	errIncompleteEscape = errors.New("incomplete escape sequence")
	errIllegalEscape    = errors.New("illegal escape character")
	errUnclosedQuoted   = errors.New("unclosed quoted")
)

func checkEscape(s *Stream, c rune, quoted uint8) (r rune, err error) {
	if c != '\\' {
		return c, nil
	}

	s.pos++
	if c, err = s.current(); err != nil {
		if errors.Is(err, errForbidden) {
			return 0, errIllegalEscape
		} else if quoted > 0 {
			return 0, errIncompleteEscape
		}
		return 0, errIllegalEscape
	} else if isWhitespace(c) || isLineTerminator(c) {
		if quoted == 3 {
			if isLineTerminator(c) {
				return 0, errIncompleteEscape
			}
			return 0, errIllegalEscape
		} else if quoted == 0 || quoted == 1 && !isLineTerminator(c) {
			return 0, errIllegalEscape
		}
		return // r = 0 used to signify line terminator
	}
	return c, nil
}

func lexUnquotedArgument(s *Stream) (arg []rune, err error) {
	for s.reading() {
		c, err := s.current()
		if err != nil {
			return nil, err
		} else if !argumentOk(c) {
			return arg, nil
		} else if c, err = checkEscape(s, c, 0); err != nil {
			return nil, err
		}

		arg = append(arg, c)
		s.pos++
	}

	return
}

func lexQuotedArgument(s *Stream) (arg []rune, err error) {
	for s.reading() {
		if c, err := s.current(); errors.Is(err, errForbidden) {
			return nil, errForbidden
		} else if !quotedArgumentOk(c) {
			if c != '"' {
				return nil, errUnclosedQuoted
			}

			s.pos++
			return arg, nil
		} else if c, err = checkEscape(s, c, 1); err != nil {
			return nil, err
		} else if c > 0 { // escaped line terminators allowed in quoted arguments
			arg = append(arg, c)
		}

		s.pos++
	}

	return nil, errUnclosedQuoted
}

func lexTripleQuotedArgument(s *Stream) (arg []rune, err error) {
	for endsMatched := 0; s.reading(); {
		c, err := s.current()
		if errors.Is(err, errForbidden) {
			return nil, errForbidden
		} else if !tripleQuotedArgumentOk(c) {
			if c != '"' {
				return nil, errUnclosedQuoted
			}

			s.pos++

			endsMatched++
			if endsMatched != 3 {
				continue
			}
			return arg, nil
		} else if endsMatched > 0 {
			arg = append(arg, slices.Repeat([]rune{'"'}, endsMatched)...)
			endsMatched = 0
			continue
		} else if c, err = checkEscape(s, c, 3); err != nil {
			return nil, err
		}

		arg = append(arg, c)
		s.pos++
	}

	return nil, errUnclosedQuoted
}

func lexArgument(s *Stream, quotes int) (arg []rune, err error) {
	switch quotes {
	case 0:
		return lexUnquotedArgument(s)
	case 1:
		return lexQuotedArgument(s)
	case 3:
		return lexTripleQuotedArgument(s)
	}
	return
}

func lex(src string) (p []Token, err error) {
	src = strings.TrimPrefix(src, "\uFEFF") // remove BOMs
	src = strings.TrimPrefix(src, "\uFFFE")
	src = strings.TrimSuffix(src, "\u001a") // remove end ^Z

	if !utf8.ValidString(src) {
		return nil, errors.New("malformed UTF-8")
	}

	// check for forbidden characters must be done based on token/location

	for s := (Stream{src: []rune(src)}); s.reading(); {
		c, err := s.current()
		if err != nil {
			break
		}

		// fmt.Printf("lex: %q\n", c)

		s.pos++
		switch argQuotes := 0; {
		case isLineTerminator(c):
			p = append(p, Token{Type: TokNewline})

		case isWhitespace(c):
			p = append(p, Token{Type: TokWhitespace})

		case c == '#': // comment until end of line
			op := s.pos
			for {
				s.pos++
				c, err = s.current()
				// fmt.Printf("comment: %c\n", c)
				if errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					break
				}
			}
			p = append(p, Token{Type: TokComment, Content: string(s.src[op:s.pos])})

		case c == ';':
			p = append(p, Token{Type: TokSemicolon})

		case c == '{':
			p = append(p, Token{Type: TokOpenBrace})

		case c == '}':
			p = append(p, Token{Type: TokCloseBrace})

		case c == '\\' && isLineTerminator(s.next(0)):
			s.pos++
			p = append(p, Token{Type: TokLineContinuation})

		case c == '"' && s.next(0) == '"' && s.next(1) == '"': // triple quoted argument
			argQuotes += 2
			fallthrough

		case c == '"': // quoted argument
			argQuotes++
			fallthrough

		default: // unquoted argument
			s.pos += argQuotes - 1

			arg, err := lexArgument(&s, argQuotes)
			if err != nil {
				return nil, err
			}

			p = append(p, Token{Type: TokArgument, Content: string(arg)})
		}
	}

	return
}
