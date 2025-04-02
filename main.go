// implementation of the Confetti language

package main

import (
	"fmt"
	"strings"
)

const text = `\{\"\'\}\; bar`

func printDirective(d Directive, depth int) {
	prefix := strings.Repeat("  ", depth)

	fmt.Println(prefix + "Directive:")
	for _, arg := range d.Arguments {
		fmt.Printf(prefix+"  %q\n", string(arg))
	}
	for _, sub := range d.Subdirectives {
		printDirective(sub, depth+1)
	}
}

func main() {
	fmt.Println("Lexing")

	ts, err := lex(text)
	if err != nil {
		panic(err)
	}

	fmt.Println("Parsing")

	// for _, t := range ts {
	// 	if t.Type == TokWhitespace {
	// 		continue
	// 	}
	// 	fmt.Printf("%14s  %s\n", t.Type, t.Content)
	// }

	p, err := parse(ts)
	if err != nil {
		panic(err)
	}

	for _, d := range p {
		printDirective(d, 0)
	}
}
