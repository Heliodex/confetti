package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const testsDir = "./confetti/tests/suite"

type TestCase struct {
	Name          string
	Input, Output *string
}

func runTest(t *testing.T, c *TestCase) {
	if c.Input == nil {
		t.Fatalf("Test case %s is missing input", c.Name)
	} else if c.Output == nil {
		t.Fatalf("Test case %s is missing output", c.Name)
	}

	rin, rout := *c.Input, *c.Output

	var p []Directive
	var out string

	ts, err := lex(rin)
	if err != nil {
		out = fmt.Sprintf("error: %s\n", err.Error())
	}

	if err == nil {
		if p, err = parse(ts); err != nil {
			out = fmt.Sprintf("error: %s\n", err.Error())
		}
	}

	if err == nil {
		if out, err = testFormat(p, 0); err != nil {
			out = fmt.Sprintf("error: %s\n", err.Error())
		}
	}

	if out != rout {
		fmt.Println(c.Input)

		// print location of the mismatch
		fmt.Println(len(rout), len(out))
		for pos := range min(len(out), len(rout)) {
			if out[pos] != rout[pos] {
				t.Logf("Mismatch at position %d, expected %q, got %q", pos,
					rout[max(pos-10, 0):min(len(rout), pos+10)],
					out[max(pos-10, 0):min(len(out), pos+10)])
				break
			}
		}

		t.Fatalf("Output mismatch\n-- Expected:\n%s\n-- Got:\n%s", rout, out)
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
		case "conf":
			c.Input = &strdata
		case "pass", "fail":
			c.Output = &strdata
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
				if err := addTest(c, n, v); err != nil {
					t.Fatalf("Failed to add test case: %v", err)
				}
				found = true
				break
			}
		}

		if !found {
			c := &TestCase{Name: n}
			if err := addTest(c, n, v); err != nil {
				t.Fatalf("Failed to add test case: %v", err)
			}
			cases = append(cases, c)
		}
	}

	for i, c := range cases {
		t.Logf("%-03d Test case %s", i+1, c.Name)
		runTest(t, c)
	}
}
