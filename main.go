// implementation of the Confetti language

package main

import (
	"errors"
	"fmt"
	"slices"
	"unicode"
)

// surrogate, unassigned
func isForbidden(r rune) bool {
	return unicode.Is(unicode.Cs, r) || unicode.Is(unicode.Co, r)
}

var lineTerminators = [...]rune{'\r', '\n', '\u000a', '\u000b', '\u000c', '\u000d', '\u0085', '\u2028', '\u2029'}

func isLineTerminator(r rune) bool {
	return slices.Contains(lineTerminators[:], r)
}

// all unicode chars with whitespace property
func isWhitespace(r rune) bool {
	return !isLineTerminator(r) && unicode.IsSpace(r)
}

var reserved = [...]rune{'"', '#', ';', '{', '}'}

func isReserved(r rune) bool {
	return slices.Contains(reserved[:], r)
}

// A directive “argument” shall be a sequence of one or more characters from the argument character set. The argument character set shall consist of any Unicode scalar value excluding characters from the white space, line terminator, reserved punctuator, and forbidden character sets.

func isArgument(r rune) bool {
	return !isWhitespace(r) && !isLineTerminator(r) && !isReserved(r) && !isForbidden(r)
}

// The Confetti language consists of zero or more directives. A directive consists of one or more arguments and optional subdirectives.

type Argument []rune

type Directive struct {
	Arguments     []Argument
	Subdirectives []Directive
}

type Stream struct {
	src []rune
	pos int
}

func (p *Stream) len() int {
	return len(p.src)
}

var errEOF = errors.New("EOF")

func (p *Stream) current() (c rune, err error) {
	if p.pos >= p.len() {
		return 0, errEOF
	}

	c = p.src[p.pos]
	if isForbidden(c) {
		return 0, fmt.Errorf("forbidden character %q at %d", c, p.pos)
	}

	return
}

func (p *Stream) advance() (n rune, err error) {
	n, err = p.current()
	p.pos++

	return
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

func argumentOk(r rune) bool {
	return !isWhitespace(r) && !isLineTerminator(r) && !isReserved(r) && !isForbidden(r)
}

func quotedArgumentOk(r rune) bool {
	return !isLineTerminator(r) && r != '"' && !isForbidden(r)
}

func tripleQuotedArgumentOk(r rune) bool {
	return !isForbidden(r) && r != '"'
}

func lexArgument(s *Stream) (arg []rune, err error) {
	for {
		c, err := s.current()
		if err != nil {
			return nil, err
		} else if argumentOk(c) {
			arg = append(arg, c)
			s.advance()
			continue
		}
		return arg, nil
	}
}

func lexQuotedArgument(s *Stream) (arg []rune, err error) {
	for {
		if c, err := s.current(); err != nil {
			return nil, err
		} else if quotedArgumentOk(c) {
			arg = append(arg, c)
			s.advance()
			continue
		} else if c != '"' {
			continue
		}

		s.advance()
		return arg, nil
	}
}

func lexTripleQuotedArgument(s *Stream) (arg []rune, err error) {
	var endsMatched int
	for {
		c, err := s.current()
		if err != nil {
			return nil, err
		} else if tripleQuotedArgumentOk(c) {
			if endsMatched > 0 {
				arg = append(arg, slices.Repeat([]rune{'"'}, endsMatched)...)
				endsMatched = 0
				continue
			}

			arg = append(arg, c)
			s.advance()
			continue
		} else if c != '"' {
			continue
		}

		endsMatched++
		s.advance()

		if endsMatched == 3 {
			return arg, nil
		}
	}
}

func lex(src string) (p []Token, err error) {
	s := Stream{src: []rune(src)}

	for {
		c, err := s.current()
		if err != nil {
			break
		}

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
				if err != nil || isLineTerminator(c) {
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

		case c == '"': // quoted argument
			s.advance()

			// if next 2 are also quotes, triple quoted
			if s.pos+2 < s.len() && s.src[s.pos] == '"' && s.src[s.pos+1] == '"' {
				s.advance()
				s.advance()

				arg, err := lexTripleQuotedArgument(&s)
				if err != nil {
					return nil, err
				} else if len(arg) == 0 {
					return nil, fmt.Errorf("empty triple quoted argument at %d", s.pos)
				}

				p = append(p, Token{Type: TokArgument, Content: string(arg)})
				break
			}

			arg, err := lexQuotedArgument(&s)
			if err != nil {
				return nil, err
			} else if len(arg) == 0 {
				return nil, fmt.Errorf("empty quoted argument at %d", s.pos)
			}

			p = append(p, Token{Type: TokArgument, Content: string(arg)})

		default: // unquoted argument
			arg, err := lexArgument(&s)
			if err != nil {
				return nil, err
			} else if len(arg) == 0 {
				return nil, fmt.Errorf("empty argument at %d", s.pos)
			}

			p = append(p, Token{Type: TokArgument, Content: string(arg)})
		}
	}

	return
}

const text = `
# This is a comment.

probe-device eth0 eth1

user * {
login anonymous
password "${ENV:ANONPASS}"
machine 167.89.14.1
proxy {
	try-ports 582 583 584
}
}

user "Joe Williams" {
login joe
machine 167.89.14.1
}

paragraph """
Lorem
ipsum
"dolor"
sit
amet."""
`

func main() {
	p, err := lex(text)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done!")
	// fmt.Println(p)

	for _, t := range p {
		if t.Type == TokWhitespace {
			continue
		}
		fmt.Println(t.Type, ":", t.Content)
	}
}
