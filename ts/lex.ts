import { isUtf8 } from "node:buffer"
import type { Extensions } from "./main"

const lineTerminators = [0x0a, 0x0b, 0x0c, 0x0d, 0x85, 0x2028, 0x2029]

const isLineTerminator = (r: number): boolean => lineTerminators.includes(r)

const whitespaceRegex = /[\s]/

// all unicode chars with whitespace property
const isWhitespace = (r: number): boolean =>
	!isLineTerminator(r) && whitespaceRegex.test(String.fromCharCode(r))

// characters not in any Unicode category
const isUnassigned = (r: number): boolean => r >= 0x40000 && r <= 0xeffff
const isControl = (r: number): boolean =>
	(r <= 0x1f || (r >= 0x7f && r <= 0x9f)) &&
	!isLineTerminator(r) &&
	!whitespaceRegex.test(String.fromCharCode(r))
const isSurrogate = (r: number): boolean => r >= 0xd800 && r <= 0xdfff

// surrogate, private use, unassigned
const isForbidden = (r: number): boolean =>
	isControl(r) || isSurrogate(r) || r > 0x10ffff || isUnassigned(r)

// " # ; { }
const reserved = [0x22, 0x23, 0x3b, 0x7b, 0x7d]

const isReserved = (r: number, exts: Extensions): boolean =>
	reserved.includes(r) ||
	("ExpressionArguments" in exts && r === 0x28) /* ( */

const errForbidden = new Error("illegal character")

class stream {
	src: string
	pos = 0

	constructor(src: string) {
		this.src = src
	}

	reading(): boolean {
		return this.pos < this.src.length
	}

	current(): number {
		const c = this.src[this.pos]
		if (!c) throw new Error("EOF")

		const r = c.charCodeAt(0)
		if (isForbidden(r)) {
			// get illegal character as U+XXXX
			if (r < 0x10000)
				throw new Error(
					`${errForbidden} U+${r.toString(16).padStart(4, "0")}`
				)
			throw new Error(`${errForbidden} U+${r.toString(16)}`)
		}

		return r
	}

	increment(n: number) {
		this.pos += n
	}

	next(n: number): number {
		return this.src[this.pos + n]?.charCodeAt(0) || 0
	}
}

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
	Og?: string
}

// A directive “argument” shall be a sequence of one or more characters from the argument character set. The argument character set shall consist of any Unicode scalar value excluding characters from the white space, line terminator, reserved punctuator, and forbidden character sets.
const argumentOk = (r: number, exts: Extensions): boolean =>
	!isWhitespace(r) && !isLineTerminator(r) && !isReserved(r, exts)
const quotedArgumentOk = (r: number): boolean =>
	!isLineTerminator(r) && r !== 0x22 // "
const tripleQuotedArgumentOk = (r: number): boolean => r !== 0x22 // "

const errIncompleteEscape = new Error("incomplete escape sequence")
const errIllegalEscape = new Error("illegal escape character")
const errUnclosedQuoted = new Error("unclosed quoted")

export function lex(input: string, exts: Extensions): token[] {
	if (!isUtf8(Buffer.from(input))) throw new Error("malformed UTF-8")

	let src = input
	const ts: token[] = []

	// remove BOMs
	if (src.startsWith("\ufeff")) {
		ts.push({ Type: "Unicode", Content: "\ufeff" })
		src = src.slice(1)
	} else if (src.startsWith("\ufffe")) {
		ts.push({ Type: "Unicode", Content: "\ufffe" })
		src = src.slice(1)
	}

	// remove ^Z
	let removeCtrlZ = false
	if (src.endsWith("\u001a")) {
		removeCtrlZ = true
		src = src.slice(0, -1)
	}

	// check for forbidden characters must be done based on token/location
	for (const s = new stream(src); s.reading(); ) {
		const c = s.current()

		const op = s.pos
		if (isLineTerminator(c)) {
		} else if (isWhitespace(c)) {
		} else if (
			"CStyleComments" in exts &&
			c === 0x2f &&
			s.next(1) === 0x2f //
		) {
			// C-style comment
		} else if (c === 0x23 /* # */) {
			// comment until end of line
		} else if (
			"CStyleComments" in exts &&
			c === 0x2f &&
			s.next(1) === 0x2a /* */
		) {
			// block comment
		} else if (c === 0x3b /* ; */) {
		} else if (c === 0x7b /* { */) {
		} else if (c === 0x7d /* } */) {
		} else if (c === 0x5c /* \ */ && isLineTerminator(s.next(1))) {
		} else if ("ExpressionArguments" in exts && c === 0x28 /* ( */) {
			// } else if ("PunctuatorArguments" in exts && ) {
		} else if (c === 0x22 && s.next(1) === 0x22 && s.next(2) === 0x22) {
			// triple quated argument
		} else if (c === 0x22) {
			// quoted argument
		} else {
			// unquoted argument
		}
	}

	if (removeCtrlZ) ts.push({ Type: "Unicode", Content: "\u001a" })

	return ts
}
