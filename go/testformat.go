package confetti

import "strings"

func testFormat(p []Directive, depth int) string {
	var b strings.Builder

	indent := strings.Repeat("    ", depth)

	for _, d := range p {
		b.WriteString(indent)
		for i, a := range d.Arguments {
			b.WriteString("<" + a + ">")
			if i < len(d.Arguments)-1 {
				b.WriteByte(' ')
			}
		}

		if d.Subdirectives == nil {
			b.WriteByte('\n')
			continue
		}

		b.WriteString(" [\n" + testFormat(d.Subdirectives, depth+1) + indent + "]\n")
	}

	return b.String()
}
