import { expect, test } from "bun:test"
import load from "./main"

test("conformance", () => {
	const res = load("test")

	expect(res).toEqual([])
})
