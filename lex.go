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

func (p *Stream) len() int {
	return len(p.src)
}

func (p *Stream) reading() bool {
	return p.pos < p.len()
}

var (
	errEOF       = errors.New("EOF")
	errForbidden = errors.New("illegal character")
)

func (p *Stream) current() (c rune, err error) {
	if p.pos >= p.len() {
		// panic("EOF")
		return 0, errEOF
	}

	c = p.src[p.pos]
	if isForbidden(c) {
		// get illegal character as U+XXXX
		if c < 0x10000 {
			return 0, fmt.Errorf("%w U+%04X", errForbidden, c)
		}
		return 0, fmt.Errorf("%w U+%X", errForbidden, c)
	}

	return
}

func (p *Stream) advance() {
	p.pos++
}

// tokens
type TokenType string

const (
	TokArgument       TokenType = "Argument"
	TokNewline        TokenType = "Newline"
	TokWhitespace     TokenType = "Whitespace"
	TokComment        TokenType = "Comment"
	TokSemicolon      TokenType = "Semicolon"
	TokOpenBrace      TokenType = "OpenBrace"
	TokCloseBrace     TokenType = "CloseBrace"
	TokReverseSolidus TokenType = "ReverseSolidus"
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

	s.advance()
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
		return
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
		s.advance()
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

			s.advance()
			return arg, nil
		} else if c, err = checkEscape(s, c, 1); err != nil {
			return nil, err
		} else if c > 0 {
			arg = append(arg, c)
		}

		s.advance()
	}

	return nil, errUnclosedQuoted
}

func lexTripleQuotedArgument(s *Stream) (arg []rune, err error) {
	var endsMatched int
	for s.reading() {
		if c, err := s.current(); errors.Is(err, errForbidden) {
			return nil, errForbidden
		} else if tripleQuotedArgumentOk(c) {
			if endsMatched > 0 {
				arg = append(arg, slices.Repeat([]rune{'"'}, endsMatched)...)
				endsMatched = 0
				continue
			} else if c, err = checkEscape(s, c, 3); err != nil {
				return nil, err
			}

			arg = append(arg, c)
			s.advance()
			continue
		} else if c != '"' {
			return nil, fmt.Errorf("expected '\"' at %d", s.pos)
		}

		s.advance()

		if endsMatched == 2 {
			return arg, nil
		}
		endsMatched++
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
	src = strings.TrimSuffix(src, "\u001a") // remove ^Z

	if !utf8.Valid([]byte(src)) {
		return nil, errors.New("malformed UTF-8")
	}

	// check for forbidden characters must be done based on token/location

	s := Stream{src: []rune(src)}

	for s.reading() {
		c, err := s.current()
		if err != nil {
			break
		}

		// fmt.Printf("lex: %q\n", c)
		argQuotes := 0

		switch {
		case isLineTerminator(c):
			s.advance()
			p = append(p, Token{Type: TokNewline})

		case isWhitespace(c):
			s.advance()
			p = append(p, Token{Type: TokWhitespace})

		case c == '#': // comment until end of line
			s.advance()
			var comment []rune
			for {
				s.advance()
				c, err = s.current()
				// fmt.Printf("comment: %c\n", c)
				if errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					break
				}
				comment = append(comment, c)
			}
			p = append(p, Token{Type: TokComment, Content: string(comment)})

		case c == ';':
			s.advance()
			p = append(p, Token{Type: TokSemicolon})

		case c == '{':
			s.advance()
			p = append(p, Token{Type: TokOpenBrace})

		case c == '}':
			s.advance()
			p = append(p, Token{Type: TokCloseBrace})

		case c == '\\' && s.pos+1 < s.len() && isLineTerminator(s.src[s.pos+1]):
			s.advance()
			p = append(p, Token{Type: TokReverseSolidus})

		case c == '"' && s.pos+3 < s.len() && s.src[s.pos+1] == '"' && s.src[s.pos+2] == '"':
			argQuotes += 2
			fallthrough

		case c == '"': // quoted argument
			argQuotes++
			fallthrough

		default: // unquoted argument
			for range argQuotes {
				s.advance()
			}

			arg, err := lexArgument(&s, argQuotes)
			if err != nil {
				return nil, err
			}

			p = append(p, Token{Type: TokArgument, Content: string(arg)})
		}
	}

	return
}
