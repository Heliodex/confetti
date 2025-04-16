package confetti_test

import (
	"fmt"
	"strings"
	"testing"

	confetti "github.com/Heliodex/confetti-go"
)

func printDirective(d confetti.Directive, depth int) {
	prefix := strings.Repeat("  ", depth)

	fmt.Println(prefix + "Directive:")
	for _, arg := range d.Arguments {
		fmt.Printf(prefix+"  %q\n", arg)
	}
	for _, sub := range d.Subdirectives {
		printDirective(sub, depth+1)
	}
}

type LibraryTest struct {
	Input      string
	Extensions confetti.Extensions
	Output     []confetti.Directive
}

var tests = []LibraryTest{
	{
		Input: `// This is a simple, C-like program.
msg:="Hello, World!"
if(isEmpty(msg)){
msg="(nil message)"
}
echo msg
`,
		Extensions: confetti.Extensions{
			confetti.ExtCStyleComments:      "",
			confetti.ExtExpressionArguments: "",
			confetti.ExtPunctuatorArguments: "=\n:=",
		},
		Output: []confetti.Directive{
			{
				Arguments: []string{"msg", ":=", "Hello, World!"},
			},
			{
				Arguments: []string{"if", "isEmpty(msg)"},
				Subdirectives: []confetti.Directive{
					{
						Arguments: []string{"msg", "=", "(nil message)"},
					},
				},
			},
			{
				Arguments: []string{"echo", "msg"},
			},
		},
	},
}

func TestLibrary(t *testing.T) {
	for _, test := range tests {
		dirs, err := confetti.Load(test.Input, test.Extensions)
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}

		for i, d := range dirs {
			fmt.Printf("Directive %d:\n", i)
			printDirective(d, 0)

			if !d.Equals(test.Output[i]) {
				t.Fatalf("Directive mismatch at index %d\nExpected:\n%v\nGot:\n%v", i, test.Output[i], d)
			}
		}
	}
}
