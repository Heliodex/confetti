package main

import "strings"

func testReformat(ts []Token) (string, error) {
	var b strings.Builder

	for _, t := range ts {
		switch t.Type {
		case TokUnicode:
			b.WriteString(t.Content)

		case Tok0qArgument:
			b.WriteString(t.Og)

		case Tok1qArgument:
			b.WriteString("\"" + t.Og + "\"")

		case Tok3qArgument:
			b.WriteString("\"\"\"" + t.Og + "\"\"\"")

		case TokNewline:
			b.WriteString(t.Content)

		case TokLineContinuation:
			b.WriteString("\\\n")

		case TokWhitespace:
			b.WriteString(t.Content)

		case TokComment:
			b.WriteString(t.Og)

		case TokSemicolon:
			b.WriteByte(';')

		case TokOpenBrace:
			b.WriteByte('{')

		case TokCloseBrace:
			b.WriteByte('}')
		}
	}

	return b.String(), nil
}
