package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const testsDir = "./confetti/tests/suite"

type TestCase struct {
	Name, Input, Output string
}

func runTest(t *testing.T, c *TestCase) {
	var p []Directive
	var out string

	ts, err := lex(c.Input)
	if err != nil {
		out = fmt.Sprintf("error: %s\n", err.Error())
	}

	if err == nil {
		p, err = parse(ts)
		if err != nil {
			out = fmt.Sprintf("error: %s\n", err.Error())
		}
	}

	if err == nil {
		out, err = testFormat(p, 0)
		if err != nil {
			out = fmt.Sprintf("error: %s\n", err.Error())
		}
	}

	if out != c.Output {
		fmt.Println(c.Input)

		// print location of the mismatch
		fmt.Println(len(c.Output), len(out))
		for pos := range min(len(out), len(c.Output)) {
			if out[pos] != c.Output[pos] {
				t.Logf("Mismatch at position %d, expected %q, got %q", pos,
					c.Output[max(pos-10, 0):min(len(c.Output), pos+10)],
					out[max(pos-10, 0):min(len(out), pos+10)])
				break
			}
		}

		t.Fatalf("Output mismatch\n-- Expected:\n%s\n-- Got:\n%s", c.Output, out)
	}
}

func TestConformance(t *testing.T) {
	dir, err := os.ReadDir(testsDir)
	if err != nil {
		t.Fatalf("Failed to read tests directory: %v", err)
	}

	var cases []*TestCase

	addTest := func(c *TestCase, n, v string) error {
		// read file
		data, err := os.ReadFile(testsDir + "/" + n + "." + v)
		if err != nil {
			return fmt.Errorf("failed to read file %s.%s: %w", n, v, err)
		}

		strdata := strings.ReplaceAll(string(data), "\r\n", "\n")

		switch v {
		case "in":
			c.Input = strdata
		case "out", "err":
			c.Output = strdata
		}

		return nil
	}

	for _, entry := range dir {
		split := strings.Split(entry.Name(), ".")
		n, v := split[0], split[1]

		// search for the test case with the same name
		var found bool
		for _, c := range cases {
			if c.Name == n {
				err := addTest(c, n, v)
				if err != nil {
					t.Fatalf("Failed to add test case: %v", err)
				}
				found = true
				break
			}
		}

		if !found {
			c := &TestCase{Name: n}
			err := addTest(c, n, v)
			if err != nil {
				t.Fatalf("Failed to add test case: %v", err)
			}
			cases = append(cases, c)
		}
	}

	for i, c := range cases {
		t.Logf("%-03d Test case %s", i, c.Name)
		runTest(t, c)
	}
}
