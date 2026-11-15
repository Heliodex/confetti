package confetti

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const testsDir = "../confetti/tests/conformance"

type testCase struct {
	Name          string
	Input, Output *string
	Extensions    Extensions
}

func getCases(t *testing.T) (cases []*testCase, err error) {
	dir, err := os.ReadDir(testsDir)
	if err != nil {
		t.Fatalf("Failed to read tests directory: %v", err)
	}

	addTest := func(c *testCase, n, v string) error {
		// read file
		data, err := os.ReadFile(testsDir + "/" + n + "." + v)
		if err != nil {
			return fmt.Errorf("failed to read file %s.%s: %w", n, v, err)
		}

		strdata := strings.ReplaceAll(string(data), "\r\n", "\n")

		switch {
		case v == "conf":
			c.Input = &strdata
		case v == "pass", v == "fail":
			c.Output = &strdata
		case strings.HasPrefix(v, "ext_"):
			if c.Extensions == nil {
				c.Extensions = make(Extensions, 1)
			}

			switch v[4:] {
			case "c_style_comments":
				c.Extensions[ExtCStyleComments] = strdata
			case "expression_arguments":
				c.Extensions[ExtExpressionArguments] = strdata
			case "punctuator_arguments":
				c.Extensions[ExtPunctuatorArguments] = strdata
			default:
				return fmt.Errorf("unknown extension %s", v[4:])
			}
		default:
			return fmt.Errorf("unknown file type %s", v)
		}

		return nil
	}

dirloop:
	for _, entry := range dir {
		split := strings.Split(entry.Name(), ".")
		n, v := split[0], split[1]

		// search for the test case with the same name
		for _, c := range cases {
			if c.Name != n {
				continue
			} else if err = addTest(c, n, v); err != nil {
				return
			}
			continue dirloop
		}

		c := &testCase{Name: n}
		if err = addTest(c, n, v); err != nil {
			return
		}
		cases = append(cases, c)
	}

	for _, c := range cases {
		if c.Input == nil {
			t.Fatalf("Test case %s is missing input", c.Name)
		} else if c.Output == nil {
			t.Fatalf("Test case %s is missing output", c.Name)
		}
	}

	return
}

func runConformanceTest(c *testCase, t *testing.T) {
	rin, rout, exts := *c.Input, *c.Output, c.Extensions

	var out string
	if p, err := Load(rin, exts); err != nil {
		out = err.Error() + "\n"
	} else {
		out = testFormat(p, 0)
	}

	if rout != out {
		t.Fatalf("Output mismatch\n-- Expected:\n%s\n-- Got:\n%s", rout, out)
	}
}

func TestConformance(t *testing.T) {
	cases, err := getCases(t)
	if err != nil {
		t.Fatalf("Failed to get test cases: %v", err)
	}

	for i, c := range cases {
		if i != 144 {
			continue
		}
		
		t.Logf("Test case %d\nconfetti/tests/conformance/%s.conf", i+1, c.Name)
		runConformanceTest(c, t)
	}
}

func runReformatTest(c *testCase, t *testing.T) {
	rin, exts := *c.Input, c.Extensions

	var out string

	ts, err := lex(rin, exts)
	if err != nil {
		t.Fatal(err)
	} else if out, err = testReformat(ts); err != nil {
		t.Fatal(err)
	} else if rin != out {
		t.Fatalf("Output mismatch\n-- Expected:\n%s\n-- Got:\n%s", rin, out)
	}
}

func TestReformat(t *testing.T) {
	cases, err := getCases(t)
	if err != nil {
		t.Fatalf("Failed to get test cases: %v", err)
	}

	for i, c := range cases {
		if strings.HasPrefix(*c.Output, "error:") {
			continue
		}
		t.Logf("Test case %d\nconfetti/tests/conformance/%s.conf", i+1, c.Name)
		runReformatTest(c, t)
	}
}
