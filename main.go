// implementation of the Confetti language

package main

import (
	"fmt"
	"strings"
)

const text = `
# This is a comment.

probe-device eth0 \
eth1

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
	t, err := lex(text)
	if err != nil {
		panic(err)
	}

	// for _, t := range t {
	// 	if t.Type == TokWhitespace {
	// 		continue
	// 	}
	// 	fmt.Printf("%14s  %s\n", t.Type, t.Content)
	// }

	p, err := parse(t)
	if err != nil {
		panic(err)
	}

	for _, d := range p {
		printDirective(d, 0)
	}
}
