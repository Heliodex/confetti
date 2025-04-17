import type { Directive } from "./parse"

export function testFormat(p: Directive[], depth: number): string {
	const b: string[] = []

	const indent = "    ".repeat(depth)

	for (const d of p) {
		b.push(indent)
		for (let i = 0; i < d.Arguments.length; i++) {
			const a = d.Arguments[i]
			b.push(`<${a}>`)
			if (i < d.Arguments.length - 1) b.push(" ")
		}

		if (d.Subdirectives.length === 0) {
			b.push("\n")
			continue
		}

		b.push(` [\n${testFormat(d.Subdirectives, depth + 1) + indent}]\n`)
	}

	return b.join("")
}
