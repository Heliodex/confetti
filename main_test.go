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
	ts, err := lex(c.Input)
	if err != nil {
		t.Fatalf("Failed to lex input: %v", err)
	}

	p, err := parse(ts)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	tf, err := testFormat(p)
	if err != nil {
		t.Fatalf("Failed to format output: %v", err)
	}

	if tf != c.Output {
		// print location of the mismatch
		for pos := 0; pos < min(len(tf), len(c.Output)); pos++ {
			if tf[pos] != c.Output[pos] {
				t.Logf("Mismatch at position %d, expected %s, got %s", pos,
					fmt.Sprintf("%q", c.Output[max(pos-10, 0):pos+2]),
					fmt.Sprintf("%q", tf[max(pos-10, 0):pos+2]))
				break
			}
		}

		t.Fatalf("Output mismatch\n-- Expected:\n%s\n-- Got:\n%s", c.Output, tf)
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

	for _, c := range cases {
		t.Log("Test case", c.Name)
		runTest(t, c)
	}
}
