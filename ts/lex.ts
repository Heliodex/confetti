import type { Extensions } from "./main"

const lineTerminators = [0x0a, 0x0b, 0x0c, 0x0d, 0x85, 0x2028, 0x2029]

const isLineTerminator = (r: number): boolean => lineTerminators.includes(r)

const whitespaceRegex = /\s/

// all unicode chars with whitespace property
const isWhitespace = (r: number): boolean =>
	!isLineTerminator(r) && whitespaceRegex.test(String.fromCharCode(r))

const isControl = (r: number): boolean =>
	(r <= 0x1f || (r >= 0x7f && r <= 0x9f)) &&
	!isLineTerminator(r) &&
	!whitespaceRegex.test(String.fromCharCode(r))

// makes up for js bad unicode handling lel
const isHighSurrogate = (r: number): boolean => r >= 0xd800 && r <= 0xdbff
const isLowSurrogate = (r: number): boolean => r >= 0xdc00 && r <= 0xdfff

// characters not in any Unicode category
const isUnassigned = (r: number): boolean => r >= 0x40000 && r <= 0xeffff

// surrogate, private use, unassigned
const isForbidden = (r: number): boolean =>
	isControl(r) || isHighSurrogate(r) || r > 0x10ffff || isUnassigned(r)

// " # ; { }
const reserved = [0x22, 0x23, 0x3b, 0x7b, 0x7d]

const isReserved = (r: number, exts: Extensions): boolean =>
	reserved.includes(r) ||
	("ExpressionArguments" in exts && r === 0x28) /* ( */

const errForbidden = "illegal character"

class stream {
	src: string
	pos = 0

	constructor(src: string) {
		this.src = src
	}

	reading(): boolean {
		return this.pos < this.src.length
	}

	current(): [number, string] {
		// TODO: wip error type
		const r = this.src.charCodeAt(this.pos)
		if (Number.isNaN(r)) return [0, "EOF"]

		// if high surrogate, add second part (low surrogate)
		if (isHighSurrogate(r)) {
			const next = this.src.charCodeAt(this.pos + 1)
			if (!isLowSurrogate(next))
				return [0, `${errForbidden} U+${r.toString(16).toUpperCase()}`]

			this.pos += 1
			return [0x10000 + ((r - 0xd800) << 10) + (next - 0xdc00), ""]
		}

		if (isForbidden(r)) {
			// get illegal character as U+XXXX
			if (r < 0x10000)
				return [
					0,
					`${errForbidden} U+${r.toString(16).toUpperCase().padStart(4, "0")}`,
				]
			return [0, `${errForbidden} U+${r.toString(16).toUpperCase()}`]
		}

		return [r, ""]
	}

	increment(n: number) {
		this.pos += n
	}

	next(n: number): number {
		return this.src[this.pos + n]?.charCodeAt(0) || 0
	}
}

export type tokenType =
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
	Content?: number[]
	Og?: number[]
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

function checkEscape(s: stream, r: number, quoted: number): [number, boolean] {
	if (r !== 0x5c /* \ */) return [r, false]

	s.increment(1)
	const [c, err] = s.current()
	if (err) {
		// find if error message starts with errForbidden
		if (err.startsWith(errForbidden) || quoted === 0) throw errIllegalEscape
		throw errIncompleteEscape
	}
	if (isWhitespace(c) || isLineTerminator(c)) {
		if (quoted === 3) {
			if (isLineTerminator(c)) throw errIncompleteEscape
			throw errIllegalEscape
		}
		if (quoted === 0 || (quoted === 1 && !isLineTerminator(c)))
			throw errIllegalEscape
		return [0, true] // r = 0 used to signify line terminatosr
	}

	return [c, true]
}

function getPunctuator(s: stream, pstr: string): number {
	let ps = pstr
	ps = ps.replaceAll("\r\n", "\n")
	ps = ps.replaceAll("\r", "\n")
	ps = ps.trim()

	const puncts = ps.split("\n")
	// sort puncts by length descending
	puncts.sort((a, b) => b.length - a.length)

	for (const p of puncts) {
		const rest = s.src.slice(s.pos)
		const l = p.length
		if (l <= rest.length && rest.slice(0, l) === p) {
			console.log("punctuator", p.charCodeAt(0), l)
			return l}
	}

	return 0
}

function lex0qArgument(s: stream, exts: Extensions): [number[], number[]] {
	const arg: number[] = []
	const ogarg: number[] = []

	while (s.reading()) {
		const [c, err] = s.current()
		if (err) throw new Error(err)
		if (
			!argumentOk(c, exts) ||
			(exts.PunctuatorArguments !== undefined &&
				getPunctuator(s, exts.PunctuatorArguments) !== 0)
		)
			return [arg, ogarg]

		const [ec, escd] = checkEscape(s, c, 0)
		if (escd) ogarg.push(0x5c /* \ */)

		arg.push(ec)
		ogarg.push(ec)
		s.increment(1)
	}

	return [arg, ogarg]
}

function lex1qArgument(s: stream): [number[], number[]] {
	const arg: number[] = []
	const ogarg: number[] = []

	while (s.reading()) {
		const [c, err] = s.current()
		if (err.startsWith(errForbidden)) throw new Error(errForbidden)
		if (!quotedArgumentOk(c)) {
			if (c !== 0x22 /* " */) throw errUnclosedQuoted

			s.increment(1)
			return [arg, ogarg]
		}

		const [ec, escd] = checkEscape(s, c, 1)
		if (escd) ogarg.push(0x5c /* \ */)

		if (ec === 0) {
			// escaped line terminators allowed in quoted arguments
			const [nc, _] = s.current()
			ogarg.push(nc)
			continue
		}

		arg.push(ec)
		ogarg.push(ec)
		s.increment(1)
	}

	throw errUnclosedQuoted
}

function lex3qArgument(s: stream): [number[], number[]] {
	const arg: number[] = []
	const ogarg: number[] = []

	let endsMatched = 0
	while (s.reading()) {
		const [c, err] = s.current()
		if (err.startsWith(errForbidden)) throw new Error(errForbidden)
		if (!tripleQuotedArgumentOk(c)) {
			if (c !== 0x22 /* " */) throw errUnclosedQuoted

			ogarg.push(c)
			s.increment(1)

			if (endsMatched === 2) return [arg, ogarg.slice(0, -3)]
			endsMatched++
			continue
		}
		if (endsMatched > 0) {
			for (let i = 0; i < endsMatched; i++) arg.push(0x22 /* " */)
			endsMatched = 0
			continue
		}

		const [ec, escd] = checkEscape(s, c, 3)
		if (escd) ogarg.push(0x5c /* \ */)

		arg.push(ec)
		ogarg.push(ec)
		s.increment(1)
	}

	throw errUnclosedQuoted
}

export function lex(input: string, exts: Extensions): token[] {
	if (input.includes("\ufffd"))
		// replacement character
		throw new Error("malformed UTF-8")

	let src = input
	const ts: token[] = []

	// remove BOMs
	if (src.startsWith("\ufeff")) {
		ts.push({ Type: "Unicode", Content: [0xfeff] })
		src = src.slice(1)
	} else if (src.startsWith("\ufffe")) {
		ts.push({ Type: "Unicode", Content: [0xfffe] })
		src = src.slice(1)
	}

	// remove ^Z
	let removeCtrlZ = false
	if (src.endsWith("\u001a")) {
		removeCtrlZ = true
		src = src.slice(0, -1)
	}

	// character buffer
	const buf: number[] = []
	for (let i = 0; i < src.length; i++) {
		const c = src.charCodeAt(i)
		buf.push(c)
	}

	// check for forbidden characters must be done based on token/location

	for (const s = new stream(src); s.reading(); ) {
		const [c, err] = s.current()
		if (err) throw new Error(err)

		const op = s.pos
		if (isLineTerminator(c)) {
			s.increment(1)
			ts.push({ Type: "Newline", Content: [c] })
		} else if (isWhitespace(c)) {
			s.increment(1)
			ts.push({ Type: "Whitespace", Content: [c] })
		} else if (
			"CStyleComments" in exts &&
			c === 0x2f &&
			s.next(1) === 0x2f //
		) {
			// C-style comment
			for (s.increment(1); ; ) {
				s.increment(1)
				const [c, err] = s.current()
				if (err.startsWith(errForbidden)) throw new Error(errForbidden)
				if (err || isLineTerminator(c)) break
			}
			const content = buf.slice(op + 2, s.pos)
			ts.push({
				Type: "Comment",
				Content: content,
				Og: Array.from(Uint8Array.from(`#${content}`)),
			})
		} else if (c === 0x23 /* # */) {
			// comment until end of line
			while (true) {
				s.increment(1)
				const [c, err] = s.current()
				if (err.startsWith(errForbidden)) throw new Error(errForbidden)
				if (err || isLineTerminator(c)) break
			}
			const content = buf.slice(op + 1, s.pos)
			ts.push({
				Type: "Comment",
				Content: content,
				Og: Array.from(Uint8Array.from(`#${content}`)),
			})
		} else if (
			"CStyleComments" in exts &&
			c === 0x2f &&
			s.next(1) === 0x2a /* */
		) {
			// block comment
			for (s.increment(1); ; ) {
				s.increment(1)
				const [c, err] = s.current()
				if (err.startsWith(errForbidden)) throw new Error(errForbidden)
				if (err) throw new Error("unterminated multi-line comment")
				if (c === 0x2a /* * */ && s.next(1) === 0x2f /* / */) break
			}
			const content = buf.slice(op + 2, s.pos)
			ts.push({
				Type: "Comment",
				Content: content,
				Og: Array.from(Uint8Array.from(`/*${content}*/`)),
			})
			s.increment(2) // */
		} else if (c === 0x3b /* ; */) {
			s.increment(1)
			ts.push({ Type: "Semicolon" })
		} else if (c === 0x7b /* { */) {
			s.increment(1)
			ts.push({ Type: "OpenBrace" })
		} else if (c === 0x7d /* } */) {
			s.increment(1)
			ts.push({ Type: "CloseBrace" })
		} else if (c === 0x5c /* \ */ && isLineTerminator(s.next(1))) {
			s.increment(2)
			ts.push({ Type: "LineContinuation" })
		} else if ("ExpressionArguments" in exts && c === 0x28 /* ( */) {
			// read until corresponding closing parenthesis
			let depth = 0
			while (true) {
				s.increment(1)
				const [c, err] = s.current()
				if (err.startsWith(errForbidden)) throw new Error(errForbidden)
				if (err || isLineTerminator(c))
					throw new Error("incomplete expression")
				if (c === 0x28 /* ( */) depth++
				else if (c === 0x29 /* ) */) {
					if (depth === 0) break
					depth--
				}
			}
			const content = buf.slice(op + 1, s.pos)
			ts.push({
				Type: "0qArgument",
				Content: content,
				Og: Array.from(Uint8Array.from(`(${content})`)),
			})
			s.increment(1) // )
		} else if (
			exts.PunctuatorArguments !== undefined &&
			getPunctuator(s, exts.PunctuatorArguments) !== 0
		) {
			// read punctuator as argument
			s.increment(getPunctuator(s, exts.PunctuatorArguments))
			const content = buf.slice(op, s.pos)

		console.log("punctuator", content)

			ts.push({ Type: "0qArgument", Content: content, Og: content })
		} else if (c === 0x22 && s.next(1) === 0x22 && s.next(2) === 0x22) {
			// triple quated argument
			s.increment(3)
			const [arg, ogarg] = lex3qArgument(s)
			ts.push({ Type: "3qArgument", Content: arg, Og: ogarg })
		} else if (c === 0x22) {
			// quoted argument
			s.increment(1)
			const [arg, ogarg] = lex1qArgument(s)
			ts.push({ Type: "1qArgument", Content: arg, Og: ogarg })
		} else {
			// unquoted argument
			const [arg, ogarg] = lex0qArgument(s, exts)
			ts.push({ Type: "0qArgument", Content: arg, Og: ogarg })
		}
	}

	if (removeCtrlZ) ts.push({ Type: "Unicode", Content: [0x001a] })

	return ts
}
