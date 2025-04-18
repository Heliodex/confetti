package confetti

import (
	"errors"
	"fmt"
)

// The Confetti language consists of zero or more directives. A directive consists of one or more arguments and optional subdirectives.

// The entire AST of the language is ONE struct!!!!
type Directive struct {
	Arguments     []string
	Subdirectives []Directive
}

func (d Directive) Equals(other Directive) (eq bool) {
	if len(d.Arguments) != len(other.Arguments) {
		return
	}
	for i, arg := range d.Arguments {
		if arg != other.Arguments[i] {
			return
		}
	}

	if len(d.Subdirectives) != len(other.Subdirectives) {
		return
	}
	for i, sub := range d.Subdirectives {
		if !sub.Equals(other.Subdirectives[i]) {
			return
		}
	}

	return true
}

func parse(ts []token, exts Extensions) (p []Directive, err error) {
	var current Directive
	push := func() {
		if current.Arguments == nil {
			return
		}
		p = append(p, current)
		current = Directive{}
	}

	i := 0

	for prevSignificant := func() tokenType {
		for ci := i - 1; ci > 0; ci-- {
			if prev := ts[ci].Type; prev != tokWhitespace && prev != tokComment {
				return prev
			}
		}
		return tokUnicode
	}; i < len(ts); i++ {
		switch t := ts[i]; t.Type {
		case tok0qArgument, tok1qArgument, tok3qArgument:
			current.Arguments = append(current.Arguments, t.Content)

		case tokSemicolon: // end of directive
			if prev := prevSignificant(); prev == tokSemicolon || prev == tokNewline || prev == tokLineContinuation {
				return nil, errors.New("unexpected ';'")
			}
			push()

		case tokNewline: // end of directive
			push()

		case tokOpenBrace:
			if i == len(ts)-1 || prevSignificant() == tokSemicolon {
				return nil, fmt.Errorf("unexpected '{'")
			}

			// Get all tokens until next close brace
			i++
			si := i
			for depth := 0; i < len(ts); i++ {
				// escapes should be dealt with in lexer
				if t2 := ts[i]; t2.Type == tokOpenBrace {
					depth++
				} else if t2.Type == tokCloseBrace {
					if depth == 0 {
						break
					}
					depth--
				} else if i == len(ts)-1 {
					return nil, fmt.Errorf("expected '}'")
				}
			}

			subp, err := parse(ts[si:i], exts)
			if err != nil {
				return nil, err
			} else if current.Arguments == nil {
				// push to the previous directive
				p[len(p)-1].Subdirectives = subp
				break
			}

			current.Subdirectives = subp
			push()

		case tokCloseBrace:
			return nil, errors.New("found '}' without matching '{'")

		case tokLineContinuation:
			if current.Arguments == nil {
				return nil, fmt.Errorf("unexpected line continuation")
			}
		}
	}

	push()
	return
}
