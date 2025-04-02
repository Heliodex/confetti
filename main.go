// implementation of the Confetti language

package main

import (
	"fmt"
	"strings"
)

const text = `
states {
    greet_player {
        look_at $player
        wait 1s # Pause one second before walking towards the player.
        walk_to $player
        say "Good evening traveler."
    }

    last_words {
        say "Tis a cruel world!"
    }
}

events {
    player_spotted {
        goto_state greet_player
    }

    died {
        goto_state last_words
    }
}
`

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

	fmt.Printf("Parsing")

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
