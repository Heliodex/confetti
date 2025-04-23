import type { Directive } from "./parse"

export function testFormat(p: Directive[], depth: number): string {
	const b: number[] = []

	for (const d of p) {
		for (let i = 0; i < depth * 4; i++) b.push(0x20)
		for (let i = 0; i < d.Arguments.length; i++) {
			const a = d.Arguments[i] || []
			b.push(0x3c) // <
			for (const c of a) b.push(c)

			b.push(0x3e) // >
			if (i < d.Arguments.length - 1) b.push(0x20) // " "
		}

		if (d.Subdirectives.length === 0) {
			b.push(0x0a) // "\n"
			continue
		}

		b.push(0x20, 0x5b, 0x0a) // " [\n"
		b.push(
			...testFormat(d.Subdirectives, depth + 1)
				.split("")
				.map(c => c.charCodeAt(0))
		)

		for (let i = 0; i < depth * 4; i++) b.push(0x20)

		b.push(0x5d, 0x0a) // "]\n"
	}

	return String.fromCodePoint(...b)
}
