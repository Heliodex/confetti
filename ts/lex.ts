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

const isSurrogate = (r: number): boolean => r >= 0xd800 && r <= 0xdfff

// characters not in any Unicode category
const isUnassigned = (r: number): boolean => r >= 0x40000 && r <= 0xeffff

// surrogate, private use, unassigned
const isForbidden = (r: number): boolean =>
	isControl(r) || isSurrogate(r) || r > 0x10ffff || isUnassigned(r)

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
	Content?: string
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

function checkEscape(s: stream, r: number, quoted: number): [string, boolean] {
	if (r !== 0x5c /* \ */) return [String.fromCharCode(r), false]

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
		return ["", true] // r = "" used to signify line terminatosr
	}

	return [String.fromCharCode(c), true]
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
		if (l <= rest.length && rest.slice(0, l) === p) return l
	}

	return 0
}

function lex0qArgument(s: stream, exts: Extensions): [string, string] {
	let arg = ""
	let ogarg = ""

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
		if (escd) ogarg += "\\"

		arg += ec
		ogarg += ec
		s.increment(1)
	}

	return [arg, ogarg]
}

function lex1qArgument(s: stream): [string, string] {
	let arg = ""
	let ogarg = ""

	for (; s.reading(); s.increment(1)) {
		const [c, err] = s.current()
		if (err.startsWith(errForbidden)) throw new Error(errForbidden)
		if (!quotedArgumentOk(c)) {
			if (c !== 0x22 /* " */) throw errUnclosedQuoted

			s.increment(1)
			return [arg, ogarg]
		}

		const [ec, escd] = checkEscape(s, c, 1)
		if (escd) ogarg += "\\"

		if (ec === "") {
			// escaped line terminators allowed in quoted arguments
			const [nc, _] = s.current()
			ogarg += String.fromCharCode(nc)
			continue
		}
		arg += ec
		ogarg += ec
	}

	throw errUnclosedQuoted
}

function lex3qArgument(s: stream): [string, string] {
	let arg = ""
	let ogarg = ""

	let endsMatched = 0
	while (s.reading()) {
		const [c, err] = s.current()
		if (err.startsWith(errForbidden)) throw new Error(errForbidden)
		if (!tripleQuotedArgumentOk(c)) {
			if (c !== 0x22 /* " */) throw errUnclosedQuoted

			ogarg += String.fromCharCode(c)
			s.increment(1)

			if (endsMatched === 2) return [arg, ogarg.slice(0, -3)]
			endsMatched++
			continue
		}
		if (endsMatched > 0) {
			arg += '"'.repeat(endsMatched)
			endsMatched = 0
			continue
		}

		const [ec, escd] = checkEscape(s, c, 3)
		if (escd) ogarg += "\\"

		arg += ec
		ogarg += ec
		s.increment(1)
	}

	throw errUnclosedQuoted
}

function str2codepoints(str: string): number[] {
	const codepoints: number[] = []
	for (let i = 0; i < str.length; i++) {
		const code = str.codePointAt(i)
		if (code === undefined) continue
		codepoints.push(code)
		if (code > 0xffff) i++ // surrogate pair
	}
	return codepoints
}

export function lex(input: string, exts: Extensions): token[] {
	if (Buffer.from(input).toString("utf-8").includes("\ufffd"))
		// replacement character
		throw new Error("malformed UTF-8")

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
		const [c, err] = s.current()
		if (err) throw new Error(err)

		const op = s.pos
		if (isLineTerminator(c)) {
			s.increment(1)
			ts.push({ Type: "Newline", Content: String.fromCharCode(c) })
		} else if (isWhitespace(c)) {
			s.increment(1)
			ts.push({ Type: "Whitespace", Content: String.fromCharCode(c) })
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
			const content = src.slice(op + 2, s.pos)
			ts.push({ Type: "Comment", Content: content, Og: `#${content}` })
		} else if (c === 0x23 /* # */) {
			// comment until end of line
			while (true) {
				s.increment(1)
				const [c, err] = s.current()
				if (err.startsWith(errForbidden)) throw new Error(errForbidden)
				if (err || isLineTerminator(c)) break
			}
			const content = src.slice(op + 1, s.pos)
			ts.push({ Type: "Comment", Content: content, Og: `#${content}` })
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
				if (err) {
					console.log(err)
					throw new Error("unterminated multi-line comment")
				}
				if (c === 0x2a /* * */ && s.next(1) === 0x2f /* / */) break
			}
			const content = src.slice(op + 2, s.pos)
			ts.push({ Type: "Comment", Content: content, Og: `/*${content}*/` })
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
			for (let depth = 0; ; ) {
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
			const content = src.slice(op + 1, s.pos)
			ts.push({
				Type: "0qArgument",
				Content: content,
				Og: `(${content})`,
			})
			s.increment(1) // )
		} else if (
			exts.PunctuatorArguments !== undefined &&
			getPunctuator(s, exts.PunctuatorArguments) !== 0
		) {
			// read punctuator as argument
			s.increment(getPunctuator(s, exts.PunctuatorArguments))
			const content = src.slice(op, s.pos)
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

	if (removeCtrlZ) ts.push({ Type: "Unicode", Content: "\u001a" })

	return ts
}
