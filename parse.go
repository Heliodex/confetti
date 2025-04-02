package main

import "fmt"

// The Confetti language consists of zero or more directives. A directive consists of one or more arguments and optional subdirectives.

type Argument []rune

type Directive struct {
	Arguments     []Argument
	Subdirectives []Directive
}

func parse(lexed []Token) (p []Directive, err error) {
	var current Directive
	push := func() {
		if len(current.Arguments) == 0 {
			return
		}
		p = append(p, current)
		current = Directive{}
	}

	for i := 0; i < len(lexed); i++ {
		t := lexed[i]

		switch t.Type {
		case TokArgument:
			current.Arguments = append(current.Arguments, Argument(t.Content))
		case TokNewline, TokSemicolon: // end of directive
			push()
		case TokWhitespace, TokComment: // Ignore whitespace and comments
		case TokOpenBrace:
			// Get all tokens until next close brace
			var ts []Token

			// also account for nested
			depth := 1

			for i++; i < len(lexed); i++ {
				t = lexed[i]

				if t.Type == TokOpenBrace {
					depth++
				} else if t.Type == TokCloseBrace {
					depth--
				}

				if depth == 0 {
					break
				}
				ts = append(ts, t)
			}

			if t.Type != TokCloseBrace {
				return nil, fmt.Errorf("missing closing brace for subdirective")
			}

			subdirs, err := parse(ts)
			if err != nil {
				return nil, err
			}

			current.Subdirectives = subdirs
			push()
		case TokReverseSolidus:
			// escape character
			if i+1 < len(lexed) {
				i++
				t = lexed[i]
			}
		default:
			return nil, fmt.Errorf("unexpected token %s", t.Type)
		}
	}

	push()
	return
}
