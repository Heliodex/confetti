import type { token } from "./lex"
import type { Extensions } from "./main"

export type Directive = {
	arguments: string[]
	subdirectives: Directive[]
}

function isEqual(a: Directive, b: Directive): boolean {
	if (a.arguments.length !== b.arguments.length) return false

	for (let i = 0; i < a.arguments.length; i++)
		if (a.arguments[i] !== b.arguments[i]) return false

	if (a.subdirectives.length !== b.subdirectives.length) return false

	for (let i = 0; i < a.subdirectives.length; i++) {
		const subA = a.subdirectives[i]
		const subB = b.subdirectives[i]
		if (!(subA && subB && isEqual(subA, subB))) return false
	}

	return true
}

export function parse(ts: token[], exts: Extensions): Directive[] {
	const p: Directive[] = []

	return p
}
