package main

import "strings"

func testFormat(p []Directive, depth int) (f string, err error) {
	var b strings.Builder

	indent := strings.Repeat("    ", depth)

	for _, d := range p {
		b.WriteString(indent)
		for i, a := range d.Arguments {
			b.WriteByte('<')
			b.WriteString(string(a))
			b.WriteByte('>')
			if i < len(d.Arguments)-1 {
				b.WriteByte(' ')
			}
		}

		if len(d.Subdirectives) > 0 {
			b.WriteString(" [\n")
			subdirs, err := testFormat(d.Subdirectives, depth+1)
			if err != nil {
				return "", err
			}

			b.WriteString(subdirs)
			b.WriteString(indent)
			b.WriteString("]")

		}

		b.WriteByte('\n')
	}

	return b.String(), nil
}
