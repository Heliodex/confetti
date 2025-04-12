package main

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

func parse(ts []Token, exts Extensions) (p []Directive, err error) {
	var current Directive
	push := func() {
		if current.Arguments == nil {
			return
		}
		p = append(p, current)
		current = Directive{}
	}

	prevSignificant := func(i int) (prev TokenType) {
		for i--; i > 0; i-- {
			if prev = ts[i].Type; prev != TokWhitespace && prev != TokComment {
				return
			}
		}
		return
	}

	for i := 0; i < len(ts); i++ {
		switch t := ts[i]; t.Type {
		case Tok0qArgument, Tok1qArgument, Tok3qArgument:
			current.Arguments = append(current.Arguments, t.Content)

		case TokSemicolon: // end of directive
			if prev := prevSignificant(i); prev == TokSemicolon || prev == TokNewline || prev == TokLineContinuation {
				return nil, errors.New("unexpected ';'")
			}
			fallthrough

		case TokNewline: // end of directive
			push()

		case TokOpenBrace:
			if i == len(ts)-1 || prevSignificant(i) == TokSemicolon {
				// fmt.Println(prevNonWhitespace(i).Type == TokSemicolon)
				return nil, fmt.Errorf("unexpected '{'")
			}

			// Get all tokens until next close brace
			i++
			si := i
			for depth := 0; i < len(ts); i++ {
				// escapes should be dealt with in lexer
				if t = ts[i]; t.Type == TokOpenBrace {
					depth++
				} else if t.Type == TokCloseBrace {
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

		case TokCloseBrace:
			return nil, errors.New("found '}' without matching '{'")

		case TokLineContinuation:
			if current.Arguments == nil {
				return nil, fmt.Errorf("unexpected line continuation")
			}
		}
	}

	push()
	return
}
