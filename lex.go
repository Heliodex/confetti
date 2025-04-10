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

func isReserved(r rune, exts Extensions) bool {
	return slices.Contains(reserved[:], r) || exts.Has("expression_arguments") && r == '('
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
	} else if c = s.src[s.pos]; isForbidden(c) {
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
func argumentOk(r rune, exts Extensions) bool {
	return !isWhitespace(r) && !isLineTerminator(r) && !isReserved(r, exts)
}

func quotedArgumentOk(r rune) bool {
	return !isLineTerminator(r) && r != '"'
}

func tripleQuotedArgumentOk(r rune) bool {
	return r != '"'
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

func getPunctuator(s *Stream, ps string) (l int) {
	ps = strings.ReplaceAll(ps, "\r\n", "\n")
	ps = strings.ReplaceAll(ps, "\r", "\n")
	ps = strings.TrimSpace(ps)

	puncts := strings.Split(ps, "\n")
	// sort puncts by length descending
	slices.SortFunc(puncts, func(a, b string) int {
		return len([]rune(b)) - len([]rune(a)) // footgun: len() counts bytes for strings
	})
	// fmt.Printf("punctuators: %q\n", puncts)

	for _, p := range puncts {
		rest := s.src[s.pos:]

		rp := []rune(p)
		l = len(rp)
		// fmt.Printf("%q %d\n", p, l)
		if l <= len(rest) && string(rest[:l]) == p {
			// fmt.Printf("punctuator: %q\n", p)
			return
		}
	}
	// fmt.Printf("not a punctuator: %q\n", s.src[s.pos:])
	return 0
}

func lexUnquotedArgument(s *Stream, exts Extensions) (arg []rune, err error) {
	for s.reading() {
		c, err := s.current()
		if err != nil {
			return nil, err
		} else if !argumentOk(c, exts) || exts.Has("punctuator_arguments") && getPunctuator(s, exts["punctuator_arguments"]) != 0 {
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

func lex(src string, exts Extensions) (p []Token, err error) {
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

		switch op := s.pos; {
		case isLineTerminator(c):
			s.pos++
			p = append(p, Token{Type: TokNewline})

		case isWhitespace(c):
			s.pos++
			p = append(p, Token{Type: TokWhitespace})

		case exts.Has("c_style_comments") && c == '/' && s.next(1) == '/': // C-style comment
			for s.pos += 2; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					break
				}
			}
			p = append(p, Token{Type: TokComment, Content: string(s.src[op+2 : s.pos])})

		case c == '#': // comment until end of line
			for s.pos++; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					break
				}
			}
			p = append(p, Token{Type: TokComment, Content: string(s.src[op+1 : s.pos])})

		case exts.Has("c_style_comments") && c == '/' && s.next(1) == '*': // block comment
			for s.pos += 2; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil {
					return nil, errors.New("unterminated multi-line comment")
				} else if c == '*' && s.next(1) == '/' {
					break
				}
			}
			p = append(p, Token{Type: TokComment, Content: string(s.src[op+2 : s.pos])})
			s.pos += 2 // */

		case c == ';':
			s.pos++
			p = append(p, Token{Type: TokSemicolon})

		case c == '{':
			s.pos++
			p = append(p, Token{Type: TokOpenBrace})

		case c == '}':
			s.pos++
			p = append(p, Token{Type: TokCloseBrace})

		case c == '\\' && isLineTerminator(s.next(1)):
			s.pos += 2
			p = append(p, Token{Type: TokLineContinuation})

		case exts.Has("expression_arguments") && c == '(':
			s.pos++
			// read until corresponding closing parenthesis
			for depth := 0; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					return nil, errors.New("incomplete expression")
				} else if c == '(' {
					depth++
				} else if c == ')' {
					if depth == 0 {
						break
					}
					depth--
				}
			}
			p = append(p, Token{Type: TokArgument, Content: string(s.src[op+1 : s.pos])})
			s.pos++

		case exts.Has("punctuator_arguments") && getPunctuator(&s, exts["punctuator_arguments"]) != 0:
			// read punctuator as argument
			s.pos += getPunctuator(&s, exts["punctuator_arguments"])
			p = append(p, Token{Type: TokArgument, Content: string(s.src[op:s.pos])})

		case c == '"' && s.next(1) == '"' && s.next(2) == '"': // triple quoted argument
			s.pos += 3
			arg, err := lexTripleQuotedArgument(&s)
			if err != nil {
				return nil, err
			}
			p = append(p, Token{Type: TokArgument, Content: string(arg)})

		case c == '"': // quoted argument
			s.pos++
			arg, err := lexQuotedArgument(&s)
			if err != nil {
				return nil, err
			}
			p = append(p, Token{Type: TokArgument, Content: string(arg)})

		default: // unquoted argument
			arg, err := lexUnquotedArgument(&s, exts)
			if err != nil {
				return nil, err
			}
			p = append(p, Token{Type: TokArgument, Content: string(arg)})
		}
	}

	return
}
