import { expect, test } from "bun:test"
import { readdirSync } from "node:fs"
import Load, { type Extensions } from "./main"
import { testFormat } from "./testformat"

const testsDir = "../confetti/tests/conformance"

type testCase = {
	Name: string
	Input?: string
	Output?: string
	Extensions: Extensions
}

async function getCases(): Promise<testCase[]> {
	const cases: testCase[] = []
	const dir = readdirSync(testsDir).sort()

	const addTest = async (c: testCase, n: string, v: string) => {
		// read file
		const data = await Bun.file(`${testsDir}/${n}.${v}`).text()

		const strdata = data.replaceAll("\r\n", "\n")

		if (v === "conf") c.Input = strdata
		else if (v === "pass" || v === "fail") c.Output = strdata
		else if (v.startsWith("ext_")) {
			if (!c.Extensions) c.Extensions = {}

			switch (v.slice(4)) {
				case "c_style_comments":
					c.Extensions.CStyleComments = strdata
					break
				case "expression_arguments":
					c.Extensions.ExpressionArguments = strdata
					break
				case "punctuator_arguments":
					c.Extensions.PunctuatorArguments = strdata
					break
				default:
					throw new Error(`unknown extension ${v}`)
			}
		} else throw new Error(`unknown file type ${v}`)
	}

	dirloop: for (const entry of dir) {
		const split = entry.split(".")
		const [n, v] = split
		if (!n || !v) continue

		// search for the test case with the same name
		for (const c of cases) {
			if (c.Name !== n) continue
			await addTest(c, n, v)
			continue dirloop
		}

		const c: testCase = {
			Name: n,
			Extensions: {},
		}
		await addTest(c, n, v)
		cases.push(c)
	}

	for (const c of cases) {
		if (c.Input === undefined)
			throw new Error(`Test case ${c.Name} is missing input`)
		if (c.Output === undefined)
			throw new Error(`Test case ${c.Name} is missing output`)
	}

	return cases
}

function runConformanceTest(c: testCase) {
	const rin = c.Input
	const rout = c.Output
	const exts = c.Extensions
	if (!rin || !rout) return

	let out = ""
	try {
		out = testFormat(Load(rin, exts), 0)
	} catch (e) {
		if (!(e instanceof Error)) throw e
		out = `error: ${e.message}\n`
	}

	if (out !== rout)
		console.log(
			`Test case ${c.Name} failed:`,
			"\nInput:\n",
			Buffer.from(rin),
			"\nExpected:\n",
			Buffer.from(rout),
			"\nGot:\n",
			Buffer.from(out)
		)
	expect(out).toEqual(rout)
}

test("conformance", async () => {
	const cases = await getCases()

	for (let i = 144; i < cases.length; i++) {
		const c = cases[i]
		if (!c) continue

		console.log(
			`Test case ${i + 1}\nconfetti/tests/conformance/${c.Name}.conf`
		)
		runConformanceTest(c)
	}
})
