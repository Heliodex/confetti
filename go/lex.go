package confetti

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var lineTerminators = []rune{'\r', '\n', '\u000a', '\u000b', '\u000c', '\u000d', '\u0085', '\u2028', '\u2029'}

func isLineTerminator(r rune) bool {
	return slices.Contains(lineTerminators, r)
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

var reserved = []rune{'"', '#', ';', '{', '}'}

func isReserved(r rune, exts Extensions) bool {
	return slices.Contains(reserved, r) || exts.Has(ExtExpressionArguments) && r == '('
}

type stream struct {
	src []rune
	pos int
}

func (s *stream) reading() bool {
	return s.pos < len(s.src)
}

var (
	errEOF       = errors.New("EOF")
	errForbidden = errors.New("illegal character")
)

func (s *stream) current() (c rune, err error) {
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

func (s *stream) next(n int) rune {
	if s.pos+n < len(s.src) {
		return s.src[s.pos+n]
	}
	return 0
}

// tokens
type tokenType uint8

const (
	tokUnicode tokenType = iota
	tok0qArgument
	tok1qArgument
	tok3qArgument
	tokNewline
	tokLineContinuation
	tokWhitespace
	tokComment
	tokSemicolon
	tokOpenBrace
	tokCloseBrace
)

type token struct {
	Type        tokenType
	Content, Og string
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

func checkEscape(s *stream, c rune, quoted uint8) (r rune, escaped bool, err error) {
	if c != '\\' {
		return c, false, nil
	}

	s.pos++
	if c, err = s.current(); err != nil {
		if errors.Is(err, errForbidden) {
			return 0, false, errIllegalEscape
		} else if quoted > 0 {
			return 0, false, errIncompleteEscape
		}
		return 0, false, errIllegalEscape
	} else if isWhitespace(c) || isLineTerminator(c) {
		if quoted == 3 {
			if isLineTerminator(c) {
				return 0, false, errIncompleteEscape
			}
			return 0, false, errIllegalEscape
		} else if quoted == 0 || quoted == 1 && !isLineTerminator(c) {
			return 0, false, errIllegalEscape
		}
		return 0, true, nil // r = 0 used to signify line terminator
	}
	return c, true, nil
}

func getPunctuator(s *stream, ps string) (l int) {
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

func lex0qArgument(s *stream, exts Extensions) (arg, ogarg []rune, err error) {
	for s.reading() {
		c, err := s.current()
		if err != nil {
			return nil, nil, err
		} else if !argumentOk(c, exts) || exts.Has(ExtPunctuatorArguments) && getPunctuator(s, exts[ExtPunctuatorArguments]) != 0 {
			return arg, ogarg, nil
		}

		ec, escd, err := checkEscape(s, c, 0)
		if err != nil {
			return nil, nil, err
		} else if escd {
			ogarg = append(ogarg, '\\')
		}

		arg = append(arg, ec)
		ogarg = append(ogarg, ec)
		s.pos++
	}

	return
}

func lex1qArgument(s *stream) (arg, ogarg []rune, err error) {
	for s.reading() {
		c, err := s.current()
		if errors.Is(err, errForbidden) {
			return nil, nil, errForbidden
		} else if !quotedArgumentOk(c) {
			if c != '"' {
				return nil, nil, errUnclosedQuoted
			}

			s.pos++
			return arg, ogarg, nil
		}

		ec, escd, err := checkEscape(s, c, 1)
		if err != nil {
			return nil, nil, err
		} else if escd {
			ogarg = append(ogarg, '\\')
		}

		if ec == 0 { // escaped line terminators allowed in quoted arguments
			nc, _ := s.current()
			ogarg = append(ogarg, nc)
		} else {
			arg = append(arg, ec)
			ogarg = append(ogarg, ec)
		}
		s.pos++
	}

	return nil, nil, errUnclosedQuoted
}

func lex3qArgument(s *stream) (arg, ogarg []rune, err error) {
	for endsMatched := 0; s.reading(); {
		c, err := s.current()
		if errors.Is(err, errForbidden) {
			return nil, nil, errForbidden
		} else if !tripleQuotedArgumentOk(c) {
			if c != '"' {
				return nil, nil, errUnclosedQuoted
			}

			ogarg = append(ogarg, c)
			s.pos++

			endsMatched++
			if endsMatched != 3 {
				continue
			}
			return arg, ogarg[:len(ogarg)-3], nil
		} else if endsMatched > 0 {
			arg = append(arg, slices.Repeat([]rune{'"'}, endsMatched)...)
			endsMatched = 0
			continue
		}

		ec, escd, err := checkEscape(s, c, 3)
		if err != nil {
			return nil, nil, err
		} else if escd {
			ogarg = append(ogarg, '\\')
		}

		arg = append(arg, ec)
		ogarg = append(ogarg, ec)
		s.pos++
	}

	return nil, nil, errUnclosedQuoted
}

func lex(src string, exts Extensions) (ts []token, err error) {
	// remove BOMs
	if strings.HasPrefix(src, "\uFEFF") {
		ts = append(ts, token{Type: tokUnicode, Content: "\uFEFF"})
		src = src[3:]
	} else if strings.HasPrefix(src, "\uFFFE") {
		ts = append(ts, token{Type: tokUnicode, Content: "\uFFFE"})
		src = src[3:]
	}

	// remove ^Z
	if strings.HasSuffix(src, "\u001a") {
		defer func() {
			ts = append(ts, token{Type: tokUnicode, Content: "\u001a"})
		}()
		src = src[:len(src)-1]
	}

	if !utf8.ValidString(src) {
		return nil, errors.New("malformed UTF-8")
	}

	// check for forbidden characters must be done based on token/location

	for s := (stream{src: []rune(src)}); s.reading(); {
		c, err := s.current()
		if err != nil {
			break
		}

		switch op := s.pos; {
		case isLineTerminator(c):
			s.pos++
			ts = append(ts, token{Type: tokNewline, Content: string(c)})

		case isWhitespace(c):
			s.pos++
			ts = append(ts, token{Type: tokWhitespace, Content: string(c)})

		case exts.Has(ExtCStyleComments) && c == '/' && s.next(1) == '/': // C-style comment
			for s.pos += 2; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					break
				}
			}
			content := string(s.src[op+2 : s.pos])
			ts = append(ts, token{Type: tokComment, Content: content, Og: "//" + content})

		case c == '#': // comment until end of line
			for s.pos++; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil || isLineTerminator(c) {
					break
				}
			}
			content := string(s.src[op+1 : s.pos])
			ts = append(ts, token{Type: tokComment, Content: content, Og: "#" + content})

		case exts.Has(ExtCStyleComments) && c == '/' && s.next(1) == '*': // block comment
			for s.pos += 2; ; s.pos++ {
				if c, err = s.current(); errors.Is(err, errForbidden) {
					return nil, errForbidden
				} else if err != nil {
					return nil, errors.New("unterminated multi-line comment")
				} else if c == '*' && s.next(1) == '/' {
					break
				}
			}
			content := string(s.src[op+2 : s.pos])
			ts = append(ts, token{Type: tokComment, Content: content, Og: "/*" + content + "*/"})
			s.pos += 2 // */

		case c == ';':
			s.pos++
			ts = append(ts, token{Type: tokSemicolon})

		case c == '{':
			s.pos++
			ts = append(ts, token{Type: tokOpenBrace})

		case c == '}':
			s.pos++
			ts = append(ts, token{Type: tokCloseBrace})

		case c == '\\' && isLineTerminator(s.next(1)):
			s.pos += 2
			ts = append(ts, token{Type: tokLineContinuation})

		case exts.Has(ExtExpressionArguments) && c == '(':
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
			content := string(s.src[op+1 : s.pos])
			ts = append(ts, token{Type: tok0qArgument, Content: content, Og: "(" + content + ")"})
			s.pos++

		case exts.Has(ExtPunctuatorArguments) && getPunctuator(&s, exts[ExtPunctuatorArguments]) != 0:
			// read punctuator as argument
			s.pos += getPunctuator(&s, exts[ExtPunctuatorArguments])
			content := string(s.src[op:s.pos])
			ts = append(ts, token{Type: tok0qArgument, Content: content, Og: content})

		case c == '"' && s.next(1) == '"' && s.next(2) == '"': // triple quoted argument
			s.pos += 3
			arg, ogarg, err := lex3qArgument(&s)
			if err != nil {
				return nil, err
			}
			ts = append(ts, token{Type: tok3qArgument, Content: string(arg), Og: string(ogarg)})

		case c == '"': // quoted argument
			s.pos++
			arg, ogarg, err := lex1qArgument(&s)
			if err != nil {
				return nil, err
			}
			ts = append(ts, token{Type: tok1qArgument, Content: string(arg), Og: string(ogarg)})

		default: // unquoted argument
			arg, ogarg, err := lex0qArgument(&s, exts)
			if err != nil {
				return nil, err
			}
			ts = append(ts, token{Type: tok0qArgument, Content: string(arg), Og: string(ogarg)})
		}
	}

	return
}
