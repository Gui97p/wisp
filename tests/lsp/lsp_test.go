package lsp

import (
	"slices"
	"strings"
	"testing"
)

const shapes = `struct P { int x; int y; }
struct L { P a; P b; }
func take(P p) {}
func make() P {
  return .{x: 1, y: 2};
}
`

func TestHover(t *testing.T) {
	c := start(t)
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"const array", "const [3]int PRI¦MES = .{2, 3, 5};\nfunc main() {}\n", []string{"const [3]int PRIMES = [2, 3, 5]", "read-only data"}},
		{"const struct", "struct Pair { int a; int b; }\nconst Pair OR¦IGIN = .{a: 1, b: 2};\nfunc main() {}\n", []string{"const Pair ORIGIN = Pair{a: 1, b: 2}"}},
		{"const strings", "const [2]string NA¦MES = .{\"x\", \"y\"};\nfunc main() {}\n", []string{"const [2]string NAMES = [\"x\", \"y\"]"}},
		{"const long array", "const BIG = static { [20]int t; yield t; };\nfunc main() { int n = len(BI¦G); }\n", []string{"const [20]int BIG", "... 12 more"}},
		{"const scalar", "const N = 3 * 2;\nfunc main() { int n = ¦N; }\n", []string{"const N = 6"}},
		{"const used in a function", "const [2]int T = .{7, 8};\nfunc main() { int n = ¦T[1]; }\n", []string{"const [2]int T = [7, 8]"}},
		{"literal key", shapes + "func main() {\n  P p = .{¦x: 1, y: 2};\n}\n", []string{"int x", "Field of struct `P`"}},
		{"literal key after a comma", shapes + "func main() {\n  P p = .{x: 1, ¦y: 2};\n}\n", []string{"int y", "Field of struct `P`"}},
		{"nested literal key", shapes + "func main() {\n  L l = .{a: .{x: 1, y: 1}, b: .{x: 2, ¦y: 2}};\n}\n", []string{"int y", "Field of struct `P`"}},
		{"key in an array of literals", shapes + "func main() {\n  [2]P ps = .{.{x: 1, y: 1}, .{¦x: 2, y: 2}};\n}\n", []string{"int x", "Field of struct `P`"}},
		{"key of an explicit literal", shapes + "func main() {\n  P p = P{x: 1, ¦y: 2};\n}\n", []string{"int y"}},
		{"key in a return", shapes + "func other() P {\n  return .{¦x: 1, y: 2};\n}\nfunc main() {}\n", []string{"int x"}},
		{"key in an argument", shapes + "func main() {\n  take(.{x: 1, ¦y: 2});\n}\n", []string{"int y"}},
		{"array literal type", shapes + "func main() {\n  let a = [2]¦P{.{x: 1, y: 1}, .{x: 2, y: 2}};\n}\n", []string{"[2]P"}},
		{"conversion type", "func main() {\n  let b = ¦int32(3);\n}\n", []string{"Signed 32-bit"}},
		{"tag", "#na¦ked\nfunc f() {}\nfunc main() {}\n", []string{"#naked", "no frame"}},
		{"expect_abort tag", "#expect_ab¦ort(\"x\")\nfunc f() {}\nfunc main() {}\n", []string{"#expect_abort", "must stop the program"}},
		{"test tag", "#te¦st\nfunc f() {}\nfunc main() {}\n", []string{"#test", "wisp test"}},
		{"label", "func main() {\n  :outer for i in 0..3 {\n    break :ou¦ter;\n  }\n}\n", []string{":outer", "declared on line 2"}},
		{"capture", "func f() ?int { return null; }\nfunc main() {\n  if f() |va¦lue| {\n  }\n}\n", []string{"int value"}},
		{"sizeof", "struct P { int x; }\nconst N = size¦of(P);\nfunc main() {}\n", []string{"func sizeof(x) int", "known at compile time"}},
		{"static keyword", "const N = sta¦tic { yield 3 * 7; };\nfunc main() {}\n", []string{"static { ... } = 21", "while the program compiles", "type `int`"}},
		{"static keyword with an array", "const T = sta¦tic { [3]int t; t[1] = 5; yield t; };\nfunc main() {}\n", []string{"= [0, 5, 0]", "type `[3]int`"}},
		{"static keyword in a function", "func main() {\n  int n = sta¦tic { yield 4; };\n}\n", []string{"static { ... } = 4"}},
		{"array variable", "func main() {\n  [3]int ¦a;\n}\n", []string{"[3]int a"}},
		{"slice variable", "func main() {\n  [3]int a;\n  []int ¦s = a;\n}\n", []string{"[]int s"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": tc.src}).hover()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("hover does not contain %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestDefinition(t *testing.T) {
	c := start(t)
	cases := []struct {
		name string
		src  string
		line int
	}{
		{"const reference", "const [3]int PRIMES = .{2, 3, 5};\nfunc main() {\n  int n = ¦PRIMES[0];\n}\n", 0},
		{"literal key", shapes + "func main() {\n  P p = .{¦x: 1, y: 2};\n}\n", 0},
		{"key in an array of literals", shapes + "func main() {\n  [2]P ps = .{.{x: 1, y: 1}, .{x: 2, ¦y: 2}};\n}\n", 0},
		{"key in a nested literal", shapes + "func main() {\n  L l = .{a: .{¦x: 1, y: 1}, b: .{x: 2, y: 2}};\n}\n", 0},
		{"key in an argument", shapes + "func main() {\n  take(.{x: 1, ¦y: 2});\n}\n", 0},
		{"array literal type", shapes + "func main() {\n  let a = [2]¦P{.{x: 1, y: 1}, .{x: 2, y: 2}};\n}\n", 0},
		{"label", "func main() {\n  :outer for i in 0..3 {\n    break :ou¦ter;\n  }\n}\n", 1},
		{"struct of a variable", shapes + "func main() {\n  ¦P p = .{x: 1, y: 2};\n}\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := c.open(map[string]string{"main.wsp": tc.src})
			got := p.definition()
			if len(got) == 0 {
				t.Fatal("no definition")
			}
			if got[0].Range.Start.Line != tc.line {
				t.Errorf("definition on line %d, want %d", got[0].Range.Start.Line, tc.line)
			}
			if !strings.HasSuffix(got[0].URI, "main.wsp") {
				t.Errorf("definition in %s", got[0].URI)
			}
		})
	}
}

func TestCompletion(t *testing.T) {
	c := start(t)
	both := []string{"x", "y"}
	cases := []struct {
		name    string
		src     string
		want    []string
		exactly bool
		absent  []string
	}{
		{"declaration", shapes + "func main() {\n  P p = .{¦};\n}\n", both, true, nil},
		{"after a key", shapes + "func main() {\n  P p = .{x: 1, ¦};\n}\n", []string{"y"}, true, []string{"x"}},
		{"with a prefix", shapes + "func main() {\n  P p = .{x¦};\n}\n", both, true, nil},
		{"argument", shapes + "func main() {\n  take(.{¦});\n}\n", both, true, nil},
		{"return", shapes + "func other() P {\n  return .{¦};\n}\nfunc main() {}\n", both, true, nil},
		{"assignment", shapes + "func main() {\n  P p;\n  p = .{¦};\n}\n", both, true, nil},
		{"nested field", shapes + "func main() {\n  L l = .{a: .{¦}};\n}\n", both, true, nil},
		{"second nested field", shapes + "func main() {\n  L l = .{a: .{x: 1, y: 1}, b: .{¦}};\n}\n", both, true, nil},
		{"element of an array", shapes + "func main() {\n  [2]P ps = .{.{x: 1, y: 1}, .{¦}};\n}\n", both, true, nil},
		{"optional", shapes + "func main() {\n  ?P p = .{¦};\n}\n", both, true, nil},
		{"explicit struct", shapes + "func main() {\n  P p = P{¦};\n}\n", both, true, nil},
		{"explicit struct after a key", shapes + "func main() {\n  P p = P{y: 1, ¦};\n}\n", []string{"x"}, true, []string{"y"}},
		{"struct fields", shapes + "func main() {\n  P p;\n  p.¦\n}\n", both, true, nil},
		{"field of a constant", "struct P { int x; int y; }\nconst P ORIGIN = .{x: 1, y: 2};\nfunc main() {\n  int n = ORIGIN.¦\n}\n", both, true, nil},
		{"field of an array element", "struct P { int x; int y; }\nconst [2]P LIST = .{.{x: 1, y: 1}, .{x: 2, y: 2}};\nfunc main() {\n  int n = LIST[0].¦\n}\n", both, true, nil},
		{"tags", "#¦\nfunc f() {}\nfunc main() {}\n", []string{"naked", "packed", "deprecated", "must_use", "noreturn", "test", "expect_abort"}, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": tc.src}).completion()
			for _, want := range tc.want {
				if !slices.Contains(got, want) {
					t.Errorf("completion lacks %q: %v", want, got)
				}
			}
			for _, bad := range tc.absent {
				if slices.Contains(got, bad) {
					t.Errorf("completion offers %q: %v", bad, got)
				}
			}
			if tc.exactly && len(got) != len(tc.want) {
				t.Errorf("completion is %v, want exactly %v", got, tc.want)
			}
		})
	}
}

func TestTestFiles(t *testing.T) {
	c := start(t)
	calc := "public func add(int a, int b) int { return a + b; }\ninternal func secret(int n) int { return n * 7; }\n"
	cases := []struct {
		name   string
		test   string
		errors int
		text   string
	}{
		{"a test next to the code sees what is internal", "import \"calc\";\n\n#test(\"secret\")\nfunc secret() {\n  int n = calc.secret(2);\n}\n", 0, ""},
		{"a test with a fallible body", "import \"calc\";\n\n#test\nfunc ok() ! {\n  int n = calc.add(1, 2);\n}\n", 0, ""},
		{"a test with a parameter", "import \"calc\";\n\n#test\nfunc bad(int n) {\n  int x = calc.add(n, 1);\n}\n", 1, "a test takes no parameters"},
		{"a test with a result", "import \"calc\";\n\n#test\nfunc bad() int {\n  return calc.add(1, 1);\n}\n", 1, "a test gives no result"},
		{"expect_abort with its test", "import \"calc\";\n\n#test\n#expect_abort(\"boom\")\nfunc stops() {\n  int x = calc.add(1, 1);\n}\n", 0, ""},
		{"expect_abort without a test", "import \"calc\";\n\n#expect_abort(\"boom\")\nfunc stops() {\n  int x = calc.add(1, 1);\n}\n", 1, "#expect_abort belongs to a test"},
		{"expect_abort text that is not text", "import \"calc\";\n\n#test\n#expect_abort(3)\nfunc stops() {\n  int x = calc.add(1, 1);\n}\n", 1, "the text of #expect_abort"},
		{"a test name that is not text", "import \"calc\";\n\n#test(3)\nfunc bad() {\n  int x = calc.add(1, 1);\n}\n", 1, "the name of a test is a text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := c.openAs("calc.test.wsp", map[string]string{"calc.wsp": calc, "calc.test.wsp": tc.test})
			var errors []string
			for _, d := range p.diagnostics() {
				if d.Severity == 1 {
					errors = append(errors, d.Message)
				}
			}
			if len(errors) != tc.errors {
				t.Fatalf("%d errors, want %d: %v", len(errors), tc.errors, errors)
			}
			if tc.errors > 0 && !strings.Contains(errors[0], tc.text) {
				t.Errorf("error %q does not contain %q", errors[0], tc.text)
			}
		})
	}
}

func TestDiagnostics(t *testing.T) {
	c := start(t)
	cases := []struct {
		name   string
		src    string
		errors int
		text   string
	}{
		{"clean", shapes + "func main() {\n  P p = .{x: 1, y: 2};\n  take(p);\n}\n", 0, ""},
		{"a literal with no target", shapes + "func main() {\n  let p = .{x: 1};\n}\n", 1, "cannot infer the type of .{...}"},
		{"too many values", "func main() {\n  [2]int a = .{1, 2, 3};\n  int n = a[0];\n}\n", 1, "takes at most 2 values"},
		{"an unknown key", shapes + "func main() {\n  P p = .{z: 1};\n  take(p);\n}\n", 1, "unknown field z"},
		{"names mixed with values", "func main() {\n  [2]int a = .{1, x: 2};\n  int n = a[0];\n}\n", 1, "either values or"},
		{"a size with nothing to count", "func main() {\n  [_]int a;\n}\n", 1, "[_] needs a value to count"},
		{"a constant index out of range", "const [3]int T = .{1, 2, 3};\nconst A = T[5];\nfunc main() { int n = A; }\n", 1, "index out of range"},
		{"a constant array of the wrong size", "const [2]int D = .{1, 2, 3};\nfunc main() { int n = D[0]; }\n", 1, "takes at most 2 values"},
		{"an index error inside static", "const B = static { [2]int x; x[3] = 1; yield x[0]; };\nfunc main() { int n = B; }\n", 1, "out of range"},
		{"division by zero inside static", "const C = static { int z = 0; yield 5 / z; };\nfunc main() { int n = C; }\n", 1, "division by zero"},
		{"the address of a constant", "const [3]int T = .{1, 2, 3};\nfunc main() {\n  *int p = &T[0];\n}\n", 1, "cannot take the address of the constant T"},
		{"a slice of a constant", "const [3]int T = .{1, 2, 3};\nfunc main() {\n  []int s = T[:];\n}\n", 1, "cannot slice the constant T"},
		{"a slice literal with elements", "func main() {\n  []int s = []int{1, 2};\n}\n", 1, "a slice literal has no elements"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var errors []string
			for _, d := range c.open(map[string]string{"main.wsp": tc.src}).diagnostics() {
				if d.Severity == 1 {
					errors = append(errors, d.Message)
				}
			}
			if len(errors) != tc.errors {
				t.Fatalf("%d errors, want %d: %v", len(errors), tc.errors, errors)
			}
			if tc.errors > 0 && !strings.Contains(errors[0], tc.text) {
				t.Errorf("error %q does not contain %q", errors[0], tc.text)
			}
		})
	}
}
