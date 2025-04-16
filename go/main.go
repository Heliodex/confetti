// package confetti-go implements the Confetti configuration language.
package confetti

import "fmt"

type extension uint8

const (
	_ extension = iota
	ExtCStyleComments
	ExtExpressionArguments
	ExtPunctuatorArguments
)

type Extensions map[extension]string

func (e Extensions) Has(ext extension) bool {
	_, ok := e[ext]
	return ok
}

func Load(conf string, exts Extensions) ([]Directive, error) {
	ts, err := lex(conf, exts)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	p, err := parse(ts, exts)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	return p, nil
}
