package lsp

import (
	"encoding/json"
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
		{"label", "func main() {\n  #outer for i in 0..=3 {\n    break #ou¦ter;\n  }\n}\n", []string{"#outer", "declared on line 2"}},
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
		{"label", "func main() {\n  #outer for i in 0..=3 {\n    break #ou¦ter;\n  }\n}\n", 1},
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
	calc := "#standalone\npublic func add(int a, int b) int { return a + b; }\ninternal func secret(int n) int { return n * 7; }\n"
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
			p := c.openAs("calc.test.wsp", map[string]string{"calc.wsp": calc, "calc.test.wsp": "#standalone\n" + tc.test})
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
		{"a slice of a constant", "const [3]int T = .{1, 2, 3};\nfunc main() {\n  []int s = T[..];\n}\n", 1, "cannot slice the constant T"},
		{"a spread of the wrong element type", "func take(int ...xs) int { return len(xs); }\nfunc main() {\n  []string words = .{\"a\"};\n  int n = take(words...);\n}\n", 1, "passes a slice of string on"},
		{"a spread that is not a slice", "func take(int ...xs) int { return len(xs); }\nfunc main() {\n  int one = 1;\n  int n = take(one...);\n}\n", 1, "passes a slice on"},
		{"a spread into a function without a variadic", "func take(int x) int { return x; }\nfunc main() {\n  []int xs = .{1};\n  int n = take(xs...);\n}\n", 1, "has none"},
		{"a spread that matches", "func take(int ...xs) int { return len(xs); }\nfunc main() {\n  []int xs = .{1};\n  int n = take(0, xs...);\n}\n", 0, ""},
		{"returning a call that gives nothing", "func nothing() {}\nfunc f(int a) {\n  if a > 0 => return nothing();\n}\nfunc main() { f(1); }\n", 0, ""},
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
		{"an import used only by a function of another profile stays", "import \"std/math\";\n\nfunc<release> f() int {\n  return math.max(1, 2);\n}\n\nfunc<!release> f() int {\n  return 0;\n}\n\nfunc main() {\n  int n = f();\n  n++;\n}\n", ""},
		{"an import used only by a function of another system stays", "import \"std/os\";\n\nfunc<windows> f() bool {\n  return os.exists(\"x\");\n}\n\nfunc<!windows> f() bool {\n  return false;\n}\n\nfunc main() {\n  bool b = f();\n  b = !b;\n}\n", ""},
		{"an import used only in a type of another profile stays", "import \"std/mem\";\n\nfunc<release> f(mem.Allocator a) {}\n\nfunc<!release> f(int a) {}\n\nfunc main() {}\n", ""},
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

func TestAddImportStaysAboveDocComments(t *testing.T) {
	c := start(t)
	src := "// header\n\n/// Doc of main.\nfunc main() {\n  int n = math.max(1, 2);\n}\n"
	p := c.open(map[string]string{"main.wsp": src})
	for _, a := range p.codeActions() {
		if a.Title != `Add import "std/math"` {
			continue
		}
		want := "// header\n\nimport \"std/math\";\n\n/// Doc of main.\nfunc main() {\n  int n = math.max(1, 2);\n}\n"
		if got := a.apply(p, src); got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
		return
	}
	t.Error("no quick fix to add the import")
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

func TestAutoImportGoesBelowTheDirectivesOfTheFile(t *testing.T) {
	c := start(t)
	body := "func main() {\n  int n = mat¦;\n}\n"
	cases := []struct {
		name, head string
		line       int
	}{
		{"a file with no header", "", 0},
		{"standalone", "#standalone\n", 1},
		{"a target", "#target x64 & linux\n", 1},
		{"a target after a comment", "// header\n#target x64\n\n", 3},
		{"standalone followed by comments", "#standalone\n// Tour.\n//\n\n", 4},
		{"directives followed by an import", "#standalone\nimport \"std/fmt\";\n\n", 2},
		{"a profile import", "import<debug> \"std/os\";\n\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := c.open(map[string]string{"main.wsp": tc.head + body})
			for _, it := range p.completionItems() {
				if it.Label == "math" {
					if len(it.AdditionalTextEdits) != 1 {
						t.Fatalf("the item does not carry the import: %+v", it.AdditionalTextEdits)
					}
					if got := int(it.AdditionalTextEdits[0].Range.Start.Line); got != tc.line {
						t.Errorf("the import goes on line %d, want %d", got, tc.line)
					}
					return
				}
			}
			t.Error("math is not offered")
		})
	}
}

func TestImportsWithASelector(t *testing.T) {
	c := start(t)

	t.Run("the path of an import with a selector is completed like any other", func(t *testing.T) {
		plain := c.open(map[string]string{"main.wsp": "import \"std/ma¦\";\n"}).completion()
		selected := c.open(map[string]string{"main.wsp": "import<debug> \"std/ma¦\";\n"}).completion()
		if len(plain) == 0 || strings.Join(plain, ",") != strings.Join(selected, ",") {
			t.Errorf("plain %v, with a selector %v", plain, selected)
		}
	})

	t.Run("an import with a selector is not offered again", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "import<debug> \"std/math\";\n\nfunc main() {\n  int n = mat¦;\n}\n"})
		for _, it := range p.completionItems() {
			if it.Label == "math" && len(it.AdditionalTextEdits) > 0 {
				t.Errorf("an import was offered for a module that is already imported: %+v", it.AdditionalTextEdits)
			}
		}
	})

	t.Run("a member of a module imported with a selector brings no import", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "import<debug> \"std/os\";\n\nfunc main() {\n  exis¦;\n}\n"})
		for _, it := range p.completionItems() {
			if it.Label == "os.exists" && len(it.AdditionalTextEdits) > 0 {
				t.Errorf("an import was offered for a module that is already imported: %+v", it.AdditionalTextEdits)
			}
		}
	})

	t.Run("hovering the path shows the module", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "import<debug> \"std/ma¦th\";\n\nfunc main() {}\n"})
		if h := p.hover(); !strings.Contains(h, "std/math") && !strings.Contains(h, "module math") {
			t.Errorf("hover: %q", h)
		}
	})

	t.Run("an import of another profile is not reported as unused", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": "import \"std/math\";\n\nfunc<release> f() int {\n  return math.max(1, 2);\n}\n\nfunc<!release> f() int {\n  return 0;\n}\n\nfunc main() {\n  int n = f();\n  n++;\n}\n"})
		for _, d := range p.diagnostics() {
			if strings.Contains(d.Message, "never used") {
				t.Errorf("a warning on an import that the release build uses: %s", d.Message)
			}
		}
	})
}

func TestAutoImportOffersModulesThatDependOnTheFile(t *testing.T) {
	c := start(t)
	p := c.openAs("low/low.wsp", map[string]string{
		"low/low.wsp": "public struct File implements Rea¦\n",
		"high/high.wsp": "import \"low\";\n\npublic interface Reader { func read(); }\npublic func use(low.File f) {}\n",
	})
	for _, it := range p.completionItems() {
		if it.Label == "high.Reader" {
			return
		}
	}
	t.Errorf("an interface of a module that imports this one is not offered: %v", p.completion())
}

func TestImportsOfOtherFilesAreNotVersionsOfEachOther(t *testing.T) {
	c := start(t)
	p := c.open(map[string]string{
		"main.wsp":  "import \"std/math\";\n\nfunc main() {\n\tint n = math.abs(-1);\n}\n",
		"other.wsp": "import<debug> \"std/math\";\n\nfunc other() int {\n\treturn math.abs(-2);\n}\n",
	})
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("an import with a selector in another file broke this one: %s", d.Message)
		}
	}
}

func TestSwitchTypeSharedBodyInLSP(t *testing.T) {
	c := start(t)
	head := "struct Circle {\n\tint radius;\n\tbool visible;\n}\nstruct Square {\n\tint side;\n\tbool visible;\n}\n"
	body := func(use string) string {
		return head + "func f(...args) int {\n\tmatch type args[0] {\n\t\t*Circle, *Square s => {\n\t\t\t" + use + "\n\t\t}\n\t\t_ => return 0;\n\t}\n}\n"
	}

	t.Run("completion offers only what the types share", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			got := c.open(map[string]string{"main.wsp": body("return s.¦")}).completion()
			if !slices.Contains(got, "visible") || slices.Contains(got, "radius") || slices.Contains(got, "side") {
				t.Fatalf("completion should be only the shared field: %v", got)
			}
		}
	})
	t.Run("hover names every type of the binding", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			got := c.open(map[string]string{"main.wsp": body("return ¦s.visible ? 1 : 0;")}).hover()
			if !strings.Contains(got, "*Circle | *Square s") {
				t.Fatalf("hover = %q", got)
			}
		}
	})
	t.Run("the shared body has no errors", func(t *testing.T) {
		for _, d := range c.open(map[string]string{"main.wsp": body("return if s.visible => 1 : 0;")}).diagnostics() {
			if d.Severity == 1 {
				t.Errorf("unexpected error: %s", d.Message)
			}
		}
	})
	t.Run("a field only one type has is an error that names the type", func(t *testing.T) {
		var errs []string
		for _, d := range c.open(map[string]string{"main.wsp": body("return s.radius;")}).diagnostics() {
			if d.Severity == 1 {
				errs = append(errs, d.Message)
			}
		}
		if len(errs) == 0 || !strings.Contains(strings.Join(errs, "\n"), "Square") {
			t.Errorf("errors = %v", errs)
		}
	})
}

func TestDeferAcceptsAnErrorToPassUp(t *testing.T) {
	c := start(t)
	head := "struct Handle {\n\tint fd;\n}\n\nfunc (*Handle h) close() ! {\n\tif h.fd < 0 => return Error(\"no\");\n}\n\n"
	errorsOf := func(body string) []string {
		var out []string
		for _, d := range c.open(map[string]string{"main.wsp": head + body}).diagnostics() {
			if d.Severity == 1 {
				out = append(out, d.Message)
			}
		}
		return out
	}

	if got := errorsOf("func good(*Handle h) ! {\n\tdefer h.close()!;\n}\n"); len(got) != 0 {
		t.Errorf("a defer that passes an error up in a function that can fail: %v", got)
	}
	if got := errorsOf("func block(*Handle h) ! {\n\tdefer {\n\t\th.close()!;\n\t}\n}\n"); len(got) != 0 {
		t.Errorf("a block that passes an error up: %v", got)
	}
	if got := errorsOf("func plain(*Handle h) {\n\tdefer h.close()!;\n}\n"); len(got) != 1 || !strings.Contains(got[0], "cannot propagate outside a fallible function") {
		t.Errorf("a function that cannot fail: %v", got)
	}
	if got := errorsOf("func late(*Handle h) ! {\n\tdefer! h.close()!;\n}\n"); len(got) != 1 || !strings.Contains(got[0], "already leaving the function") {
		t.Errorf("a defer! that passes an error up: %v", got)
	}
	if got := errorsOf("func stops(*Handle h) {\n\tdefer h.close()!!;\n}\n"); len(got) != 0 {
		t.Errorf("a defer that stops the program on an error: %v", got)
	}
}

func TestACallThatCanFailIsNotComparedDirectly(t *testing.T) {
	c := start(t)
	head := "func only() ! {\n\treturn Error(\"no\");\n}\n\nfunc more() !int {\n\treturn 1;\n}\n\n"
	for _, expr := range []string{"only() == null", "only() != null", "null == only()", "more() == null"} {
		var got []string
		for _, d := range c.open(map[string]string{"main.wsp": head + "func main() {\n\tbool b = " + expr + ";\n}\n"}).diagnostics() {
			if d.Severity == 1 {
				got = append(got, d.Message)
			}
		}
		if len(got) != 1 || !strings.Contains(got[0], "cannot be compared directly") {
			t.Errorf("%s: %v", expr, got)
		}
	}
	var clean []string
	for _, d := range c.open(map[string]string{"main.wsp": head + "func main() {\n\tlet e = only();\n\tbool b = e == null;\n\tlet v, f = more();\n\tbool c = f == null && v == 1;\n}\n"}).diagnostics() {
		if d.Severity == 1 {
			clean = append(clean, d.Message)
		}
	}
	if len(clean) != 0 {
		t.Errorf("comparing the bound error: %v", clean)
	}
}

func TestYieldLeavesBlocksAndBreakLeavesLoops(t *testing.T) {
	c := start(t)
	errorsOf := func(body string) []string {
		var out []string
		for _, d := range c.open(map[string]string{"main.wsp": body}).diagnostics() {
			if d.Severity == 1 {
				out = append(out, d.Message)
			}
		}
		return out
	}

	good := "func main() {\n\tint n;\n\t#blk {\n\t\tfor i in 0..=3 {\n\t\t\tif i == 2 => yield #blk;\n\t\t\tn++;\n\t\t}\n\t}\n\tint v = #pick {\n\t\tfor i in 0..=3 {\n\t\t\tif i == 2 => yield #pick i;\n\t\t}\n\t\tyield -1;\n\t};\n\t#outer for i in 0..=3 {\n\t\t#inner {\n\t\t\tif i == 1 => break;\n\t\t}\n\t\tif i == 2 => break #outer;\n\t}\n}\n"
	if got := errorsOf(good); len(got) != 0 {
		t.Errorf("valid uses: %v", got)
	}

	cases := []struct{ name, body, want string }{
		{"break to a block", "func main() {\n\t#blk {\n\t\tbreak #blk;\n\t}\n}\n", "labels a block, and a block is left with yield"},
		{"break to a block that gives a value", "func main() {\n\tint v = #pick {\n\t\tbreak #pick;\n\t\tyield 1;\n\t};\n}\n", "a block is left with yield"},
		{"break with a value", "func main() {\n\tint v = #pick {\n\t\tbreak #pick 1;\n\t};\n}\n", "break does not carry a value: write yield #pick value;"},
		{"yield to a loop", "func main() {\n\t#lp for i in 0..=3 {\n\t\tyield #lp;\n\t}\n}\n", "labels a loop, and a loop is left with break"},
		{"yield without anything", "func main() {\n\tyield;\n}\n", "yield needs a value, or a label to leave a block"},
		{"a value into a block that gives none", "func main() {\n\t#blk {\n\t\tyield #blk 1;\n\t}\n}\n", "gives no value, so the yield takes none"},
		{"no value for a block that gives one", "func main() {\n\tint v = #pick {\n\t\tyield #pick;\n\t};\n}\n", "the yield needs one"},
		{"an unknown label", "func main() {\n\t#blk {\n\t\tyield #nowhere;\n\t}\n}\n", "unknown label"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := errorsOf(tc.body)
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Errorf("errors = %v, want one with %q", got, tc.want)
			}
		})
	}

	t.Run("completion after yield offers blocks only", func(t *testing.T) {
		src := "func main() {\n\t#blk {\n\t\t#lp for i in 0..=3 {\n\t\t\tyield #¦\n\t\t}\n\t}\n}\n"
		got := c.open(map[string]string{"main.wsp": src}).completion()
		if !slices.Contains(got, "blk") || slices.Contains(got, "lp") {
			t.Errorf("completion = %v", got)
		}
	})
	t.Run("completion after break offers loops only", func(t *testing.T) {
		src := "func main() {\n\t#blk {\n\t\t#lp for i in 0..=3 {\n\t\t\tbreak #¦\n\t\t}\n\t}\n}\n"
		got := c.open(map[string]string{"main.wsp": src}).completion()
		if !slices.Contains(got, "lp") || slices.Contains(got, "blk") {
			t.Errorf("completion = %v", got)
		}
	})
}

func TestConvertingAnInterfaceBackToItsType(t *testing.T) {
	c := start(t)
	head := "interface Shape {\n\tfunc area() int;\n}\n\nstruct Square implements Shape {\n\tint side;\n}\n\nstruct Plain {\n\tint x;\n}\n\nfunc (*Square s) area() int => s.side;\n\n"
	errorsOf := func(body string) []string {
		var out []string
		for _, d := range c.open(map[string]string{"main.wsp": head + body}).diagnostics() {
			if d.Severity == 1 {
				out = append(out, d.Message)
			}
		}
		return out
	}
	if got := errorsOf("func main() {\n\tSquare sq = Square{side: 1};\n\tShape sh = &sq;\n\t?*Square a = (?*Square)(sh);\n\t*Square b = (*Square)(sh);\n}\n"); len(got) != 0 {
		t.Errorf("valid conversions: %v", got)
	}
	if got := errorsOf("func main() {\n\tSquare sq = Square{side: 1};\n\tShape sh = &sq;\n\t?*Plain p = (?*Plain)(sh);\n}\n"); len(got) != 1 || !strings.Contains(got[0], "can never be what Shape holds") {
		t.Errorf("a type that cannot be held: %v", got)
	}
}

func TestAnInterfaceIsNeverNull(t *testing.T) {
	c := start(t)
	src := "interface Shape {\n\tfunc area() int;\n}\n\nfunc check(Shape plain, ?Shape maybe) {\n\tbool a = plain == null;\n\tbool b = maybe == null;\n}\n\nfunc main() {}\n"
	var got []string
	for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
		if d.Severity == 1 {
			got = append(got, d.Message)
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], "Shape is never null") {
		t.Errorf("errors = %v", got)
	}
}

func TestCompletionInsideAnArrowBody(t *testing.T) {
	c := start(t)
	head := "struct Builder {\n\t[]byte bytes;\n\tint size;\n}\n\n"
	cases := []struct{ name, body string }{
		{"a method with a prefix", "public inline func (*Builder b) text() string => string(b.by¦);\n"},
		{"a method after the dot", "public inline func (*Builder b) text() string => string(b.¦);\n"},
		{"a plain result", "public inline func (*Builder b) len() int => b.si¦;\n"},
		{"inside an index", "public inline func (*Builder b) text() string => string(b.bytes[..b.si¦]);\n"},
		{"a lambda", "func make(*Builder b) () => int {\n\treturn () int => b.si¦;\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": head + tc.body}).completion()
			if !slices.Contains(got, "bytes") && !slices.Contains(got, "size") {
				t.Errorf("the fields are not offered: %v", got)
			}
		})
	}
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
	match type v {
		int i => return "int";
		_ => return "other";
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
	good := "interface Writer {\n\tfunc write([]byte data) !int;\n}\n\ninterface Number { int | float }\n\nstruct Buffer implements Writer {\n\tint length;\n}\n\nfunc (*Buffer b) write([]byte data) !int {\n\treturn len(data);\n}\n\nfunc sum<Number T>([]T items) T {\n\tT total;\n\tfor i in 0..len(items) {\n\t\ttotal += items[i];\n\t}\n\treturn total;\n}\n\nfunc send<Writer W>(W w, []byte data) !int {\n\treturn w.write(data)!;\n}\n\nfunc main() {\n\tBuffer b;\n\t[]byte data;\n\tint n = send(&b, data) ?? 0;\n\t[]int xs;\n\tint total = sum(xs);\n}\n"
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
	p := c.open(map[string]string{"main.wsp": main, "shapes.wsp": "#standalone\n" + shapes})
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
		{"a case that can never match", "interface Writer { func write(int n) int; }\nstruct P { int x; }\nfunc use(Writer w) int {\n  match type w {\n    *P p => return p.x;\n    _ => return 0;\n  }\n}\nfunc main() {}\n", "can never be what Writer holds"},
		{"a case that is not a pointer", "interface Writer { func write(int n) int; }\nstruct B implements Writer { int t; }\nfunc (*B b) write(int n) int { return n; }\nfunc use(Writer w) int {\n  match type w {\n    B b => return b.t;\n    _ => return 0;\n  }\n}\nfunc main() {}\n", "names a pointer to a type"},
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

	good := "interface Writer { func write(int n) int; }\ninterface Closer { func close(); }\ninterface WriteCloser { Writer & Closer }\nstruct B implements WriteCloser { int total; }\nfunc (*B b) write(int n) int { b.total += n; return b.total; }\nfunc (*B b) close() { b.total = 0; }\nstruct Log { Writer out; ?Writer backup; }\nfunc emit(Writer w, int n) int { return w.write(n); }\nfunc pick(bool first, Writer a, Writer b) Writer { if first => return a; return b; }\nfunc total(Writer w) int {\n  match type w {\n    *B b => return b.total;\n    _ => return -1;\n  }\n}\nfunc main() {\n  B b;\n  Writer w = &b;\n  WriteCloser wc = &b;\n  wc.close();\n  Log log = Log{out: &b, backup: &b};\n  int r = w.write(1) + emit(&b, 2) + log.out.write(3);\n  Writer chosen = pick(true, &b, &b);\n  int t = total(chosen);\n  [2]Writer pair = .{&b, &b};\n  if log.backup |x| {\n    x.write(1);\n  }\n}\n"
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
		{"optional return match arm", "func f(Token k) ?Token {\n\treturn match k {\n\t\t.Number => .¦;\n\t\t_ => null;\n\t};\n}\n", tokens},
		{"variant fields in an arm", "func f(Token k) ?AST {\n\treturn match k {\n\t\t.Number => .Literal{¦};\n\t\t_ => null;\n\t};\n}\n", []string{"value"}},
		{"member after a line without a semicolon", "func main() {\n\tLit a = .{value: 1.0}\n\ta.¦\n}\n", []string{"value", "node"}},
		{"binding of a type match arm", "func f(Node n) {\n\tmatch type n {\n\t\t*Lit l => l.¦\n\t\t_ => {}\n\t}\n}\n", []string{"value", "node"}},
		{"binding of a type match block arm", "func f(Node n) {\n\tmatch type n {\n\t\t*Lit l => {\n\t\t\tl.¦\n\t\t}\n\t\t_ => {}\n\t}\n}\n", []string{"value", "node"}},
		{"binding of a variant arm", "func f(AST a) {\n\tmatch a {\n\t\t.Literal l => l.¦\n\t\t_ => {}\n\t}\n}\n", []string{"value"}},
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

	members := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0..=9;\n\tint n = r.¦\n}\n"}).completion()
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

	hover := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0..=9;\n\tint n = r.cou¦nt();\n}\n"}).hover()
	if !strings.Contains(hover, "count() int") || !strings.Contains(hover, "How many values") {
		t.Errorf("hover of count: %q", hover)
	}

	dots := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0.¦.9;\n}\n"}).hover()
	if !strings.Contains(dots, "Range<int>") || strings.Contains(dots, "__wisp") {
		t.Errorf("hover of ..: %q", dots)
	}

	typed := c.open(map[string]string{"main.wsp": "func main() {\n\tRan¦ge<int> r = 0..=9;\n}\n"})
	if hover := typed.hover(); !strings.Contains(hover, "struct Range<T>") {
		t.Errorf("hover of Range: %q", hover)
	}
	if defs := typed.definition(); len(defs) != 0 {
		t.Errorf("Range has no file to open: %v", defs)
	}

	method := c.open(map[string]string{"main.wsp": "func main() {\n\tRange<int> r = 0..=9;\n\tint n = r.cou¦nt();\n}\n"})
	if defs := method.definition(); len(defs) != 0 {
		t.Errorf("a builtin method has no file to open: %v", defs)
	}
}

func TestStdModulesOpenedAsFiles(t *testing.T) {
	c := start(t)
	for _, name := range []string{"mem/mem.wsp", "mem/debug.wsp", "mem/arena.wsp", "seq/vec.wsp", "io/io.wsp", "io/adapters.wsp", "strings/builder.wsp", "os/file.wsp", "rand.wsp", "time/duration.wsp"} {
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

func TestDiagnosticsOfOperatorsAndCalls(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src, want string }{
		{"!! on a call with no result", "func abc() {\n}\n\nfunc main() {\n\tabc()!!;\n}\n", "abc gives no value and cannot fail, so !! has nothing to check"},
		{"calling a test", "#test\nfunc abc() ! {\n}\n\nfunc main() {\n\tabc()!!;\n}\n", "abc is a test: only `wisp test` runs it, so it cannot be called"},
		{"eq with the wrong signature", "struct P { int x; }\nfunc (P a) eq(int b) bool { return true; }\nfunc main() {\n\tP p;\n\tP q;\n\tbool same = p == q;\n}\n", "eq of P has to be func (P a) eq(P b) bool"},
		{"eq on a type that has ==", "type Meters int;\nfunc (Meters a) eq(Meters b) bool { return true; }\nfunc main() {}\n", "Meters already has ==, so eq would never be called"},
		{"in on structs with no eq", "struct P { int x; }\nfunc main() {\n\tP p;\n\t[2]P ps = .{p, p};\n\tbool has = p in ps;\n}\n", "'in' compares the elements with ==, and P has no eq"},
		{"an operator method in a static block", "struct P { int x; }\nfunc (P a) eq(P b) bool { return true; }\nconst SAME = static {\n\tP a = .{x: 1};\n\tP b = .{x: 1};\n\tyield a == b;\n};\nfunc main() {}\n", "this operator goes through a method of a type"},
		{"== on a struct with no eq", "struct P { int x; }\nfunc main() {\n\tP p;\n\tP q;\n\tbool same = p == q;\n}\n", "operator == is not defined for P"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, d := range c.open(map[string]string{"main.wsp": tc.src}).diagnostics() {
				if strings.Contains(d.Message, tc.want) {
					return
				}
			}
			t.Errorf("no diagnostic says %q", tc.want)
		})
	}
}

func TestUseAfterFreeWarning(t *testing.T) {
	c := start(t)
	head := "import \"std/mem\";\n\nfunc main() ! {\n"
	cases := []struct {
		name, body string
		warns      bool
	}{
		{"a pointer used after the free", "\t*int p = (*int)(mem.heap.alloc(8)!);\n\tmem.heap.free(p);\n\tp@ = 2;\n", true},
		{"a slice used after the free", "\t[]int xs = mem.make<int>(mem.heap, 3)!;\n\tmem.free(mem.heap, xs);\n\txs[0] = 1;\n", true},
		{"given a new value first", "\t*int p = (*int)(mem.heap.alloc(8)!);\n\tmem.heap.free(p);\n\tp = (*int)(mem.heap.alloc(8)!);\n\tp@ = 2;\n\tmem.heap.free(p);\n", false},
		{"freed only in a branch", "\t*int p = (*int)(mem.heap.alloc(8)!);\n\tint n = 1;\n\tif n > 0 {\n\t\tmem.heap.free(p);\n\t}\n\tp@ = 1;\n\tmem.heap.free(p);\n", false},
		{"only compared", "\t*int p = (*int)(mem.heap.alloc(8)!);\n\tmem.heap.free(p);\n\t*int q = (*int)(mem.heap.alloc(8)!);\n\tbool same = q == p;\n\tmem.heap.free(q);\n", false},
		{"only the length is read", "\t[]int xs = mem.make<int>(mem.heap, 3)!;\n\tmem.free(mem.heap, xs);\n\tint n = len(xs);\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			warned := false
			for _, d := range c.open(map[string]string{"main.wsp": head + tc.body + "}\n"}).diagnostics() {
				if strings.Contains(d.Message, "is used after it was freed") {
					warned = true
				}
			}
			if warned != tc.warns {
				t.Errorf("warned %v, want %v", warned, tc.warns)
			}
		})
	}
}

func TestHoverOfLoweredLoops(t *testing.T) {
	c := start(t)
	for _, tc := range []struct{ name, src, want string }{
		{"pointer loop variable", "func fill<T>([]T items, T v) {\n\tfor ¦k in &items {\n\t\tk@ = v;\n\t}\n}\nfunc main() {}\n", "*T k"},
		{"iterator loop variable", "struct C { int n; }\nfunc (*C c) next() ?int { return null; }\nfunc main() {\n\tC c;\n\tfor ¦x in c {\n\t\tint y = x;\n\t}\n}\n", "int x"},
		{"for keyword", "func fill<T>([]T items, T v) {\n\t¦for k in &items {\n\t\tk@ = v;\n\t}\n}\nfunc main() {}\n", ""},
	} {
		got := c.open(map[string]string{"main.wsp": tc.src}).hover()
		if tc.want == "" && got != "" || !strings.Contains(got, tc.want) {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDefinitionOfInlineMethodOfGenericStruct(t *testing.T) {
	c := start(t)
	src := "struct L<T> { []T items; }\ninline func (*L<T> l) size() int { return len(l.items); }\nfunc main() {\n\tL<int> l;\n\tint n = l.si¦ze();\n}\n"
	defs := c.open(map[string]string{"main.wsp": src}).definition()
	if len(defs) != 1 || defs[0].Range.Start.Line != 1 {
		t.Errorf("definition of an inline method of a generic struct: %+v", defs)
	}
}

func TestEnumCompletionInGenericCalls(t *testing.T) {
	c := start(t)
	pre := "enum Mode { Fast, Slow }\nstruct Thing { int x; }\nfunc make<T>(int a, int b, int c, Mode m) !?T { return null; }\n"
	for name, files := range map[string]map[string]string{
		"explicit arguments": {"main.wsp": pre + "func main() ! {\n\tThing t = make<Thing>(1, 2, 3, .¦);\n}\n"},
		"unfinished call":    {"main.wsp": pre + "func main() ! {\n\t?Thing t = make<Thing>(1, 2, 3, .¦\n}\n"},
		"from a module": {
			"main.wsp":    "import \"geo\";\nfunc main() ! {\n\tgeo.Thing t = geo.make<geo.Thing>(1, 2, 3, .¦);\n}\n",
			"geo/geo.wsp": "public enum Mode { Fast, Slow }\npublic struct Thing { int x; }\npublic func make<T>(int a, int b, int c, Mode m) !?T { return null; }\n",
		},
	} {
		got := c.open(files).completion()
		if len(got) != 2 || got[0] != "Fast" || got[1] != "Slow" {
			t.Errorf("%s: got %v", name, got)
		}
	}
}

func TestMethodsWithTypeParametersInTheEditor(t *testing.T) {
	c := start(t)
	decl := "import \"std/mem\";\nstruct Window { int w; }\nfunc (*Window w) draw<mem.Allocator A>(A allocator) int => w.w;\n"

	good := c.open(map[string]string{"main.wsp": decl + "func main() {\n\tWindow win = Window{w: 3};\n\tint n = win.dr¦aw(mem.heap);\n\tn++;\n}\n"})
	for _, d := range good.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("valid generic method reported: %s", d.Message)
		}
	}
	if hover := good.hover(); !strings.Contains(hover, "func (*Window w) draw(") || strings.Contains(hover, "__") {
		t.Errorf("hover: %q", hover)
	}

	sig := c.open(map[string]string{"main.wsp": decl + "func main() {\n\tWindow win = Window{w: 3};\n\tint n = win.draw(¦);\n\tn++;\n}\n"})
	if label, _ := sig.signature(); label != "draw(A allocator)" {
		t.Errorf("signature: %q", label)
	}

	bad := c.open(map[string]string{"main.wsp": decl + "func main() {\n\tWindow win = Window{w: 3};\n\tint n = win.draw(5);\n\tn++;\n}\n"})
	found := false
	for _, d := range bad.diagnostics() {
		if d.Severity == 1 && strings.Contains(d.Message, "does not satisfy") && !strings.Contains(d.Message, "__") {
			found = true
		}
	}
	if !found {
		t.Errorf("bound violation not reported")
	}
}

func TestDependencyProjectsInTheEditor(t *testing.T) {
	c := start(t)
	files := map[string]string{
		"wisp.toml":             "[dependencies]\ngeo = { path = \"geometry\" }\n",
		"main.wsp":              "import \"geo\";\nfunc main() {\n\tint n = geo.ar¦ea(2, 3);\n\tint m = geo.double(n);\n}\n",
		"geometry/geometry.wsp": "import \"util\";\npublic func area(int w, h) int => util.mul(w, h);\npublic func double(int n) int => n * 2;\n",
		"geometry/util.wsp":     "public func mul(int a, b) int => a * b;\n",
	}
	p := c.open(files)

	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("unexpected error: %s", d.Message)
		}
	}

	defs := p.definition()
	if len(defs) != 1 || !strings.HasSuffix(defs[0].URI, "geometry/geometry.wsp") || defs[0].Range.Start.Line != 1 {
		t.Errorf("the definition should be area in the dependency: %+v", defs)
	}

	if got := p.hover(); !strings.Contains(got, "func area(int w, int h) int") {
		t.Errorf("hover: %q", got)
	}

	typed := c.open(map[string]string{
		"wisp.toml":             files["wisp.toml"],
		"main.wsp":              "import \"geo\";\nfunc main() {\n\tgeo.¦\n}\n",
		"geometry/geometry.wsp": files["geometry/geometry.wsp"],
		"geometry/util.wsp":     files["geometry/util.wsp"],
	})
	got := strings.Join(typed.completion(), " ")
	if !strings.Contains(got, "area") || !strings.Contains(got, "double") {
		t.Errorf("completion after geo.: %s", got)
	}
}

func TestMissingDependencyIsAnImportError(t *testing.T) {
	c := start(t)
	p := c.open(map[string]string{
		"wisp.toml": "[dependencies]\ngeo = { path = \"nowhere\" }\n",
		"main.wsp":  "import \"geo\";\nfunc main() {}\n",
	})
	found := false
	for _, d := range p.diagnostics() {
		if strings.Contains(d.Message, "which is not a folder") {
			found = true
		}
	}
	if !found {
		t.Errorf("diagnostics: %+v", p.diagnostics())
	}
}

func TestWorkspaceMembersInTheEditor(t *testing.T) {
	c := start(t)
	p := c.openAs("tools/main.wsp", map[string]string{
		"wisp.toml":             "[workspace]\nmembers = [\"geometry\", \"tools\"]\n",
		"geometry/wisp.toml":    "[project]\nname = \"geo\"\n",
		"geometry/geometry.wsp": "public func area(int w, h) int => w * h;\n",
		"tools/wisp.toml":       "",
		"tools/main.wsp":        "import \"geo\";\nfunc main() {\n\tint n = geo.ar¦ea(2, 3);\n}\n",
	})
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("unexpected error: %s", d.Message)
		}
	}
	defs := p.definition()
	if len(defs) != 1 || !strings.HasSuffix(defs[0].URI, "geometry/geometry.wsp") {
		t.Errorf("the definition should be in the member geometry: %+v", defs)
	}
}

func errorsOf(p *project) []string {
	var out []string
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			out = append(out, d.Message)
		}
	}
	return out
}

func TestInternalMeansTheWholeProject(t *testing.T) {
	c := start(t)

	same := c.open(map[string]string{
		"wisp.toml":   "",
		"main.wsp":    "import \"lib\";\nfunc main() {\n\tint n = lib.helper();\n}\n",
		"lib/lib.wsp": "internal func helper() int => 1;\n",
	})
	if got := errorsOf(same); len(got) != 0 {
		t.Errorf("another folder of the same project may use an internal function: %q", got)
	}

	dependent := c.open(map[string]string{
		"wisp.toml":             "[dependencies]\ngeo = { path = \"geometry\" }\n",
		"main.wsp":              "import \"geo\";\nfunc main() {\n\tint n = geo.hidden();\n\tint m = geo.shown();\n}\n",
		"geometry/wisp.toml":    "",
		"geometry/geometry.wsp": "internal func hidden() int => 1;\npublic func shown() int => hidden();\n",
	})
	got := errorsOf(dependent)
	if len(got) != 1 || !strings.Contains(got[0], `hidden is internal to the project "geometry"`) {
		t.Errorf("a project that depends on another cannot see its internal names: %q", got)
	}

	field := c.open(map[string]string{
		"wisp.toml":             "[dependencies]\ngeo = { path = \"geometry\" }\n",
		"main.wsp":              "import \"geo\";\nfunc main() {\n\tgeo.Box b = geo.make();\n\tint n = b.cache;\n\tint w = b.w;\n}\n",
		"geometry/wisp.toml":    "",
		"geometry/geometry.wsp": "public struct Box {\n\tpublic int w;\n\tinternal int cache;\n}\npublic func make() Box => Box{w: 1, cache: 2};\n",
	})
	got = errorsOf(field)
	if len(got) != 1 || !strings.Contains(got[0], `field cache of Box is internal to the project "geometry"`) {
		t.Errorf("an internal field is hidden from the projects that depend on it: %q", got)
	}
}

func TestPrivateKeepsANameInItsFile(t *testing.T) {
	c := start(t)

	field := c.open(map[string]string{
		"wisp.toml": "",
		"main.wsp":  "import \"box\";\nfunc main() {\n\tbox.Box b = box.make();\n\tint n = b.secret;\n\tint m = b.open;\n}\n",
		"box.wsp":   "#standalone\npublic struct Box {\n\tprivate int secret;\n\tpublic int open;\n}\npublic func make() Box => Box{secret: 1, open: 2};\n",
	})
	got := errorsOf(field)
	if len(got) != 1 || !strings.Contains(got[0], "field secret of Box is private to box.wsp") {
		t.Errorf("a private field is only for its own file: %q", got)
	}

	own := c.open(map[string]string{
		"wisp.toml": "",
		"main.wsp":  "struct Box {\n\tprivate int secret;\n}\nfunc main() {\n\tBox b = Box{secret: 1};\n\tint n = b.secret;\n}\n",
	})
	if got := errorsOf(own); len(got) != 0 {
		t.Errorf("its own file uses it freely: %q", got)
	}

	function := c.open(map[string]string{
		"wisp.toml": "",
		"main.wsp":  "import \"util\";\nfunc main() {\n\tint n = util.helper();\n}\n",
		"util.wsp":  "#standalone\nprivate func helper() int => 1;\npublic func shown() int => helper();\n",
	})
	got = errorsOf(function)
	if len(got) != 1 || !strings.Contains(got[0], "module has no public helper") {
		t.Errorf("a private function is not exported: %q", got)
	}
}

func TestSharedPointsToInternal(t *testing.T) {
	c := start(t)
	p := c.open(map[string]string{
		"wisp.toml": "",
		"main.wsp":  "shared func f() int => 1;\nstruct S {\n\tshared int x;\n}\nfunc main() {}\n",
	})
	got := errorsOf(p)
	if len(got) != 2 {
		t.Fatalf("one error for the function and one for the field: %q", got)
	}
	for _, message := range got {
		if !strings.Contains(message, "shared was replaced by internal, which now means the whole project") {
			t.Errorf("unexpected message: %q", message)
		}
	}
}

func TestAFolderIsOneModuleInTheEditor(t *testing.T) {
	c := start(t)
	files := map[string]string{
		"wisp.toml":     "",
		"main.wsp":      "import \"net\";\nfunc main() {\n\tint n = net.pi¦ng(1);\n\tint m = helper();\n}\n",
		"util.wsp":      "func helper() int => 7;\n",
		"net/net.wsp":   "public func ping(int n) int => encode(n) + 1;\n",
		"net/codec.wsp": "func encode(int n) int => n * 10;\n",
	}
	p := c.open(files)
	if got := errorsOf(p); len(got) != 0 {
		t.Errorf("sibling files and a folder module need no import: %q", got)
	}
	defs := p.definition()
	if len(defs) != 1 || !strings.HasSuffix(defs[0].URI, "net/net.wsp") {
		t.Errorf("the definition of net.ping: %+v", defs)
	}

	inFolder := c.openAs("net/net.wsp", map[string]string{
		"wisp.toml":     "",
		"main.wsp":      files["main.wsp"],
		"util.wsp":      files["util.wsp"],
		"net/net.wsp":   "public func ping(int n) int => enc¦ode(n) + 1;\n",
		"net/codec.wsp": files["net/codec.wsp"],
	})
	if got := errorsOf(inFolder); len(got) != 0 {
		t.Errorf("a file uses the names of its sibling: %q", got)
	}
	if defs := inFolder.definition(); len(defs) != 1 || !strings.HasSuffix(defs[0].URI, "net/codec.wsp") {
		t.Errorf("the definition of a name from a sibling file: %+v", defs)
	}

	alone := c.open(map[string]string{
		"wisp.toml": "",
		"main.wsp":  "import \"util\";\nfunc main() {}\n",
		"util.wsp":  "func helper() int => 7;\n",
	})
	got := errorsOf(alone)
	if len(got) != 1 || !strings.Contains(got[0], "is one of the files of the module of its folder") || !strings.Contains(got[0], "#standalone") {
		t.Errorf("importing a file of the same folder: %q", got)
	}
}

func TestPrivateNamesOfSeveralFilesDoNotCollide(t *testing.T) {
	c := start(t)
	p := c.open(map[string]string{
		"wisp.toml": "",
		"main.wsp":  "func main() {\n\tint n = helper();\n}\n",
		"a.wsp":     "private func helper() int => 1;\nfunc fromA() int => helper();\n",
		"b.wsp":     "private func helper() int => 2;\nfunc fromB() int => helper();\n",
	})
	got := errorsOf(p)
	if len(got) != 1 || !strings.Contains(got[0], "identifier helper not declared") {
		t.Fatalf("only main.wsp is wrong: %q", got)
	}
	for _, d := range p.diagnostics() {
		if strings.Contains(d.Message, "helper is only defined as private in other files of this module") && strings.Contains(d.Message, ".wsp") {
			t.Errorf("the hint must not list files: %q", d.Message)
		}
	}
}

func TestMessagesNeverShowTheInternalNameOfARepeatedPrivateName(t *testing.T) {
	c := start(t)
	p := c.openAs("a.wsp", map[string]string{
		"wisp.toml": "",
		"main.wsp":  "func main() {}\n",
		"a.wsp":     "private struct P { int x; }\nprivate enum Color { Red }\nfunc fromA() int {\n\tP p = P{x: 1};\n\tstring s = p;\n\treturn 1;\n}\n",
		"b.wsp":     "private struct P { int y; }\nprivate enum Color { Blue }\nfunc fromB() int {\n\tP p = P{y: 1};\n\treturn p.y;\n}\n",
	})
	found := false
	for _, d := range p.diagnostics() {
		if strings.Contains(d.Message, "__") {
			t.Errorf("an internal name leaked: %q", d.Message)
		}
		if strings.Contains(d.Message, "P") && d.Severity == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error that names P: %+v", p.diagnostics())
	}
}

func TestTheEditorSeesThePrivateNameOfItsOwnFile(t *testing.T) {
	c := start(t)
	files := func(a, b string) map[string]string {
		return map[string]string{
			"wisp.toml": "",
			"main.wsp":  "func main() {}\n",
			"a.wsp":     a,
			"b.wsp":     b,
		}
	}
	a := "private func helper() int => 1;\nfunc fromA() int => helper();\n"
	b := "private func helper() int => 2;\nfunc fromB() int => helper();\n"

	inA := c.openAs("a.wsp", files("private func helper() int => 1;\nfunc fromA() int => hel¦per();\n", b))
	if defs := inA.definition(); len(defs) != 1 || !strings.HasSuffix(defs[0].URI, "/a.wsp") || defs[0].Range.Start.Line != 0 {
		t.Errorf("the helper of a.wsp: %+v", defs)
	}

	inB := c.openAs("b.wsp", files(a, "private func helper() int => 2;\nfunc fromB() int => hel¦per();\n"))
	if defs := inB.definition(); len(defs) != 1 || !strings.HasSuffix(defs[0].URI, "/b.wsp") {
		t.Errorf("the helper of b.wsp: %+v", defs)
	}
	if got := inB.hover(); !strings.Contains(got, "func helper() int") {
		t.Errorf("hover on the helper of b.wsp: %q", got)
	}

	typing := c.openAs("b.wsp", files(a, "private func helper() int => 2;\nfunc fromB() int {\n\treturn hel¦\n}\n"))
	if got := typing.completion(); !slices.Contains(got, "helper") {
		t.Errorf("the private helper of this file is offered: %v", got)
	}
}

func programsProject(toml string) map[string]string {
	return map[string]string{
		"wisp.toml":       toml,
		"main.wsp":        "func main() {}\n",
		"tools/main.wsp":  "func main() {}\n",
		"server/main.wsp": "func main() {}\n",
		"lib/lib.wsp":     "public func helper() int => 1;\n",
	}
}

func TestTheTomlMustNameRealPrograms(t *testing.T) {
	c := start(t)
	p := c.openAs("wisp.toml", programsProject("default = \"ghost\"\n\n[program.nothing]\nlink = []\n"))
	var messages []string
	for _, d := range p.diagnostics() {
		messages = append(messages, d.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, want := range []string{`default names "ghost"`, `"nothing" is not a program of this project`, "server, tools"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the diagnostics lack %q:\n%s", want, joined)
		}
	}

	ok := c.openAs("wisp.toml", programsProject("default = \"tools\"\n\n[program.server]\nname = \"srv\"\n"))
	if got := ok.diagnostics(); len(got) != 0 {
		t.Errorf("a toml that names real programs has no problems: %+v", got)
	}
}

func TestTheTomlOffersTheProgramsThatWereFound(t *testing.T) {
	c := start(t)
	header := c.openAs("wisp.toml", programsProject("[program.¦\n")).completion()
	for _, want := range []string{"program.tools", "program.server"} {
		if !slices.Contains(header, want) {
			t.Errorf("the header completion lacks %q: %v", want, header)
		}
	}
	if slices.Contains(header, "program.lib") {
		t.Errorf("a folder without main is not a program: %v", header)
	}

	values := c.openAs("wisp.toml", programsProject("default = \"¦\"\n")).completion()
	for _, want := range []string{"tools", "server"} {
		if !slices.Contains(values, want) {
			t.Errorf("the default completion lacks %q: %v", want, values)
		}
	}
}

func TestCInteropInTheEditor(t *testing.T) {
	c := start(t)
	src := "import \"std/errors\";\nextern type Db;\nextern \"sqlite3_\" {\n\tfunc open(*c_char path, **Db out) c_int;\n\t#fails(result < 0, errors.Kind.Invalid)\n\tfunc close(*Db db) !c_int;\n}\nfunc main() !int {\n\tDb* none = null;\n\treturn 0;\n}\n"
	clean := strings.Replace(src, "\tDb* none = null;\n", "\t?*Db none = null;\n\tc_int n = close(none ?? return 0)!;\n", 1)
	for _, d := range c.open(map[string]string{"main.wsp": clean}).diagnostics() {
		if d.Severity == 1 {
			t.Errorf("valid C declarations reported: %s", d.Message)
		}
	}

	value := c.open(map[string]string{"main.wsp": "extern type Db;\nfunc main() {\n\tDb d;\n}\n"})
	found := false
	for _, d := range value.diagnostics() {
		if d.Severity == 1 && strings.Contains(d.Message, "extern type") {
			found = true
		}
	}
	if !found {
		t.Errorf("an extern type used as a value was not reported")
	}

	shape := c.open(map[string]string{"main.wsp": "extern \"x_\" {\n\t#fails(result < 0, 1)\n\tfunc f(c_int a) c_int;\n}\nfunc main() {}\n"})
	found = false
	for _, d := range shape.diagnostics() {
		if d.Severity == 1 && strings.Contains(d.Message, "#fails tests the result") {
			found = true
		}
	}
	if !found {
		t.Errorf("#fails without ! was not reported")
	}

	hidden := c.open(map[string]string{"main.wsp": "extern func f(c¦);\nfunc main() {}\n"}).completion()
	for _, label := range hidden {
		if strings.HasPrefix(label, "c_") {
			t.Errorf("a C type shows up before c_ is typed: %s", label)
		}
	}
	shown := c.open(map[string]string{"main.wsp": "extern func f(c_¦);\nfunc main() {}\n"}).completion()
	if !slices.Contains(shown, "c_int") {
		t.Errorf("C types are missing after c_: %v", shown)
	}

	hover := c.open(map[string]string{"main.wsp": "extern \"x_\" {\n\tfunc ope¦n(*c_char path) c_int;\n}\nfunc main() {}\n"}).hover()
	if !strings.Contains(hover, "func open(") {
		t.Errorf("hover on an extern block function: %q", hover)
	}
}

func TestFormatStringsInTheEditor(t *testing.T) {
	c := start(t)
	cases := []struct {
		name, src, want string
	}{
		{"unclosed brace", "#format\nfunc say(string format, ...args) {}\nfunc main() {\n\tint n = 1;\n\tsay(\"a {n\");\n}\n", "never closed"},
		{"bad expression", "#format\nfunc say(string format, ...args) {}\nfunc main() {\n\tsay(\"a {1 + } b\");\n}\n", "is not valid"},
		{"unknown name", "#format\nfunc say(string format, ...args) {}\nfunc main() {\n\tsay(\"a {missing}\");\n}\n", "not declared"},
		{"tag without parameters", "#format\nfunc say(string format) {}\nfunc main() {\n\tsay(\"a\");\n}\n", "#format needs the parameters"},
	}
	for _, tc := range cases {
		found := false
		for _, d := range c.open(map[string]string{"main.wsp": tc.src}).diagnostics() {
			if d.Severity == 1 && strings.Contains(d.Message, tc.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no error with %q", tc.name, tc.want)
		}
	}

	clean := c.open(map[string]string{"main.wsp": "#format\nfunc say(string format, ...args) {}\nfunc main() {\n\tint n = 1;\n\tsay(\"a {n} {if n > 0 => 1 : 2:>4}\");\n}\n"})
	for _, d := range clean.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("valid format string reported: %s", d.Message)
		}
	}
}

func TestCallerTagInTheEditor(t *testing.T) {
	c := start(t)
	head := "import \"std/errors\";\nfunc f(string path) ! {\n\t"
	tail := ";\n}\nfunc main() {\n\tlet _ = f(\"x\");\n}\n"

	hover := c.open(map[string]string{"main.wsp": head + "return errors.erro¦rf(\"no file {path}\")" + tail}).hover()
	if !strings.Contains(hover, "func errorf(string format, ...args) Error") {
		t.Errorf("hover shows %q", hover)
	}

	label, _ := c.open(map[string]string{"main.wsp": head + "return errors.errorf(¦" + tail}).signature()
	if label != "errorf(string format, ...args) Error" {
		t.Errorf("signature help shows %q", label)
	}

	for _, call := range []string{"errors.errorf(\"no file {path}\")", "errors.annotate(Error(errors.Kind.NotFound), \"no file {path}\")"} {
		src := head + "return " + call + tail
		for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
			if d.Severity == 1 {
				t.Errorf("%s: %s", call, d.Message)
			}
		}
	}

	wrong := c.open(map[string]string{"main.wsp": "#caller\nfunc here(int n) {}\nfunc main() {\n\there(1);\n}\n"})
	found := false
	for _, d := range wrong.diagnostics() {
		if d.Severity == 1 && strings.Contains(d.Message, "#caller needs the first parameter") {
			found = true
		}
	}
	if !found {
		t.Errorf("#caller without a Here parameter was not reported")
	}
}

func TestHoverRangeInStructLiterals(t *testing.T) {
	c := start(t)
	cases := []struct {
		name, src string
		start, to int
		want      string
	}{
		{"a key", "struct P { int x; int y; }\nfunc main() {\n\tP p = P{¦x: 1, y: 2};\n\tint n = p.x + p.y;\n\tn++;\n}\n", 9, 10, "int x"},
		{"the name", "struct P { int x; int y; }\nfunc main() {\n\tP p = ¦P{x: 1, y: 2};\n\tint n = p.x + p.y;\n\tn++;\n}\n", 7, 8, "struct P"},
		{"a longer key", "struct Pt { int width; int y; }\nfunc main() {\n\tPt p = Pt{wi¦dth: 1, y: 2};\n\tint n = p.width + p.y;\n\tn++;\n}\n", 11, 16, "int width"},
	}
	for _, tc := range cases {
		p := c.open(map[string]string{"main.wsp": tc.src})
		var h struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
			Range struct {
				Start struct{ Character int } `json:"start"`
				End   struct{ Character int } `json:"end"`
			} `json:"range"`
		}
		json.Unmarshal(c.call("textDocument/hover", p.at()), &h)
		if !strings.Contains(h.Contents.Value, tc.want) || h.Range.Start.Character != tc.start || h.Range.End.Character != tc.to {
			t.Errorf("%s: hover %q range %d-%d, want %q range %d-%d", tc.name, h.Contents.Value, h.Range.Start.Character, h.Range.End.Character, tc.want, tc.start, tc.to)
		}
	}
}

func TestNativeBlocksOfOtherFilesAreNotHovered(t *testing.T) {
	c := start(t)
	native := "#target x64\n\nnative copy(*void dst, src, int n) {\n\tmov rdi, dst;\n\tmov rsi, src;\n\tmov rcx, n;\n\trep movsb;\n}\n"
	p := c.open(map[string]string{
		"main.wsp": "func run() {\n\tint n = 1;\n\tn++;\n\tint z = 2;\n\tz++;\n\tlet ¦_ = 2;\n}\n",
		"x64.wsp":  native,
	})
	hover := p.hover()
	if strings.Contains(hover, "register") || strings.Contains(hover, "assembler") {
		t.Errorf("a native block of another file of the module answered the hover: %q", hover)
	}
}

func TestDiagnosticsKeepTheLineShort(t *testing.T) {
	c := start(t)
	p := c.open(map[string]string{"main.wsp": "func f(string s) ! {\n\treturn Error(s);\n}\nfunc main() {\n\tlet _ = f(\"x\");\n}\n"})
	found := false
	for _, d := range p.diagnostics() {
		if strings.Contains(d.Primary, "constant string") {
			found = true
			if strings.Contains(d.Primary, "\n") || strings.Contains(d.Primary, "help") {
				t.Errorf("the message of the diagnostic carries more than the problem: %q", d.Primary)
			}
			if !strings.Contains(d.Message, "help: ") {
				t.Errorf("the help is lost: %q", d.Message)
			}
		}
	}
	if !found {
		t.Errorf("no diagnostic about the message of the error")
	}
}

func TestDynamicInterfacesInTheEditor(t *testing.T) {
	c := start(t)
	src := "interface Plain {\n\tfunc area() int;\n}\n\nfunc describe(...args) int {\n\tmatch type args[0] {\n\t\tPlain s => return s.area();\n\t\t_ => return 0;\n\t}\n}\n\nfunc main() {\n\tint n = describe(1);\n\tn++;\n}\n"
	var messages []string
	for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
		if d.Severity == 1 {
			messages = append(messages, d.Message)
		}
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "interface Plain needs to be dynamic to be used in a case") || !strings.Contains(messages[0], "#dynamic") {
		t.Errorf("errors %q", messages)
	}

	good := strings.Replace(src, "interface Plain", "#dynamic\ninterface Plain", 1)
	for _, d := range c.open(map[string]string{"main.wsp": good}).diagnostics() {
		if d.Severity == 1 {
			t.Errorf("a dynamic interface in a case was reported: %s", d.Message)
		}
	}
}

func TestImportsBelongToTheirFile(t *testing.T) {
	c := start(t)
	files := map[string]string{
		"wisp.toml": "",
		"a.wsp":     "import \"std/fmt\";\n\nfunc a() {\n\tfmt.println(\"a\");\n}\n",
		"main.wsp":  "func b() {\n\tfmt.println(\"b\");\n}\n\nfunc main() {\n\ta();\n\tb();\n}\n",
	}
	found := false
	for _, d := range c.open(files).diagnostics() {
		if d.Severity == 1 && strings.Contains(d.Message, "fmt") {
			found = true
		}
	}
	if !found {
		t.Errorf("a file used an import of another file of its folder without an error")
	}

	files["main.wsp"] = "import \"std/fmt\";\n" + files["main.wsp"]
	for _, d := range c.open(files).diagnostics() {
		if d.Severity == 1 {
			t.Errorf("each file importing for itself was reported: %s", d.Message)
		}
	}

	files["main.wsp"] = "import \"std/os\";\n\nfunc main() {\n\ta();\n}\n"
	files["extra.wsp"] = "import \"std/time\";\n\nfunc extra() int {\n\treturn time.now();\n}\n"
	warned := 0
	for _, d := range c.open(files).diagnostics() {
		if strings.Contains(d.Message, "import os is never used") {
			warned++
		}
	}
	if warned != 1 {
		t.Errorf("an import used only by another file should warn once, warned %d times", warned)
	}
}

func TestUnwrappingOptionalsInTheEditor(t *testing.T) {
	c := start(t)
	cases := []struct {
		name, line, want string
	}{
		{"!! on an optional", "int a = value!!;", "!! stops on an error, and ?int has none: use ? to stop when it is null"},
		{"! on an optional", "int a = value!;", "! passes an error up, and ?int has none"},
		{"? on a plain value", "int a = plain?;", "? expects an optional (?T), got int"},
		{"? on a call that can fail", "int a = risky()?;", "? is for an optional, and this call can fail"},
	}
	for _, tc := range cases {
		src := "func risky() !?int {\n\treturn 1;\n}\nfunc main() {\n\t?int value = 1;\n\tint plain = 2;\n\t" + tc.line + "\n\ta++;\n}\n"
		found := false
		for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
			if d.Severity == 1 && strings.Contains(d.Message, tc.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no error with %q", tc.name, tc.want)
		}
	}

	good := "func main() {\n\t?int value = 1;\n\tint a = value?;\n\ta++;\n}\n"
	for _, d := range c.open(map[string]string{"main.wsp": good}).diagnostics() {
		if d.Severity == 1 {
			t.Errorf("a valid ? was reported: %s", d.Message)
		}
	}
}

func TestStringToBytesInTheEditor(t *testing.T) {
	c := start(t)
	head := "func sum([]byte data) int {\n\treturn len(data);\n}\nfunc main() {\n\tstring text = \"abc\";\n\t"
	tail := "\n}\n"
	cases := []struct{ name, line, want string }{
		{"implicit", "int n = sum(text);\n\tn++;", "write []byte(text)"},
		{"not a string", "int k = 5;\n\tint n = sum([]byte(k));\n\tn++;", "cannot convert int to []uint8"},
		{"not bytes", "[]int wide = []int(text);\n\twide[0] = 1;", "cannot convert string to []int"},
	}
	for _, tc := range cases {
		found := false
		for _, d := range c.open(map[string]string{"main.wsp": head + tc.line + tail}).diagnostics() {
			if d.Severity == 1 && strings.Contains(d.Message, tc.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no error with %q", tc.name, tc.want)
		}
	}
	for _, d := range c.open(map[string]string{"main.wsp": head + "int n = sum([]byte(text));\n\tn++;" + tail}).diagnostics() {
		if d.Severity == 1 {
			t.Errorf("a valid conversion was reported: %s", d.Message)
		}
	}
}

func TestHoverOnOperatorsAndPunctuation(t *testing.T) {
	c := start(t)
	type result struct {
		Contents struct {
			Value string `json:"value"`
		} `json:"contents"`
		Range struct {
			Start struct{ Character int } `json:"start"`
			End   struct{ Character int } `json:"end"`
		} `json:"range"`
	}
	hoverOf := func(src string) (result, bool) {
		p := c.open(map[string]string{"main.wsp": src})
		raw := c.call("textDocument/hover", p.at())
		var h result
		if string(raw) == "null" || len(raw) == 0 {
			return h, false
		}
		json.Unmarshal(raw, &h)
		return h, h.Contents.Value != ""
	}
	head := "struct P { int x; int y; }\nfunc g(int a, int b) !int {\n\treturn a;\n}\nfunc main() ! {\n\t"

	h, ok := hoverOf(head + "int n = g(1, 2)¦!;\n\tn++;\n}\n")
	if !ok || !strings.Contains(h.Contents.Value, "Passes the error up") || h.Range.End.Character-h.Range.Start.Character != 1 {
		t.Errorf("hover on !: %q range %d-%d", h.Contents.Value, h.Range.Start.Character, h.Range.End.Character)
	}
	if h, ok := hoverOf(head + "int n = g(1, 2)!¦!;\n\tn++;\n}\n"); !ok || !strings.Contains(h.Contents.Value, "Stops the program") || h.Range.End.Character-h.Range.Start.Character != 2 {
		t.Errorf("hover on !!: %q range %d-%d", h.Contents.Value, h.Range.Start.Character, h.Range.End.Character)
	}
	if h, ok := hoverOf(head + "int n = g(1¦, 2)!;\n\tn++;\n}\n"); ok {
		t.Errorf("hover on a comma of a call showed %q", h.Contents.Value)
	}
	if h, ok := hoverOf(head + "P p = P{¦x: 1, y: 2};\n\tint n = p.x + p.y;\n\tn++;\n}\n"); !ok || !strings.Contains(h.Contents.Value, "int x") {
		t.Errorf("hover on a key showed %q", h.Contents.Value)
	}
	if h, ok := hoverOf(head + "P p = P{x¦: 1, y: 2};\n\tint n = p.x + p.y;\n\tn++;\n}\n"); ok {
		t.Errorf("hover on the colon of a struct literal showed %q", h.Contents.Value)
	}
}

func TestCompletionAfterADotInALoopCondition(t *testing.T) {
	c := start(t)
	ioFile := "public interface Reader {\n\tfunc read([]byte buf) !?int;\n}\n\npublic interface Writer {\n\tfunc write([]byte data) !int;\n}\n"
	writer := "import \"std/mem\";\n\npublic struct BufWriter implements Writer {\n\tWriter out;\n\t[]byte buf;\n\tint count;\n\t?mem.Allocator source;\n}\n\npublic func (*BufWriter w) write([]byte data) !int {\n\tmem.copy(w.buf, data, len(data));\n\tw.count += len(data);\n\treturn len(data);\n}\n\npublic func (*BufWriter w) release() {\n\tif w.source |s| {\n\t\tmem.free(s, w.buf);\n\t}\n}\n"
	for name, line := range map[string]string{"loop": "loop r.¦", "if": "if r.¦", "for": "for r.¦"} {
		reader := "import \"std/mem\";\n\npublic struct BufReader implements Reader {\n\tReader inp;\n\t[]byte buf;\n\tint start;\n\t?mem.Allocator source;\n}\n\npublic func (*BufReader r) read([]byte out) !?int {\n\t" + line + "\n}\n"
		files := map[string]string{"main.wsp": "func main() {\n}\n", "io/io.wsp": ioFile, "io/reader.wsp": reader, "io/writer.wsp": writer}
		got := c.openAs("io/reader.wsp", files).completion()
		if !slices.Contains(got, "inp") || !slices.Contains(got, "start") {
			t.Errorf("%s: the members of r are missing: %v", name, got)
		}
	}
}

func TestSliceViewsInTheEditor(t *testing.T) {
	c := start(t)
	head := "type Meters int;\nfunc main() {\n\t[3]int values = .{1, 2, 3};\n\t[3]int16 shorts = .{1, 2, 3};\n\tfloat scale = 1.5;\n\t"
	tail := "\n}\n"
	bad := []struct{ name, line, want string }{
		{"another size", "[]int small = []int(shorts);\n\tsmall[0] = 1;", "cannot convert [3]int16 to []int"},
		{"another kind", "[]float wide = []float(values);\n\twide[0] = scale;", "cannot convert [3]int to []float"},
		{"a number", "[]int none = []int(scale);\n\tnone[0] = 1;", "cannot convert float to []int"},
		{"a temporary array", "[]int part = []int(.{1, 2});\n\tpart[0] = 1;", "cannot convert"},
	}
	for _, tc := range bad {
		found := false
		for _, d := range c.open(map[string]string{"main.wsp": head + tc.line + tail}).diagnostics() {
			if d.Severity == 1 && strings.Contains(d.Message, tc.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no error with %q", tc.name, tc.want)
		}
	}
	for _, d := range c.open(map[string]string{"main.wsp": head + "[]Meters lengths = []Meters(values);\n\tlengths[0] = 2;" + tail}).diagnostics() {
		if d.Severity == 1 {
			t.Errorf("a valid view was reported: %s", d.Message)
		}
	}
}

func TestCompletionIgnoresTheScopesOfOtherFiles(t *testing.T) {
	c := start(t)
	other := "func spread(int a) int {\n\tint total = 0;\n\tif a > 0 {\n\t\ttotal = 1;\n\t\tif a > 1 {\n\t\t\ttotal = 2;\n\t\t\tif a > 2 {\n\t\t\t\ttotal = 3;\n\t\t\t}\n\t\t}\n\t}\n\treturn total;\n}\n"
	reader := "public struct Reader {\n\tint start;\n\tint end;\n}\n\npublic func (*Reader r) read() int {\n\tif star¦\n}\n"
	files := map[string]string{"main.wsp": "func main() {\n}\n", "lib/other.wsp": other, "lib/reader.wsp": reader}
	got := c.openAs("lib/reader.wsp", files).completion()
	if !slices.Contains(got, "r.start") {
		t.Errorf("the fields of the receiver are missing when another file has scopes in the same place: %d items", len(got))
	}

	files["lib/reader.wsp"] = "public struct Reader {\n\tint start;\n\tint end;\n}\n\npublic func (*Reader r) read() int {\n\tint n = min(1, r.¦)\n}\n"
	got = c.openAs("lib/reader.wsp", files).completion()
	if !slices.Contains(got, "start") || !slices.Contains(got, "end") {
		t.Errorf("the members of r inside a call are missing: %v", got)
	}
}

func TestCompletionRanksWhatYouUseFirst(t *testing.T) {
	c := start(t)
	src := "import \"std/strings\";\n\npublic struct Reader {\n\tint start;\n}\n\nfunc startOver() int {\n\treturn 1;\n}\n\npublic func (*Reader r) read(int stack) int {\n\tint state = 1;\n\tsta¦\n}\n"
	items := c.open(map[string]string{"main.wsp": src}).completionItems()
	rank := map[string]string{}
	for _, it := range items {
		rank[it.Label] = it.SortText
	}
	order := []string{"state", "r.start", "startOver", "strings.startsWith", "static", "staticAssert"}
	for i := 0; i+1 < len(order); i++ {
		a, b := rank[order[i]], rank[order[i+1]]
		if a == "" || b == "" || a >= b {
			t.Errorf("%s (%q) should come before %s (%q)", order[i], a, order[i+1], b)
		}
	}
}

func TestBlockErrorsPointAtTheEnd(t *testing.T) {
	c := start(t)
	src := "func f(int n) int {\n\treturn match n {\n\t\t1 => {\n\n\t\t};\n\t\t_ => 0;\n\t};\n}\nfunc main() {\n\tint a = f(1);\n\ta++;\n}\n"
	found := false
	for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
		if d.Severity == 1 && strings.Contains(d.Message, "must end with yield, return, break or continue") || strings.Contains(d.Message, "never yields a value") {
			found = true
			if d.Range.End.Line != d.Range.Start.Line {
				t.Errorf("the error covers lines %d to %d, and should point at the end of the block", d.Range.Start.Line, d.Range.End.Line)
			}
		}
	}
	if !found {
		t.Errorf("no error about the block")
	}
}

func TestDiagnosticsNeverCoverSeveralLines(t *testing.T) {
	c := start(t)
	src := "struct P { int x; int y; }\nfunc main() {\n\tP p = P{\n\t\tx: \"text\",\n\t\ty: 2,\n\t};\n\tint n = p.x;\n\tn++;\n}\n"
	count := 0
	for _, d := range c.open(map[string]string{"main.wsp": src}).diagnostics() {
		if d.Severity != 1 {
			continue
		}
		count++
		if d.Range.End.Line != d.Range.Start.Line {
			t.Errorf("%q covers lines %d to %d", d.Message, d.Range.Start.Line, d.Range.End.Line)
		}
	}
	if count == 0 {
		t.Errorf("the mismatch in the literal was not reported")
	}
}

func TestSemanticTokensAreNotTakenFromAnOlderVersionOfTheFile(t *testing.T) {
	c := start(t)
	good := "/// first\n/// second\nstruct Box { int size; }\n\nfunc main() {\n\tBox b = Box{size: 1};\n}\n"
	p := c.open(map[string]string{"main.wsp": good})
	if len(p.semanticTokens()) == 0 {
		t.Fatal("a clean file has semantic tokens")
	}
	broken := "/// first\n/// a much longer second line of the comment\n/// third\n/// fourth\nstruct Box { int size; }\n\nfunc main() {\n\tBox b = Box{size: 1;\n}\n"
	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": p.uri(), "version": 2},
		"contentChanges": []map[string]any{{"text": broken}},
	})
	lines := strings.Split(broken, "\n")
	for _, tok := range p.semanticTokens() {
		if strings.HasPrefix(strings.TrimSpace(lines[tok.line]), "//") {
			t.Fatalf("a token lands in a comment after an edit that does not parse: %+v", tok)
		}
	}
}

func TestMatchAndEachErrors(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src, want string }{
		{"each as a value", "func main() {\n\tint n = 3;\n\tint a = each { n > 1 => 1; };\n}\n", "each runs statements and gives no value"},
		{"each on types", "func main() {\n\tint n = 3;\n\teach type n {\n\t\tint v => n++;\n\t}\n}\n", "use match type here"},
		{"the old case word", "func main() {\n\tint n = 3;\n\tmatch n {\n\t\tcase 1 => n++;\n\t}\n}\n", "`case` is gone"},
		{"the old default word", "func main() {\n\tint n = 3;\n\tmatch n {\n\t\tdefault => n++;\n\t}\n}\n", "`default` is gone"},
		{"an arm after the last one", "func main() {\n\tint n = 3;\n\tmatch n {\n\t\t_ => n--;\n\t\t2 => n++;\n\t}\n}\n", "nothing can come after it"},
		{"an each arm after the last one", "func main() {\n\tint n = 3;\n\teach {\n\t\t_ => n--;\n\t\tn > 2 => n++;\n\t}\n}\n", "nothing can come after it"},
		{"a match that does not cover an enum", "enum Color { Red, Green }\nfunc main() {\n\tColor c = .Red;\n\tmatch c {\n\t\t.Red => return;\n\t}\n}\n", "this match does not cover .Green"},
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
}

func TestArrowBodies(t *testing.T) {
	c := start(t)
	errs := []struct{ name, src, want string }{
		{"an old guard", "func f(int n) int {\n\treturn 1 if n > 0;\n\treturn 0;\n}\nfunc main() {}\n", "the condition comes first"},
		{"an else after =>", "func main() {\n\tint n = 3;\n\tif n > 0 => n++; else { n--; }\n}\n", "writes its else after a colon"},
		{"an if inside an if with =>", "func main() {\n\tint n = 3;\n\tif n > 0 => if n > 1 => n++;\n}\n", "is not another if"},
		{"an if value without its else", "func main() {\n\tint n = 3;\n\tint v = if n > 1 => 2;\n}\n", "needs its else after a colon"},
		{"an until after =>", "func main() {\n\tint n = 3;\n\tloop => n++; until n > 5;\n}\n", "an until goes after braces"},
	}
	for _, tc := range errs {
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

	p := c.open(map[string]string{"main.wsp": "func main() {\n\tint n = 3;\n\tif n > 0 => { n++; }\n\tn--;\n}\n"})
	found := false
	for _, d := range p.diagnostics() {
		if d.Severity == 2 && strings.Contains(d.Message, "the => is not needed before braces") {
			found = true
		}
		if d.Severity == 1 {
			t.Errorf("unexpected error %q", d.Message)
		}
	}
	if !found {
		t.Errorf("no warning for braces after =>")
	}
}

func TestOldTernaryAndAssertForms(t *testing.T) {
	c := start(t)
	cases := []struct{ name, src, want string }{
		{"the old ternary", "func main() {\n\tbool c = true;\n\tint a = c ? 1 : 2;\n\ta++;\n}\n", "there is no `c ? a : b`: write `if c => a : b`"},
		{"the old assert", "func main() {\n\t?int v = 1;\n\tint a = v.?;\n\ta++;\n}\n", "the assert is written `x?` now"},
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
	p := c.open(map[string]string{"main.wsp": "func half(int n) ?int {\n\treturn if n % 2 == 0 => n / 2 : null;\n}\nfunc main() {\n\t?int v = half(8);\n\tint a = v?;\n\tint b = half(6)? + 1;\n\ta += b;\n}\n"})
	for _, d := range p.diagnostics() {
		if d.Severity == 1 {
			t.Errorf("valid code reported: %s", d.Message)
		}
	}
}
