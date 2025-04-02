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

// all unicode chars with whitespace property
func isWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}

var lineTerminators = [...]rune{'\r', '\n', '\u000a', '\u000b', '\u000c', '\u000d', '\u0085', '\u2028', '\u2029'}

func isLineTerminator(r rune) bool {
	return slices.Contains(lineTerminators[:], r)
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
	TokArgument   TokenType = "Argument"
	TokWhitespace TokenType = "Whitespace"
	TokNewline    TokenType = "Newline"
	TokComment    TokenType = "Comment"
	TokSemicolon  TokenType = "Semicolon"
	TokOpenBrace  TokenType = "OpenBrace"
	TokCloseBrace TokenType = "CloseBrace"
)

type Token struct {
	Type    TokenType
	Content string
}

func lexArgument(s *Stream, ok func(rune) bool, ender string) (arg Argument, err error) {
	var endsMatched int
	for {
		c, err := s.current()
		if err != nil {
			break
		}

		if ok(c) {
			arg = append(arg, c)
			s.advance()
			continue
		}

		if ender == "" {
			break
		}

		if endsMatched < len(ender) && c == rune(ender[endsMatched]) {
			endsMatched++
			s.advance()

			if endsMatched == len(ender) {
				fmt.Printf("matched ender %s at %d\n", ender, s.pos)
				break
			}

		} else {
			if endsMatched > 0 {
				return nil, fmt.Errorf("unexpected character %q at %d", c, s.pos)
			}
			break
		}
	}

	return
}

func lex(src string) (p []Token, err error) {
	s := Stream{src: []rune(src)}

	for {
		c, err := s.current()
		if err != nil {
			break
		}

		fmt.Printf("current: %q\n", c)

		switch {
		case isWhitespace(c):
			s.advance()
			p = append(p, Token{Type: TokWhitespace, Content: string(c)})

		case isLineTerminator(c):
			s.advance()
			p = append(p, Token{Type: TokNewline, Content: string(c)})

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

				arg, err := lexArgument(&s, func(r rune) bool {
					return !isReserved(r) && !isForbidden(r)
				}, `"""`)
				if err != nil {
					return nil, err
				} else if len(arg) == 0 {
					return nil, fmt.Errorf("empty triple quoted argument at %d", s.pos)
				}

				p = append(p, Token{Type: TokArgument, Content: string(arg)})
				break
			}

			arg, err := lexArgument(&s, func(r rune) bool {
				return !isLineTerminator(r) && !isReserved(r) && !isForbidden(r)
			}, `"`)
			if err != nil {
				return nil, err
			} else if len(arg) == 0 {
				return nil, fmt.Errorf("empty quoted argument at %d", s.pos)
			}

			p = append(p, Token{Type: TokArgument, Content: string(arg)})

		default: // unquoted argument
			arg, err := lexArgument(&s, func(r rune) bool {
				return !isWhitespace(r) && !isLineTerminator(r) && !isReserved(r) && !isForbidden(r)
			}, "")
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

func main() {
	const text = `
this is a test {
	# this is a comment
	this is another directive
}

yep

yep "quoted argument"
yep

yep """triple quoted 

argument"""
`

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
		fmt.Println(t.Type, t.Content)
	}
}
