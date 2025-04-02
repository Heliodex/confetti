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
		case TokNewline:
			push()
		case TokWhitespace:
			// Ignore whitespace
		case TokComment:
			// Ignore comments
		default:
			return nil, fmt.Errorf("unexpected token %s", t.Type)
		}
	}

	return
}
