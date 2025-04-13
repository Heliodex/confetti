package confetti

import "strings"

func testReformat(ts []token) (string, error) {
	var b strings.Builder

	for _, t := range ts {
		switch t.Type {
		case tokUnicode:
			b.WriteString(t.Content)

		case tok0qArgument:
			b.WriteString(t.Og)

		case tok1qArgument:
			b.WriteString("\"" + t.Og + "\"")

		case tok3qArgument:
			b.WriteString("\"\"\"" + t.Og + "\"\"\"")

		case tokNewline:
			b.WriteString(t.Content)

		case tokLineContinuation:
			b.WriteString("\\\n")

		case tokWhitespace:
			b.WriteString(t.Content)

		case tokComment:
			b.WriteString(t.Og)

		case tokSemicolon:
			b.WriteByte(';')

		case tokOpenBrace:
			b.WriteByte('{')

		case tokCloseBrace:
			b.WriteByte('}')
		}
	}

	return b.String(), nil
}
