package lsp

import (
	"slices"
	"strings"
	"testing"
)

const noisy = "import \"std/math\";\n\nfunc unused() {}\n\nfunc param(int a, int b) int {\n  return a;\n}\n\nfunc main() {\n  int x = 1;\n}\n"

func warningsOf(p *project) []string {
	var out []string
	for _, d := range p.diagnostics() {
		if d.Severity == 2 {
			out = append(out, d.Message)
		}
	}
	return out
}

func TestAllowTurnsOffWarningsByName(t *testing.T) {
	c := start(t)

	t.Run("without it the file warns", func(t *testing.T) {
		got := strings.Join(warningsOf(c.open(map[string]string{"main.wsp": noisy})), "|")
		for _, want := range []string{"import math is never used", "function unused is never used", "parameter b is never used", "variable x is never used"} {
			if !strings.Contains(got, want) {
				t.Errorf("no warning %q in %q", want, got)
			}
		}
	})

	for _, tc := range []struct {
		name, directive string
		silenced        []string
		kept            []string
	}{
		{"unused_variables", "#allow unused_variables\n", []string{"parameter b", "variable x"}, []string{"import math", "function unused"}},
		{"unused_imports", "#allow unused_imports\n", []string{"import math"}, []string{"parameter b", "variable x", "function unused"}},
		{"dead_code", "#allow dead_code\n", []string{"function unused"}, []string{"import math", "parameter b", "variable x"}},
		{"several names", "#allow unused_variables, unused_imports, dead_code\n", []string{"import math", "function unused", "parameter b", "variable x"}, nil},
		{"every warning", "#allow warnings\n", []string{"import math", "function unused", "parameter b", "variable x"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(warningsOf(c.open(map[string]string{"main.wsp": tc.directive + noisy})), "|")
			for _, gone := range tc.silenced {
				if strings.Contains(got, gone) {
					t.Errorf("%q is still reported in %q", gone, got)
				}
			}
			for _, stays := range tc.kept {
				if !strings.Contains(got, stays) {
					t.Errorf("%q should still be reported, got %q", stays, got)
				}
			}
		})
	}

	t.Run("it comes after #standalone", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "#standalone\n#allow unused_variables\n\nfunc main() {\n  int x = 1;\n}\n"})
		if got := warningsOf(p); len(got) != 0 {
			t.Errorf("warnings: %v", got)
		}
	})

	t.Run("it only covers its own file", func(t *testing.T) {
		quiet := "#allow unused_variables, dead_code\n\nfunc quiet(int a) {\n  int y = 2;\n}\n"
		files := map[string]string{"main.wsp": "func main() {\n  int x = 1;\n}\n", "quiet.wsp": quiet}
		if got := warningsOf(c.openAs("main.wsp", files)); !slices.ContainsFunc(got, func(m string) bool { return strings.Contains(m, "variable x") }) {
			t.Errorf("the other file stopped warning: %v", got)
		}
		if got := warningsOf(c.openAs("quiet.wsp", files)); len(got) != 0 {
			t.Errorf("the file with the directive still warns: %v", got)
		}
	})
}

func TestAllowRejectsWhatItDoesNotKnow(t *testing.T) {
	c := start(t)
	for _, tc := range []struct{ name, directive, want string }{
		{"a misspelled name", "#allow unused_variabels\n", "unknown warning unused_variabels"},
		{"no names", "#allow\n", "#allow takes the names"},
		{"a trailing comma", "#allow unused_variables,\n", "#allow takes the names"},
		{"a missing comma", "#allow unused_variables dead_code\n", "expected a comma"},
		{"parentheses", "#allow(unused_variables)\n", "#allow takes the names"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := c.open(map[string]string{"main.wsp": tc.directive + "func main() {}\n"})
			found := false
			for _, d := range p.diagnostics() {
				if d.Severity == 1 && strings.Contains(d.Message, tc.want) {
					found = true
				}
			}
			if !found {
				t.Errorf("no error with %q", tc.want)
			}
		})
	}
}

func TestAllowInTheEditor(t *testing.T) {
	c := start(t)

	t.Run("completion offers the names", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": "#allow ¦\nfunc main() {}\n"}).completion()
		for _, want := range []string{"unused_variables", "unused_imports", "dead_code", "unreachable_code", "warnings"} {
			if !slices.Contains(got, want) {
				t.Errorf("%q is missing in %v", want, got)
			}
		}
	})
	t.Run("completion leaves out the names already written", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": "#allow dead_code, ¦\nfunc main() {}\n"}).completion()
		if slices.Contains(got, "dead_code") || !slices.Contains(got, "unused_imports") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("the directive is offered at the top of a file", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": "#al¦\nfunc main() {}\n"}).completion()
		if !slices.Contains(got, "allow") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("hover explains it", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": "#al¦low unused_variables\nfunc main() {}\n"}).hover()
		if !strings.Contains(got, "Turns off warnings") {
			t.Errorf("hover %q", got)
		}
	})
	t.Run("the formatter keeps it", func(t *testing.T) {
		src := "#standalone\n#allow unused_variables, dead_code\n\nfunc main() {}\n"
		p := c.open(map[string]string{"main.wsp": src})
		if out, ok := p.formatted(); ok && out != src {
			t.Errorf("formatted wrongly:\n%s", out)
		}
	})
}
