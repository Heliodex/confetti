// implementation of the Confetti language

package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
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

func (p *Stream) next(n int) (s string, err error) {
	if p.pos >= p.len() {
		return "", errEOF
	}

	s = string(p.src[p.pos:min(p.pos+n, p.len())])
	for _, r := range s {
		if isForbidden(r) {
			return "", fmt.Errorf("forbidden character %q at %d", r, p.pos)
		}
	}

	return
}

func (p *Stream) advance() (n rune, err error) {
	n, err = p.current()
	p.pos++

	return
}

func parse(src string) (p []Directive, err error) {
	s := Stream{src: []rune(src)}

	var dir Directive
	var arg Argument

	defer func() {
		if len(arg) > 0 {
			dir.Arguments = append(dir.Arguments, arg)
		}
		if len(dir.Arguments) > 0 {
			p = append(p, dir) // add last directive if any
		}
	}()

	for {
		curr, err := s.advance()
		if err != nil {
			if err == errEOF {
				// end of stream
				break
			}
			return nil, err
		}

		var canBeSpace, canBeNewline bool

		// if starts with 1 quote, can be space
		// if starts with 3 quotes, can be space or newline
		if curr == '"' {
			canBeSpace = true

			curr, err = s.advance()
			if err != nil {
				return nil, fmt.Errorf("error at start of quoted argument parsing: %w", err)
			}

			if curr == '"' {
				curr, err = s.advance()
				if err != nil {
					return nil, fmt.Errorf("error at end of quoted argument parsing: %w", err)
				}

				if curr == '"' {
					// triple quotes, can be space or newline
					canBeNewline = true
					fmt.Println("start tripel quote")
					curr, err = s.advance()
					if err != nil {
						return nil, fmt.Errorf("error at start of triple quoted argument parsing: %w", err)
					}
				} else {
					// double quotes, end of argument
					canBeSpace = false

					dir.Arguments = append(dir.Arguments, arg)
					arg = Argument{} // reset argument
				}
			}
		}

		if isArgument(curr) || (canBeSpace && isWhitespace(curr)) || (canBeNewline && isLineTerminator(curr)) {
			// start an argument
			fmt.Println(canBeNewline, canBeSpace, fmt.Sprintf("%q", string(curr)))
			for isArgument(curr) || (canBeSpace && isWhitespace(curr)) || (canBeNewline && isLineTerminator(curr)) {
				arg = append(arg, curr)
				curr, err = s.advance()
				if err != nil {
					if err == errEOF {
						return p, nil
					}
					return nil, fmt.Errorf("error during argument parsing: %w", err)
				}

				if curr == '"' {
					if canBeSpace && !canBeNewline {
						// end of argument
						canBeSpace = false
						break
					}

					n2, err := s.next(2)
					if err != nil {
						return nil, fmt.Errorf("expected end triple quotes quotes during argument parsing: %w", err)
					}

					if n2 == `"""` && canBeNewline {
						// end of argument
						canBeSpace = false
						canBeNewline = false
						s.pos += 2 // skip the quotes
						break
					}
				}
			}

			if len(arg) > 0 {
				// argument parsed
				dir.Arguments = append(dir.Arguments, arg)
				arg = Argument{} // reset argument
			}
		}

		if (isLineTerminator(curr) || curr == ';') && len(dir.Arguments) > 0 {
			fmt.Println("were done here")
			// end of directive
			p = append(p, dir)
			dir = Directive{} // reset directive
		} else if curr == '{' {
			// start of subdirective
			if len(dir.Arguments) == 0 {
				return nil, errors.New("subdirective without arguments")
			}

			//
		}
	}

	return
}

func printDirective(d Directive, depth int) {
	prefix := strings.Repeat("  ", depth)

	fmt.Println(prefix + "Directive:")
	for _, arg := range d.Arguments {
		fmt.Printf(prefix+"  Argument: %q\n", string(arg))
	}
	for _, sub := range d.Subdirectives {
		fmt.Println(prefix + "  Subdirective:")
		printDirective(sub, depth+1)
	}
}

func main() {
	const text = `
hows it """triple
quoted"" going"""
`

	p, err := parse(text)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done!")
	// fmt.Println(p)

	for _, d := range p {
		printDirective(d, 0)
	}
}
