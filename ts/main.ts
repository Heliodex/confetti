import { lex } from "./lex"
import { type Directive, parse } from "./parse"

type extension =
	| "CStyleComments"
	| "ExpressionArguments"
	| "PunctuatorArguments"

export type Extensions = { [_ in extension]?: string }

export default (conf: string, exts: Extensions = {}): Directive[] => {
	const ts = lex(conf, exts)
	const p = parse(ts, exts)

	return p
}
