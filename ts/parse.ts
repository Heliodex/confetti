import type { token } from "./lex"
import type { Extensions } from "./main"

export type Directive = {
	Arguments: string[]
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

	return p
}
