package lsp

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
		{"a spread of the wrong element type", "func take(int ...xs) int { return len(xs); }\nfunc main() {\n  []string words = .{\"a\"};\n  int n = take(words...);\n}\n", 1, "passes a slice of string on"},
		{"a spread that is not a slice", "func take(int ...xs) int { return len(xs); }\nfunc main() {\n  int one = 1;\n  int n = take(one...);\n}\n", 1, "passes a slice on"},
		{"a spread into a function without a variadic", "func take(int x) int { return x; }\nfunc main() {\n  []int xs = .{1};\n  int n = take(xs...);\n}\n", 1, "has none"},
		{"a spread that matches", "func take(int ...xs) int { return len(xs); }\nfunc main() {\n  []int xs = .{1};\n  int n = take(0, xs...);\n}\n", 0, ""},
		{"returning a call that gives nothing", "func nothing() {}\nfunc f(int a) {\n  return nothing() if a > 0;\n}\nfunc main() { f(1); }\n", 0, ""},
		{"returning a call that gives a value from a function with none", "func one() int { return 1; }\nfunc f() {\n  return one();\n}\nfunc main() { f(); }\n", 1, "expected 0 return values, got 1"},
		{"a call that gives nothing used as a value", "func nothing() {}\nfunc main() {\n  int x = nothing();\n}\n", 1, "function doesn't expect any return"},
		{"returning a call that gives nothing from a function that gives a value", "func nothing() {}\nfunc f() int {\n  return nothing();\n}\nfunc main() { int x = f(); }\n", 1, ""},
		{"a clean generic program", "struct Box<T> { T value; }\nfunc (*Box<T> b) get() T { return b.value; }\nfunc first<T>(Box<T> b) T { return b.value; }\nfunc main() {\n  Box<int> b = Box<int>{value: 1};\n  int x = first(b) + b.get();\n}\n", 0, ""},
		{"a generic struct with the wrong number of arguments", "struct Box<T> { T value; }\nfunc main() {\n  Box<int, int> b;\n}\n", 1, "Box takes 1 type arguments, got 2"},
		{"a generic struct without arguments", "struct Box<T> { T value; }\nfunc main() {\n  Box b;\n}\n", 1, "Box is generic"},
		{"a type that cannot be inferred", "func make<T>() T { T v; return v; }\nfunc main() {\n  int x = make();\n}\n", 1, "cannot infer T"},
		{"conflicting types in one call", "func same<T>(T a, T b) T { return a; }\nfunc main() {\n  int x = same(1, \"a\");\n}\n", 1, "T is both int and string"},
		{"a bound is not accepted yet", "func f<T: Number>(T a) {}\nfunc main() {}\n", 1, "goes before its name"},
		{"a generic function used as a value", "func id<T>(T a) T { return a; }\nfunc main() {\n  let f = id;\n}\n", 1, "id is generic"},
		{"an error in the body of an instance", "func bad<T>(T a) T { return a.missing; }\nfunc main() {\n  int x = bad(1);\n}\n", 2, "with T = int, instantiated here"},
		{"an operation a bare type parameter does not have", "func add<T>(T a, T b) T { return a + b; }\nfunc main() {}\n", 1, "invalid operator + for T and T"},
		{"an interface value has no zero value", "interface Writer { func write(int n) int; }\nfunc main() {\n  Writer w;\n}\n", 1, "always holds something"},
		{"a type set is not a value", "interface Number { int | float }\nfunc main() {\n  Number n;\n}\n", 1, "cannot be the type of a value"},
		{"a value parameter that is not constant", "struct Ring<T, const int N> { [N]T slots; }\nfunc main() {\n  int n = 3;\n  Ring<int, n> r;\n}\n", 1, "value parameter"},
		{"a generic method on a type that is not generic", "struct P { int x; }\nfunc (*P<T> p) f() {}\nfunc main() {}\n", 1, "not a generic struct"},
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

func TestFormatting(t *testing.T) {
	c := start(t)
	cases := []struct {
		name    string
		src     string
		want    string
		changed bool
	}{
		{"a messy file", "func f() {\n    int a=1;\n}\n", "func f() {\n\tint a = 1;\n}\n", true},
		{"a formatted file", "func f() {\n\tint a = 1;\n}\n", "func f() {\n\tint a = 1;\n}\n", false},
		{"a file with a syntax error", "func f( {\n", "func f( {\n", false},
		{"comments survive", "func f() {   // here\n// alone\n    int a=1;\n}\n", "func f() {  // here\n\t// alone\n\tint a = 1;\n}\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := c.open(map[string]string{"main.wsp": tc.src}).formatted()
			if changed != tc.changed || got != tc.want {
				t.Errorf("changed=%v, text:\n%s\nwant changed=%v:\n%s", changed, got, tc.changed, tc.want)
			}
		})
	}

	t.Run("excluded files", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "func f() {\n    int a=1;\n}\n", "wisp.toml": "[fmt]\nexclude = [\"main.wsp\"]\n"})
		if _, changed := p.formatted(); changed {
			t.Errorf("an excluded file was formatted")
		}
	})
}

func TestOrganizeImports(t *testing.T) {
	c := start(t)
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"an unused import is removed", "import \"std/math\";\n\nfunc main() {}\n", "\nfunc main() {}\n"},
		{"a used import stays", "import \"std/math\";\n\nfunc main() {\n  int n = math.max(1, 2);\n}\n", ""},
		{"a missing import is added", "func main() {\n  int n = math.max(1, 2);\n}\n", "import \"std/math\";\n\nfunc main() {\n  int n = math.max(1, 2);\n}\n"},
		{"one is removed and one is added", "import \"std/fmt\";\n\nfunc main() {\n  int n = math.max(1, 2);\n}\n", "import \"std/math\";\n\nfunc main() {\n  int n = math.max(1, 2);\n}\n"},
		{"two missing imports go in as a block", "func main() {\n  int n = math.max(1, 2);\n  string s = strings.trimSpace(\" x \");\n}\n", "import \"std/math\";\nimport \"std/strings\";\n\nfunc main() {\n  int n = math.max(1, 2);\n  string s = strings.trimSpace(\" x \");\n}\n"},
		{"nothing to do", "func main() {}\n", ""},
		{"a file with a syntax error", "import \"std/math\";\nfunc main( {\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := c.open(map[string]string{"main.wsp": tc.src})
			actions := p.codeActions("source.organizeImports")
			if tc.want == "" {
				if len(actions) != 0 {
					t.Fatalf("an action was offered: %+v", actions)
				}
				return
			}
			if len(actions) != 1 || actions[0].Kind != "source.organizeImports" {
				t.Fatalf("actions: %+v", actions)
			}
			if got := actions[0].apply(p, tc.src); got != tc.want {
				t.Errorf("organized wrongly\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}

	t.Run("it is also offered without a filter", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "import \"std/math\";\n\nfunc main() {}\n"})
		found := false
		for _, a := range p.codeActions() {
			if a.Kind == "source.organizeImports" {
				found = true
			}
		}
		if !found {
			t.Errorf("no organize imports action in the menu")
		}
	})
}

func TestImportQuickFixes(t *testing.T) {
	c := start(t)
	t.Run("remove an unused import", func(t *testing.T) {
		src := "import \"std/fmt\";\nimport \"std/math\";\n\nfunc main() {\n  int n = math.max(1, 2);\n}\n"
		p := c.open(map[string]string{"main.wsp": src})
		var fix *codeAction
		for _, a := range p.codeActions() {
			if strings.HasPrefix(a.Title, "Remove unused import") {
				a := a
				fix = &a
			}
		}
		if fix == nil {
			t.Fatal("no quick fix to remove the import")
		}
		want := "import \"std/math\";\n\nfunc main() {\n  int n = math.max(1, 2);\n}\n"
		if got := fix.apply(p, src); got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
		if !strings.Contains(fix.Title, "fmt") {
			t.Errorf("the title does not name the import: %s", fix.Title)
		}
	})
	t.Run("add a missing import", func(t *testing.T) {
		src := "func main() {\n  int n = math.max(1, 2);\n}\n"
		p := c.open(map[string]string{"main.wsp": src})
		found := false
		for _, a := range p.codeActions() {
			if a.Title == `Add import "std/math"` {
				found = true
			}
		}
		if !found {
			t.Errorf("no quick fix to add the import")
		}
	})
}

func TestAutoImportCompletion(t *testing.T) {
	c := start(t)
	t.Run("a module name brings its import", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "func main() {\n  int n = mat¦;\n}\n"})
		for _, it := range p.completionItems() {
			if it.Label == "math" {
				if len(it.AdditionalTextEdits) != 1 || !strings.Contains(it.AdditionalTextEdits[0].NewText, `import "std/math";`) {
					t.Errorf("the item does not carry the import: %+v", it.AdditionalTextEdits)
				}
				return
			}
		}
		t.Error("math is not offered")
	})
	t.Run("a member of a module brings its import", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "func main() {\n  int n = max¦;\n}\n"})
		for _, it := range p.completionItems() {
			if it.Label == "math.max" {
				if len(it.AdditionalTextEdits) != 1 || !strings.Contains(it.AdditionalTextEdits[0].NewText, `import "std/math";`) {
					t.Errorf("the item does not carry the import: %+v", it.AdditionalTextEdits)
				}
				return
			}
		}
		t.Error("math.max is not offered")
	})
	t.Run("an imported module is not offered again", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "import \"std/math\";\n\nfunc main() {\n  int n = mat¦;\n}\n"})
		for _, it := range p.completionItems() {
			if it.Label == "math" && len(it.AdditionalTextEdits) > 0 {
				t.Errorf("an import was offered for a module that is already imported")
			}
		}
	})
}

func TestSemanticTokensGenerics(t *testing.T) {
	c := start(t)
	src := "struct Pair<A, B> {\n\tA first;\n\tB second;\n}\n\nfunc swap<A, B>(Pair<A, B> p) Pair<B, A> {\n\treturn Pair<B, A>{first: p.second, second: p.first};\n}\n\nfunc main() {\n\tPair<int, string> p = Pair<int, string>{first: 1, second: \"a\"};\n\tPair<string, int> q = swap(p);\n}\n"
	p := c.open(map[string]string{"main.wsp": src})
	got := map[string]string{}
	for _, tok := range p.semanticTokens() {
		got[strconv.Itoa(tok.line)+":"+strconv.Itoa(tok.col)] = tok.kind
		if tok.kind == "type" && tok.length != 4 {
			t.Errorf("type token at %d:%d has length %d, want the name only (4)", tok.line, tok.col, tok.length)
		}
	}
	want := map[string]string{
		"0:12": "typeParameter", "0:15": "typeParameter",
		"1:1": "typeParameter", "2:1": "typeParameter",
		"5:10": "typeParameter", "5:13": "typeParameter",
		"5:16": "type", "5:21": "typeParameter", "5:24": "typeParameter",
		"5:30": "type", "5:35": "typeParameter", "5:38": "typeParameter",
		"6:8": "type", "6:13": "typeParameter", "6:16": "typeParameter",
		"10:1": "type", "10:23": "type",
	}
	for at, kind := range want {
		if got[at] != kind {
			t.Errorf("token at %s: want %s, got %q", at, kind, got[at])
		}
	}
}

const genericShapes = `struct Pair<A, B> {
	A first;
	B second;
}

func (*Pair<A, B> p) swapped() Pair<B, A> {
	return Pair<B, A>{first: p.second, second: p.first};
}

func firstOf<T>(Pair<T, int> p) T {
	return p.first;
}

func spare<T>(T v) T {
	T copy = v;
	return copy;
}

func main() {
	Pair<int, string> p = Pair<int, string>{first: 1, second: "a"};
	int n = firstOf(Pair<int, int>{first: 1, second: 2});
}
`

func TestGenericsHoverAndDefinition(t *testing.T) {
	c := start(t)
	cases := []struct {
		name  string
		probe string
		hover string
		line  int
	}{
		{"call shows the generic name and the type arguments", "first¦Of(Pair<int, int>", "func firstOf<int>(Pair<int, int> p) int", 9},
		{"type parameter in the signature", "func firstOf<¦T>", "type parameter T", 9},
		{"type parameter used in a parameter", "firstOf<T>(Pair<¦T, int> p)", "type parameter T", 9},
		{"parameter keeps the declared type", "Pair<T, int> ¦p) T", "Pair<T, int> p", -1},
		{"field in a generic body, with an instance", "return p.fi¦rst;\n}\n\nfunc spare", "Field of struct `Pair<int, int>`", 1},
		{"field of an instance", "int n = firstOf(Pair<int, int>{fi¦rst: 1", "Field of struct `Pair<int, int>`", 1},
		{"body with no instance", "return ¦copy;", "T copy", 14},
		{"receiver type parameter", "func (*Pair<¦A, B> p)", "type parameter A", 5},
	}
	for _, tc := range cases {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(genericShapes, plain, tc.probe, 1)})
		if got := p.hover(); !strings.Contains(got, tc.hover) {
			t.Errorf("%s: hover %q does not contain %q", tc.name, got, tc.hover)
		}
		if tc.line >= 0 {
			defs := p.definition()
			if len(defs) == 0 || int(defs[0].Range.Start.Line) != tc.line {
				t.Errorf("%s: definition %v, want line %d", tc.name, defs, tc.line)
			}
		}
	}
}

func TestGenericsPhantomDiagnostics(t *testing.T) {
	c := start(t)
	src := `struct Box<T> {
	T value;
}

func (*Box<T> b) get() T {
	return b.missing;
}

func broken<T>(T v) T {
	return unknownName;
}

func fine<T>(Box<T> b) T {
	T copy = b.value;
	return copy;
}

func describe<T>(T v) string {
	switch type v {
		case int i => return "int";
		default => return "other";
	}
}

func main() {
}
`
	p := c.open(map[string]string{"main.wsp": src})
	var errors []string
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			errors = append(errors, d.Message)
		}
	}
	want := []string{"invalid field missing in struct Box<T>", "identifier unknownName not declared in this scope"}
	if !slices.Equal(errors, want) {
		t.Errorf("errors %q, want %q", errors, want)
	}
}

func TestGenericsCompletion(t *testing.T) {
	c := start(t)
	base := "struct Box<T> { T v; }\nstruct Item { int x; }\nfunc id<T>(T v) T { return v; }\n"
	cases := []struct {
		name string
		body string
		want []string
		deny []string
	}{
		{"type arguments of a type", "func main() {\n\tBox<I¦> b;\n}\n", []string{"Item", "Box", "int"}, []string{"if", "return"}},
		{"type arguments of a call", "func main() {\n\tint n = id<¦>(3);\n}\n", []string{"Item", "int"}, []string{"if"}},
		{"second argument", "func main() {\n\tBox<Box<int>, ¦> b;\n}\n", []string{"Item", "int"}, []string{"if"}},
		{"type parameter in a generic body", "func first<T>(Box<T> b) T {\n\tBox<¦> c;\n\treturn b.v;\n}\nfunc main() {}\n", []string{"T", "Item"}, nil},
		{"comparison is not a type context", "func main() {\n\tint n = 3 <¦ 4;\n}\n", []string{"if"}, nil},
		{"declaring a parameter offers the types of a constraint", "struct Pair<¦> { int a; }\nfunc main() {}\n", []string{"Item", "int"}, []string{"if"}},
		{"naming a parameter offers nothing", "struct Pair<Item ¦> { int a; }\nfunc main() {}\n", nil, []string{"Item", "int", "if"}},
	}
	for _, tc := range cases {
		p := c.open(map[string]string{"main.wsp": base + tc.body})
		got := p.completion()
		for _, w := range tc.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: %q missing from the completion", tc.name, w)
			}
		}
		for _, d := range tc.deny {
			if slices.Contains(got, d) {
				t.Errorf("%s: %q should not be offered", tc.name, d)
			}
		}
	}
}

func TestGenericsRename(t *testing.T) {
	c := start(t)
	src := "struct Box<T> {\n\tT value;\n}\n\nfunc (*Box<T> b) get() T {\n\treturn b.value;\n}\n\nfunc wrap<T, U>(T a, Box<U> b) Box<T> {\n\tT copy = a;\n\treturn Box<T>{value: copy};\n}\n\nfunc main() {}\n"
	p := c.open(map[string]string{"main.wsp": strings.Replace(src, "T copy = a;", "¦T copy = a;", 1)})
	got := p.renamed("Item")
	want := "func wrap<Item, U>(Item a, Box<U> b) Box<Item> {\n\tItem copy = a;\n\treturn Box<Item>{value: copy};\n}"
	if !strings.Contains(got, want) {
		t.Errorf("function parameter:\n%s", got)
	}
	if !strings.Contains(got, "struct Box<T> {\n\tT value;") {
		t.Errorf("another declaration was renamed:\n%s", got)
	}

	p = c.open(map[string]string{"main.wsp": strings.Replace(src, "struct Box<T>", "struct Box<¦T>", 1)})
	got = p.renamed("E")
	if !strings.Contains(got, "struct Box<E> {\n\tE value;\n}") || !strings.Contains(got, "func (*Box<T> b) get() T {") {
		t.Errorf("struct parameter:\n%s", got)
	}

	p = c.open(map[string]string{"main.wsp": strings.Replace(src, "func (*Box<T> b) get() T {", "func (*Box<T> b) get() ¦T {", 1)})
	got = p.renamed("V")
	if !strings.Contains(got, "func (*Box<V> b) get() V {") || !strings.Contains(got, "struct Box<T> {") {
		t.Errorf("receiver parameter:\n%s", got)
	}
}

func TestGenericsDeclaredTypesInHover(t *testing.T) {
	c := start(t)
	src := `struct Ring<T, const int N> {
	[N]T items;
	int head;
}

func (*Ring<T, N> r) push(T v) {
	r.head = (r.head + 1) % N;
}

func twice<T>(T v) T {
	T copy = v;
	T again = copy;
	return again;
}

func main() {
	int n = twice(3);
	Ring<int, 4> r;
	r.push(1);
}
`
	cases := []struct{ probe, want string }{
		{"T co¦py = v;", "T copy"},
		{"T again = ¦copy;", "T copy"},
		{"return ¦again;", "T again"},
		{"% ¦N;", "const int N"},
		{"func (*Ring<T, N> r) push(¦T v)", "type parameter T"},
	}
	for _, tc := range cases {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(src, plain, tc.probe, 1)})
		if got := p.hover(); !strings.Contains(got, tc.want) {
			t.Errorf("%s: hover %q does not contain %q", tc.probe, got, tc.want)
		}
	}
}

func TestGenericTypeDeclarationsInLSP(t *testing.T) {
	c := start(t)
	src := "enum Maybe<T> {\n\tNothing,\n\tJust { T value; }\n}\n\ntype Op<T> (T, T) => T;\n\nfunc (*Maybe<T> m) get(T fallback) T {\n\treturn fallback;\n}\n\nfunc add(int a, int b) int {\n\treturn a + b;\n}\n\nfunc main() {\n\tMaybe<int> a = Maybe<int>.Just{value: 1};\n\tOp<int> f = add;\n}\n"
	cases := []struct{ probe, want string }{
		{"Just { ¦T value; }", "type parameter T"},
		{"type Op<T> (¦T, T) => T;", "type parameter T"},
		{"func (*Maybe<¦T> m)", "type parameter T"},
		{"get(T fallback) ¦T {", "type parameter T"},
		{"Maybe<int> a = Maybe<int>.J¦ust{value: 1};", "Just"},
	}
	for _, tc := range cases {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(src, plain, tc.probe, 1)})
		if got := p.hover(); !strings.Contains(got, tc.want) {
			t.Errorf("%s: hover %q does not contain %q", tc.probe, got, tc.want)
		}
	}
	p := c.open(map[string]string{"main.wsp": src})
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("unexpected error: %s", d.Message)
		}
	}
}

func TestGenericDeclarationNamesHover(t *testing.T) {
	c := start(t)
	src := `struct Box<T> {
	T value;
}

enum Maybe<T> {
	Nothing,
	Just { T value; }
}

type Op<T> (T, T) => T;

func (*Box<T> b) get() T {
	return b.value;
}

func id<T, const int N>(T v, ...rest) (T, T) {
	return v, v;
}

func main() {
}
`
	cases := []struct{ probe, want string }{
		{"struct B¦ox<T>", "struct Box<T> {\n  T value\n}"},
		{"enum Ma¦ybe<T>", "enum Maybe<T> {\n  Nothing\n  Just { T value }\n}"},
		{"type O¦p<T>", "type Op<T> (T, T) => T"},
		{"func i¦d<T", "func id<T, const int N>(T v, ...rest) (T, T)"},
		{"func (*Box<T> b) g¦et()", "func (*Box<T> b) get() T"},
	}
	for _, tc := range cases {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(src, plain, tc.probe, 1)})
		if got := p.hover(); !strings.Contains(got, tc.want) {
			t.Errorf("%s: hover %q does not contain %q", tc.probe, got, tc.want)
		}
	}
}

func TestGenericTypeArgumentsHoverAndLiteralDefinition(t *testing.T) {
	c := start(t)
	src := "struct Box<T> {\n\tT value;\n}\n\nstruct Plain {\n\tint value;\n}\n\nenum Maybe<T> {\n\tNothing,\n\tJust { T value; }\n}\n\nfunc main() {\n\tBox<Maybe<int>> nested;\n\tBox<int> b = Box<int>{value: 1};\n\tPlain p = Plain{value: 2};\n}\n"
	hovers := []struct{ probe, want string }{
		{"\tBox<May¦be<int>> nested", "enum Maybe<T>"},
		{"\tBox<Maybe<i¦nt>> nested", "int"},
		{"\tBo¦x<Maybe<int>> nested", "struct Box<T>"},
	}
	for _, tc := range hovers {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(src, plain, tc.probe, 1)})
		if got := p.hover(); !strings.Contains(got, tc.want) || strings.Contains(got, "struct Box<T>") != strings.Contains(tc.want, "struct Box<T>") {
			t.Errorf("%s: hover %q, want %q", tc.probe, got, tc.want)
		}
	}
	definitions := []struct {
		probe string
		line  int
	}{
		{"= Bo¦x<int>{value: 1}", 0},
		{"= Pla¦in{value: 2}", 4},
	}
	for _, tc := range definitions {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(src, plain, tc.probe, 1)})
		defs := p.definition()
		if len(defs) == 0 || int(defs[0].Range.Start.Line) != tc.line {
			t.Errorf("%s: definition %v, want line %d", tc.probe, defs, tc.line)
		}
	}
}

func TestCompletionOfCapturedVariable(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src string }{
		{"if with a half typed name", "func get() ?int {\n\treturn 1;\n}\n\nfunc main() {\n\tif get() |v| {\n\t\tv¦\n\t}\n}\n"},
		{"loop with a half typed name", "func get() ?int {\n\treturn 1;\n}\n\nfunc main() {\n\tloop get() |v| {\n\t\tv¦\n\t}\n}\n"},
		{"loop on an empty line", "func get() ?int {\n\treturn 1;\n}\n\nfunc main() {\n\tloop get() |v| {\n\t\t¦\n\t}\n}\n"},
		{"after other statements", "func get() ?int {\n\treturn 1;\n}\n\nfunc main() {\n\tif get() |v| {\n\t\tint x = v;\n\t\tv¦\n\t}\n}\n"},
	}
	for round := 0; round < 20; round++ {
		for _, tc := range cases {
			p := c.open(map[string]string{"main.wsp": tc.src})
			if got := p.completion(); !slices.Contains(got, "v") {
				t.Errorf("%s (round %d): v is not offered", tc.name, round)
				break
			}
		}
	}
}

func TestInterfacesInLSP(t *testing.T) {
	c := start(t)
	good := "interface Writer {\n\tfunc write([]byte data) !int;\n}\n\ninterface Number { int | float }\n\nstruct Buffer implements Writer {\n\tint length;\n}\n\nfunc (*Buffer b) write([]byte data) !int {\n\treturn len(data);\n}\n\nfunc sum<Number T>([]T items) T {\n\tT total;\n\tfor i in 0..len(items) - 1 {\n\t\ttotal += items[i];\n\t}\n\treturn total;\n}\n\nfunc send<Writer W>(W w, []byte data) !int {\n\treturn w.write(data)!;\n}\n\nfunc main() {\n\tBuffer b;\n\t[]byte data;\n\tint n = send(&b, data) ?? 0;\n\t[]int xs;\n\tint total = sum(xs);\n}\n"
	p := c.open(map[string]string{"main.wsp": good})
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("unexpected error: %s", d.Message)
		}
	}
	if text, changed := p.formatted(); changed {
		t.Errorf("formatting changed an already formatted file:\n%s", text)
	}

	bad := "interface Writer { func write(int n) int; }\ninterface Number { int | float }\nstruct Plain { int x; }\nfunc (*Plain p) write(int n) int { return n; }\nstruct Wrong implements Writer { int x; }\nfunc send<Writer W>(W w) {}\nfunc add<Number T>(T a, T b) T { return a + b; }\nfunc main() {\n\tPlain p;\n\tsend(&p);\n\tadd(\"a\", \"b\");\n}\n"
	p = c.open(map[string]string{"main.wsp": bad})
	var messages []string
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			messages = append(messages, d.Message)
		}
	}
	for _, want := range []string{"Wrong declares implements Writer, but it does not", "does not satisfy Writer", "string does not satisfy Number"} {
		if !slices.ContainsFunc(messages, func(m string) bool { return strings.Contains(m, want) }) {
			t.Errorf("no error with %q in %q", want, messages)
		}
	}
}

func TestInterfaceVisibilityAcrossModules(t *testing.T) {
	c := start(t)
	shapes := "public interface Shape {\n\tfunc area() int;\n}\n"
	main := "import \"shapes\";\n\npublic struct Open implements shapes.Shape {\n\tint side;\n}\n\nfunc (*Open o) area() int {\n\treturn o.side;\n}\n\nstruct Closed implements shapes.Shape {\n\tint side;\n}\n\nfunc (*Closed o) area() int {\n\treturn o.side;\n}\n\nfunc main() {\n}\n"
	p := c.open(map[string]string{"main.wsp": main, "shapes.wsp": shapes})
	var messages []string
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			messages = append(messages, d.Message)
		}
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "Open declares implements Shape") {
		t.Errorf("errors %q, want only the one about the public Open with a private method", messages)
	}
}

func TestInterfaceValuesInLSP(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src, want string }{
		{"a value instead of a pointer", "interface Writer { func write(int n) int; }\nstruct B implements Writer { int t; }\nfunc (*B b) write(int n) int { return n; }\nfunc main() {\n  B b;\n  Writer w = b;\n}\n", "an interface holds a pointer to one: write &value"},
		{"a pointer to a type that does not implement it", "interface Writer { func write(int n) int; }\nstruct P { int x; }\nfunc main() {\n  P p;\n  Writer w = &p;\n}\n", "cannot be used as Writer"},
		{"a pointer to a type that has the methods but does not declare", "interface Writer { func write(int n) int; }\nstruct P { int x; }\nfunc (*P p) write(int n) int { return n; }\nfunc main() {\n  P p;\n  Writer w = &p;\n}\n", "add `implements Writer`"},
		{"an interface for another interface", "interface Writer { func write(int n) int; }\ninterface Closer { func close(); }\nstruct B implements Writer, Closer { int t; }\nfunc (*B b) write(int n) int { return n; }\nfunc (*B b) close() {}\nfunc main() {\n  B b;\n  Writer w = &b;\n  Closer c = w;\n}\n", "cannot become a Closer yet"},
		{"a method that does not exist", "interface Writer { func write(int n) int; }\nfunc use(Writer w) {\n  w.missing();\n}\nfunc main() {}\n", "interface Writer has no method missing"},
		{"a field of an interface", "interface Writer { func write(int n) int; }\nfunc use(Writer w) int {\n  return w.total;\n}\nfunc main() {}\n", "has methods and no fields"},
		{"comparing interface values", "interface Writer { func write(int n) int; }\nfunc same(Writer a, Writer b) bool {\n  return a == b;\n}\nfunc main() {}\n", "is not defined for Writer"},
		{"a case that can never match", "interface Writer { func write(int n) int; }\nstruct P { int x; }\nfunc use(Writer w) int {\n  switch type w {\n    case *P p => return p.x;\n    default => return 0;\n  }\n}\nfunc main() {}\n", "can never be what Writer holds"},
		{"a case that is not a pointer", "interface Writer { func write(int n) int; }\nstruct B implements Writer { int t; }\nfunc (*B b) write(int n) int { return n; }\nfunc use(Writer w) int {\n  switch type w {\n    case B b => return b.t;\n    default => return 0;\n  }\n}\nfunc main() {}\n", "names a pointer to a type"},
	}
	for _, tc := range cases {
		p := c.open(map[string]string{"main.wsp": tc.src})
		var messages []string
		for _, d := range p.diagnostics() {
			if d.Severity == 1 {
				messages = append(messages, d.Message)
			}
		}
		if !slices.ContainsFunc(messages, func(m string) bool { return strings.Contains(m, tc.want) }) {
			t.Errorf("%s: errors %q, want one with %q", tc.name, messages, tc.want)
		}
	}

	good := "interface Writer { func write(int n) int; }\ninterface Closer { func close(); }\ninterface WriteCloser { Writer & Closer }\nstruct B implements WriteCloser { int total; }\nfunc (*B b) write(int n) int { b.total += n; return b.total; }\nfunc (*B b) close() { b.total = 0; }\nstruct Log { Writer out; ?Writer backup; }\nfunc emit(Writer w, int n) int { return w.write(n); }\nfunc pick(bool first, Writer a, Writer b) Writer { return a if first; return b; }\nfunc total(Writer w) int {\n  switch type w {\n    case *B b => return b.total;\n    default => return -1;\n  }\n}\nfunc main() {\n  B b;\n  Writer w = &b;\n  WriteCloser wc = &b;\n  wc.close();\n  Log log = Log{out: &b, backup: &b};\n  int r = w.write(1) + emit(&b, 2) + log.out.write(3);\n  Writer chosen = pick(true, &b, &b);\n  int t = total(chosen);\n  [2]Writer pair = .{&b, &b};\n  if log.backup |x| {\n    x.write(1);\n  }\n}\n"
	p := c.open(map[string]string{"main.wsp": good})
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("unexpected error in the valid program: %s", d.Message)
		}
	}
}

const interfaceProject = `/// Writes bytes somewhere.
interface Writer {
	func write(int n) int;
	func flush();
}

interface Number { int | float }

interface Closer { func close(); }

interface WriteCloser { Writer & Closer }

interface Source<T> { func next() T; }

struct Buffer implements Writer, Closer {
	int total;
}

func (*Buffer b) write(int n) int { return n; }
func (*Buffer b) flush() {}
func (*Buffer b) close() {}

struct Files implements Writer {
	int open;
}

func (Files f) write(int n) int { return n + f.open; }
func (Files f) flush() {}

func sum<Number T>(T a) T { return a; }

func send<Writer W>(W w) int { return w.write(1); }

func use(Writer w) int {
	return w.write(2);
}

func main() {
	Buffer b;
	Writer w = &b;
	w.flush();
}
`

func TestInterfaceHoverAndDefinition(t *testing.T) {
	c := start(t)
	cases := []struct {
		probe string
		hover string
		line  int
	}{
		{"interface Wri¦ter {", "interface Writer {\n  func write(int n) int\n  func flush()\n}", -1},
		{"func wri¦te(int n) int;", "Method of the interface `Writer`", -1},
		{"struct Buffer implements Wri¦ter,", "interface Writer {", 1},
		{"func sum<Num¦ber T>", "interface Number { int | float }", 6},
		{"func use(Wri¦ter w)", "interface Writer {", 1},
		{"return w.wri¦te(2);", "func (Writer) write(int n) int", 2},
		{"w.flu¦sh();", "func (Writer) flush()", 3},
		{"interface WriteCloser { Wri¦ter & Closer }", "interface Writer {", 1},
		{"interface WriteCloser { Writer & Clo¦ser }", "interface Closer {\n  func close()\n}", 8},
		{"func send<Wri¦ter W>", "interface Writer {", 1},
	}
	for _, tc := range cases {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(interfaceProject, plain, tc.probe, 1)})
		if got := p.hover(); !strings.Contains(got, tc.hover) {
			t.Errorf("%s: hover %q does not contain %q", tc.probe, got, tc.hover)
		}
		if tc.line >= 0 {
			defs := p.definition()
			if len(defs) == 0 || int(defs[0].Range.Start.Line) != tc.line {
				t.Errorf("%s: definition %v, want line %d", tc.probe, defs, tc.line)
			}
		}
	}

	p := c.open(map[string]string{"main.wsp": interfaceProject})
	if got := p.hover(); got != "" {
		t.Errorf("hover without a cursor: %q", got)
	}
}

func TestInterfaceImplementations(t *testing.T) {
	c := start(t)
	cases := []struct {
		probe string
		lines []int
	}{
		{"interface Wri¦ter {", []int{14, 22}},
		{"func use(Wri¦ter w)", []int{14, 22}},
		{"func wri¦te(int n) int;", []int{18, 26}},
		{"return w.wri¦te(2);", []int{18, 26}},
		{"w.flu¦sh();", []int{19, 27}},
		{"interface Closer { func clo¦se(); }", []int{20}},
	}
	for _, tc := range cases {
		plain := strings.Replace(tc.probe, "¦", "", 1)
		p := c.open(map[string]string{"main.wsp": strings.Replace(interfaceProject, plain, tc.probe, 1)})
		var got []int
		for _, loc := range p.implementations() {
			got = append(got, int(loc.Range.Start.Line))
		}
		slices.Sort(got)
		if !slices.Equal(got, tc.lines) {
			t.Errorf("%s: implementations at lines %v, want %v", tc.probe, got, tc.lines)
		}
	}
}

func TestInterfaceOutline(t *testing.T) {
	c := start(t)
	p := c.open(map[string]string{"main.wsp": interfaceProject})
	symbols := p.symbols()
	if !slices.Contains(symbols, "Writer:11[write,flush]") || !slices.Contains(symbols, "Number:11[]") {
		t.Errorf("outline %q lacks the interfaces and their methods", symbols)
	}
}

func TestInterfaceCompletion(t *testing.T) {
	c := start(t)
	cases := []struct {
		name string
		src  string
		want []string
		deny []string
	}{
		{"methods of an interface value", "interface Writer { func write(int n) int; func flush(); }\nfunc use(Writer w) {\n\tw.¦\n}\nfunc main() {}\n", []string{"write", "flush"}, []string{"close"}},
		{"methods of a composed interface", "interface Writer { func write(int n) int; }\ninterface Closer { func close(); }\ninterface Both { Writer & Closer }\nfunc use(Both b) {\n\tb.¦\n}\nfunc main() {}\n", []string{"write", "close"}, nil},
		{"interfaces after implements", "interface Writer { func write(int n) int; }\nstruct Plain { int x; }\nstruct Buffer implements ¦", []string{"Writer"}, []string{"Plain", "int", "if"}},
		{"interfaces after a comma", "interface Writer { func write(int n) int; }\ninterface Closer { func close(); }\nstruct Buffer implements Writer, ¦", []string{"Writer", "Closer"}, []string{"int"}},
		{"interfaces in a constraint", "interface Writer { func write(int n) int; }\nfunc send<W¦>(int x) {}\nfunc main() {}\n", []string{"Writer", "int"}, []string{"if"}},
		{"an interface as the type of a value", "interface Writer { func write(int n) int; }\nfunc use(Wr¦) {}\nfunc main() {}\n", []string{"Writer"}, nil},
		{"the keywords", "func main() {\n\timpl¦\n}\n", []string{"implements"}, nil},
	}
	for _, tc := range cases {
		p := c.open(map[string]string{"main.wsp": tc.src})
		got := p.completion()
		for _, w := range tc.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: %q missing from %v", tc.name, w, got[:min(len(got), 12)])
			}
		}
		for _, d := range tc.deny {
			if slices.Contains(got, d) {
				t.Errorf("%s: %q should not be offered", tc.name, d)
			}
		}
	}
}

func TestInterfaceSemanticTokens(t *testing.T) {
	c := start(t)
	src := "interface Source<T> {\n\tfunc next() T;\n}\n\nfunc main() {\n}\n"
	p := c.open(map[string]string{"main.wsp": src})
	got := map[string]string{}
	for _, tok := range p.semanticTokens() {
		got[strconv.Itoa(tok.line)+":"+strconv.Itoa(tok.col)] = tok.kind
	}
	for at, kind := range map[string]string{"0:17": "typeParameter", "1:13": "typeParameter"} {
		if got[at] != kind {
			t.Errorf("token at %s: want %s, got %q (%v)", at, kind, got[at], got)
		}
	}
}

func TestQuickFixDeclaresImplements(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src, want string }{
		{"a struct with no implements", "interface Writer { func write(int n) int; }\nstruct Plain {\n\tint x;\n}\nfunc (*Plain p) write(int n) int { return n; }\nfunc send<Writer W>(W w) {}\nfunc main() {\n\tPlain p;\n\tsend(&p);\n}\n", "struct Plain implements Writer {"},
		{"a struct that already implements another", "interface Writer { func write(int n) int; }\ninterface Closer { func close(); }\nstruct Plain implements Closer {\n\tint x;\n}\nfunc (*Plain p) write(int n) int { return n; }\nfunc (*Plain p) close() {}\nfunc send<Writer W>(W w) {}\nfunc main() {\n\tPlain p;\n\tsend(&p);\n}\n", "struct Plain implements Closer, Writer {"},
		{"a type", "interface Writer { func write(int n) int; }\ntype Meters int;\nfunc (Meters m) write(int n) int { return n; }\nfunc use(Writer w) {}\nfunc main() {\n\tMeters m = 3;\n\tuse(&m);\n}\n", "type Meters int implements Writer;"},
	}
	for _, tc := range cases {
		p := c.open(map[string]string{"main.wsp": tc.src})
		actions := p.codeActions()
		var fixed string
		for _, a := range actions {
			if strings.HasPrefix(a.Title, "Declare that") {
				fixed = a.apply(p, tc.src)
			}
		}
		if !strings.Contains(fixed, tc.want) {
			t.Errorf("%s: the fixed source lacks %q:\n%s", tc.name, tc.want, fixed)
			continue
		}
		again := c.open(map[string]string{"main.wsp": fixed})
		for _, d := range again.diagnostics() {
			if d.Severity == 1 {
				t.Errorf("%s: still an error after the fix: %s", tc.name, d.Message)
			}
		}
	}
}

func TestOnlyTripleSlashIsDocumentation(t *testing.T) {
	c := start(t)
	src := "/// The documented one.\nfunc docu¦mented() {}\n\n// Only a comment.\nfunc plain() {}\n\nfunc main() {\n\tdocumented();\n\tplain();\n}\n"
	p := c.open(map[string]string{"main.wsp": src})
	if got := p.hover(); !strings.Contains(got, "The documented one.") || strings.Contains(got, "/ The") {
		t.Errorf("hover %q lacks the doc comment", got)
	}
	q := c.open(map[string]string{"main.wsp": strings.Replace(strings.Replace(src, "docu¦mented", "documented", 1), "func plain", "func pl¦ain", 1)})
	if got := q.hover(); strings.Contains(got, "Only a comment") {
		t.Errorf("hover %q shows a plain comment as documentation", got)
	}
}

func TestCompletionInTypedContexts(t *testing.T) {
	c := start(t)
	head := "enum Token { Number, Plus, Minus }\ninterface Node { func node(); }\nstruct Lit implements Node { float value; }\nfunc (*Lit l) node() {}\nenum AST {\n\tUndefined,\n\tLiteral{float value;}\n}\n"
	tokens := []string{"Number", "Plus", "Minus"}
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"assignment", "func main() {\n\tToken t;\n\tt = .¦\n}\n", tokens},
		{"yield", "func f() Token {\n\tToken t = {\n\t\tyield .¦;\n\t};\n\treturn t;\n}\n", tokens},
		{"yield inside an if", "func f(bool b) Token {\n\tToken t = {\n\t\tif b {\n\t\t\tyield .¦;\n\t\t}\n\t\tyield .Plus;\n\t};\n\treturn t;\n}\n", tokens},
		{"yield of an enum type", "func f() Token {\n\tToken t = {\n\t\tyield Token.¦;\n\t};\n\treturn t;\n}\n", tokens},
		{"optional return switch arm", "func f(Token k) ?Token {\n\treturn switch k {\n\t\tcase .Number => .¦;\n\t\tdefault => null;\n\t};\n}\n", tokens},
		{"variant fields in an arm", "func f(Token k) ?AST {\n\treturn switch k {\n\t\tcase .Number => .Literal{¦};\n\t\tdefault => null;\n\t};\n}\n", []string{"value"}},
		{"member after a line without a semicolon", "func main() {\n\tLit a = .{value: 1.0}\n\ta.¦\n}\n", []string{"value", "node"}},
		{"binding of a type switch arm", "func f(Node n) {\n\tswitch type n {\n\t\tcase *Lit l => l.¦\n\t\tdefault => {}\n\t}\n}\n", []string{"value", "node"}},
		{"binding of a type switch block arm", "func f(Node n) {\n\tswitch type n {\n\t\tcase *Lit l => {\n\t\t\tl.¦\n\t\t}\n\t\tdefault => {}\n\t}\n}\n", []string{"value", "node"}},
		{"binding of a variant arm", "func f(AST a) {\n\tswitch a {\n\t\tcase .Literal l => l.¦\n\t\tdefault => {}\n\t}\n}\n", []string{"value"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": head + tc.body}).completion()
			for _, want := range tc.want {
				if !slices.Contains(got, want) {
					t.Errorf("completion lacks %q: %v", want, got)
				}
			}
		})
	}
}

func TestRangeInLSP(t *testing.T) {
	c := start(t)

	members := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0..9;\n\tint n = r.¦\n}\n"}).completion()
	for _, want := range []string{"start", "end", "contains", "count", "iter", "step"} {
		if !slices.Contains(members, want) {
			t.Errorf("a range lacks %q: %v", want, members)
		}
	}
	for _, hidden := range []string{"by", "stride", "makeRange"} {
		if slices.Contains(members, hidden) {
			t.Errorf("a range offers the internal %q: %v", hidden, members)
		}
	}

	names := c.open(map[string]string{"main.wsp": "func main() {\n\tRan¦\n}\n"}).completion()
	if !slices.Contains(names, "Range") {
		t.Errorf("Range is not offered: %v", names)
	}

	hover := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0..9;\n\tint n = r.cou¦nt();\n}\n"}).hover()
	if !strings.Contains(hover, "count() int") || !strings.Contains(hover, "How many values") {
		t.Errorf("hover of count: %q", hover)
	}

	dots := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0.¦.9;\n}\n"}).hover()
	if !strings.Contains(dots, "Range<int>") || strings.Contains(dots, "__wisp") {
		t.Errorf("hover of ..: %q", dots)
	}

	typed := c.open(map[string]string{"main.wsp": "func main() {\n\tRan¦ge<int> r = 0..9;\n}\n"})
	if hover := typed.hover(); !strings.Contains(hover, "struct Range<T>") {
		t.Errorf("hover of Range: %q", hover)
	}
	if defs := typed.definition(); len(defs) != 0 {
		t.Errorf("Range has no file to open: %v", defs)
	}

	method := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0..9;\n\tint n = r.cou¦nt();\n}\n"})
	if defs := method.definition(); len(defs) != 0 {
		t.Errorf("a builtin method has no file to open: %v", defs)
	}
}

func TestStdModulesOpenedAsFiles(t *testing.T) {
	c := start(t)
	for _, name := range []string{"mem/mem.wsp", "mem/debug.wsp", "mem/arena.wsp"} {
		path := filepath.Join(repoRoot(), "std", filepath.FromSlash(name))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p := &project{c: c, dir: filepath.Dir(path), path: path, text: string(data)}
		c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": "file://" + path, "languageId": "wisp", "version": 1, "text": string(data)}})
		for _, d := range p.diagnostics() {
			if d.Severity == 1 {
				t.Errorf("%s: %s", name, d.Message)
			}
		}
	}
}

func TestSemanticTokensOfGenericMethodsHaveTheLengthOfTheName(t *testing.T) {
	c := start(t)
	src := "struct List<T> { T x; }\nfunc (*List<T> l) append(T v) ! {}\nfunc (*List<T> l) insert(int i, T v) ! {}\n"
	for round := 0; round < 10; round++ {
		for _, tk := range c.open(map[string]string{"main.wsp": src}).semanticTokens() {
			if tk.kind == "type" && tk.length != len("List") {
				t.Fatalf("a type token of length %d at line %d: %+v", tk.length, tk.line, tk)
			}
		}
	}
}

func TestCompletionInsideGenericInlineMethods(t *testing.T) {
	c := start(t)
	head := "struct List<T> {\n\tinternal []T buf;\n\tinternal int size;\n}\n\n"
	cases := []struct{ name, src string }{
		{"a return", head + "inline func (*List<T> l) first() int {\n\treturn l¦\n}\n"},
		{"a declaration", head + "inline func (*List<T> l) first() int {\n\tint n = l¦\n\treturn n;\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": tc.src}).completion()
			if !slices.Contains(got, "l") {
				t.Errorf("the receiver is not offered: %v", got)
			}
		})
	}
}

func TestGenericFunctionsHelpWhileTyping(t *testing.T) {
	c := start(t)
	head := "import \"std/mem\";\n\nfunc twice<T>(T v, int n) T { return v; }\n\nfunc main() ! {\n\t[]int xs = mem.make<int>(mem.heap, 4)!;\n"
	cases := []struct {
		name, src, label string
		active           int
	}{
		{"a module function, first argument", head + "\txs = mem.resize(¦\n}\n", "resize<T, Allocator A>(A a, []T items, int n) ![]T", 0},
		{"a module function, third argument", head + "\txs = mem.resize(mem.heap, xs, ¦\n}\n", "resize<T, Allocator A>(A a, []T items, int n) ![]T", 2},
		{"a function of the module", head + "\tint k = twice(3, ¦\n}\n", "twice<T>(T v, int n) T", 1},
		{"with the types written", head + "\tint k = twice<int>(3, ¦\n}\n", "twice<T>(T v, int n) T", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			label, active := c.open(map[string]string{"main.wsp": tc.src}).signature()
			if label != tc.label || active != tc.active {
				t.Errorf("signature %q (parameter %d), want %q (parameter %d)", label, active, tc.label, tc.active)
			}
		})
	}
}

func TestHoverOfACallThatIsStillBeingTyped(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src, want string }{
		{"a generic of a module", "import \"std/mem\";\n\nfunc main() ! {\n\t[]int xs = mem.make<int>(mem.heap, 4)!;\n\txs = mem.resi¦ze(mem.heap, \n}\n", "resize<T, Allocator A>(A a, []T items, int n) ![]T"},
		{"a generic of the module", "func twice<T>(T v, int n) T { return v; }\n\nfunc main() {\n\tint k = twi¦ce(3, \n}\n", "func twice<T>(T v, int n) T"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": tc.src}).hover()
			if !strings.Contains(got, tc.want) {
				t.Errorf("hover %q lacks %q", got, tc.want)
			}
		})
	}
}

func TestMembersAfterACallInAChain(t *testing.T) {
	c := start(t)
	head := "import \"std/mem\";\n\npublic struct List<T> {\n\t?mem.Allocator parent;\n\tinternal []T buf;\n}\n\ninline func (*List<T> l) source() mem.Allocator {\n\treturn l.parent ?? mem.heap;\n}\n\nstruct Plain { ?mem.Allocator parent; }\nfunc (*Plain p) source() mem.Allocator { return p.parent ?? mem.heap; }\nfunc (*Plain p) self() *Plain { return p; }\n\n"
	cases := []struct {
		name, src string
		want, no  []string
	}{
		{"a method of a generic type", head + "func (*List<T> l) release() {\n\tl.source().¦\n}\n", []string{"alloc", "free"}, []string{"len", "parent"}},
		{"a method of a plain type", head + "func (*Plain p) release() {\n\tp.source().¦\n}\n", []string{"alloc", "free"}, []string{"len"}},
		{"a method that gives its own type", head + "func (*Plain p) release() {\n\tp.self().¦\n}\n", []string{"parent", "source", "self"}, []string{"len"}},
		{"with a half typed member", head + "func (*List<T> l) release() {\n\tl.source().fr¦\n}\n", []string{"free"}, []string{"parent"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": tc.src}).completion()
			for _, want := range tc.want {
				if !slices.Contains(got, want) {
					t.Errorf("completion lacks %q: %v", want, got)
				}
			}
			for _, bad := range tc.no {
				if slices.Contains(got, bad) {
					t.Errorf("completion offers %q: %v", bad, got)
				}
			}
		})
	}
}

func TestHoverOfAMethodCallWithoutItsSemicolon(t *testing.T) {
	c := start(t)
	src := "import \"std/mem\";\n\nstruct Plain { ?mem.Allocator parent; }\nfunc (*Plain p) source() mem.Allocator { return p.parent ?? mem.heap; }\n\nfunc (*Plain p) release() {\n\tp.sou¦rce()\n}\n"
	got := c.open(map[string]string{"main.wsp": src}).hover()
	if !strings.Contains(got, "source() Allocator") {
		t.Errorf("hover %q", got)
	}
}

func TestErrorsOfGenericCallsUseTheWrittenName(t *testing.T) {
	c := start(t)
	src := "func twice<T>(T v, int n) T { return v; }\n\nfunc main() {\n\tint k = twice(1);\n}\n"
	for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
		if strings.Contains(d.Message, "expects 2 arguments") {
			if strings.Contains(d.Message, "twice_") || !strings.Contains(d.Message, "function twice expects 2 arguments") {
				t.Errorf("the message shows an internal name: %q", d.Message)
			}
			return
		}
	}
	t.Error("no error about the arguments")
}

func TestMembersOfAnyExpressionBeforeTheDot(t *testing.T) {
	c := start(t)
	head := "struct Box { int width; int height; }\nfunc mk() !Box { return Box{width: 1, height: 2}; }\nfunc pick<T>(T v) T { return v; }\nfunc find() ?Box { return null; }\nfunc ls() [2]Box { return .{}; }\n\n"
	cases := []struct{ name, body string }{
		{"a fallible call", "func main() ! {\n\tmk()!.¦\n}\n"},
		{"a generic call", "func main() {\n\tBox b;\n\tpick(b).¦\n}\n"},
		{"a generic call with its types", "func main() {\n\tBox b;\n\tpick<Box>(b).¦\n}\n"},
		{"an index", "func main() {\n\t[2]Box xs;\n\txs[1].¦\n}\n"},
		{"an optional taken out", "func main() {\n\tfind()!!.¦\n}\n"},
		{"an index of a call", "func main() {\n\tls()[0].¦\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": head + tc.body}).completion()
			if !slices.Contains(got, "width") || !slices.Contains(got, "height") || len(got) != 2 {
				t.Errorf("completion is %v, want the two fields of Box", got)
			}
		})
	}
}
