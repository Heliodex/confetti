import type { Extensions } from "./main"

type tokenType =
	| "Unicode"
	| "0qArgument"
	| "1qArgument"
	| "3qArgument"
	| "Newline"
	| "LineContinuation"
	| "Whitespace"
	| "Comment"
	| "Semicolon"
	| "OpenBrace"
	| "CloseBrace"

export type token = {
	Type: tokenType
	Content: string
	Og: string
}

export function lex(src: string, exts: Extensions): token[] {
	const ts: token[] = []

	return ts
}
