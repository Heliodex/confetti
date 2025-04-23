import type { token, tokenType } from "./lex"
import type { Extensions } from "./main"

export type Directive = {
	Arguments: number[][]
	Subdirectives: Directive[]
}

function isEqual(a: Directive, b: Directive): boolean {
	if (a.Arguments.length !== b.Arguments.length) return false

	for (let i = 0; i < a.Arguments.length; i++)
		if (a.Arguments[i] !== b.Arguments[i]) return false

	if (a.Subdirectives.length !== b.Subdirectives.length) return false

	for (let i = 0; i < a.Subdirectives.length; i++) {
		const subA = a.Subdirectives[i]
		const subB = b.Subdirectives[i]
		if (!(subA && subB && isEqual(subA, subB))) return false
	}

	return true
}

export function parse(ts: token[], exts: Extensions): Directive[] {
	const p: Directive[] = []

	let current: Directive = { Arguments: [], Subdirectives: [] }
	function push() {
		if (!current || current.Arguments.length === 0) return
		p.push(current)
		current = { Arguments: [], Subdirectives: [] }
	}

	let i = 0
	function prevSignificant(): tokenType {
		for (let ci = i - 1; ci > 0; ci--) {
			const prev = ts[ci]?.Type
			if (prev && prev !== "Whitespace" && prev !== "Comment") return prev
		}
		return "Unicode"
	}

	for (; i < ts.length; i++) {
		const t = ts[i]
		if (!t) continue
		switch (t.Type) {
			case "0qArgument":
			case "1qArgument":
			case "3qArgument":
				current.Arguments.push(t.Content || [])
				break
			case "Semicolon": {
				const prev = prevSignificant()
				if (
					prev === "Semicolon" ||
					prev === "Newline" ||
					prev === "LineContinuation"
				)
					throw new Error("unexpected ';'")

				push()
				break
			}
			case "Newline":
				push()
				break
			case "OpenBrace": {
				if (i === ts.length - 1 || prevSignificant() === "Semicolon")
					throw new Error("unexpected '{'")

				// Get all tokens until next close brace
				i++
				const si = i
				for (let depth = 0; i < ts.length; i++) {
					// escapes should be dealt with in the lexer
					const t2 = ts[i]
					if (!t2) break
					if (t2.Type === "OpenBrace") depth++
					else if (t2.Type === "CloseBrace") {
						if (depth-- === 0) break
					} else if (i === ts.length - 1)
						throw new Error("expected '}'")
				}

				const subp = parse(ts.slice(si, i), exts)
				if (current.Arguments.length === 0) {
					// push to the previous directive
					const prev = p.at(-1)
					if (prev) prev.Subdirectives = subp
					break
				}

				current.Subdirectives = subp
				push()
				break
			}
			case "CloseBrace":
				throw new Error("found '}' without matching '{'")
			case "LineContinuation":
				if (current.Arguments.length === 0)
					throw new Error("unexpected line continuation")
		}
	}

	push()
	return p
}
