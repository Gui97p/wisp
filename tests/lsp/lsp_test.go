package lsp

import (
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
		{"a bound is not accepted yet", "func f<T: Number>(T a) {}\nfunc main() {}\n", 1, "no bound yet"},
		{"a generic function used as a value", "func id<T>(T a) T { return a; }\nfunc main() {\n  let f = id;\n}\n", 1, "id is generic"},
		{"an error in the body of an instance", "func bad<T>(T a) T { return a.missing; }\nfunc main() {\n  int x = bad(1);\n}\n", 1, "with T = int, instantiated here"},
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
		{"declaring parameters offers nothing", "struct Pair<¦> { int a; }\nfunc main() {}\n", nil, []string{"Item", "int", "if"}},
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
