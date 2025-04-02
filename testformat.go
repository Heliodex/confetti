package main

import "strings"

func testFormat(p []Directive) (f string, err error) {
	var b strings.Builder

	for _, d := range p {
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
			subdirs, err := testFormat(d.Subdirectives)
			if err != nil {
				return "", err
			}

			// indent subdirs
			lines := strings.Split(subdirs, "\n")
			for i, line := range lines {
				if i > 0 {
					b.WriteByte('\n')
				}
				if i < len(lines)-1 {
					b.WriteString("    ")
				}
				b.WriteString(line)
			}

			b.WriteString("]")

		}

		b.WriteByte('\n')
	}

	return b.String(), nil
}
