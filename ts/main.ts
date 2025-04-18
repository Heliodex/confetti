import { lex } from "./lex"
import { type Directive, parse } from "./parse"

type extension =
	| "CStyleComments"
	| "ExpressionArguments"
	| "PunctuatorArguments"

export type Extensions = { [_ in extension]?: string }

export default (conf: string, exts: Extensions = {}): Directive[] =>
	parse(lex(conf, exts), exts)
