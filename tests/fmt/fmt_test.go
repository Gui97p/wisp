package fmttests

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Gui97p/wisp/internal/format"
)

var binary string

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestMain(m *testing.M) {
	root := repoRoot()
	binary = filepath.Join(root, "bin", "wisp")
	build := exec.Command("go", "build", "-o", binary, "./cmd/main")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestRules(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"tabs and trailing space", "func f() {\n    int a = 1;   \n\t  int b = 2;\n}\n", "func f() {\n\tint a = 1;\n\tint b = 2;\n}\n"},
		{"binary operators", "func f() {\n\tint a=1+2*3-4/5%6;\n\tbool b=a<3&&a>=1||a!=2;\n}\n", "func f() {\n\tint a = 1 + 2 * 3 - 4 / 5 % 6;\n\tbool b = a < 3 && a >= 1 || a != 2;\n}\n"},
		{"unary operators stay tight", "func f() {\n\tint a = - 1;\n\tbool b = ! true;\n\tint c = a - -1;\n}\n", "func f() {\n\tint a = -1;\n\tbool b = !true;\n\tint c = a - -1;\n}\n"},
		{"commas and calls", "func f(int a,b,int c) {\n\tg ( a,b ,c );\n}\n", "func f(int a, b, int c) {\n\tg(a, b, c);\n}\n"},
		{"literals", "func f() {\n\tP p=P{ x:1,y:2 };\n\tP q=.{ x:1 };\n\t[3]int a=.{ 1,2,3 };\n}\n", "func f() {\n\tP p = P{x: 1, y: 2};\n\tP q = .{x: 1};\n\t[3]int a = .{1, 2, 3};\n}\n"},
		{"multi-line literals keep their lines", "const [3]int T = .{\n1,\n  2,\n3\n};\n", "const [3]int T = .{\n\t1,\n\t2,\n\t3\n};\n"},
		{"a comment before a closing brace stays inside", "func f() {\n\tint a = 1;\n// the last\n  // and another\n}\nstruct P {\n\tint x;\n// after the field\n}\nfunc g() {\n\tif true {\n\t\tint b = 2;\n// in the if\n\t}\n// after the if\n}\n", "func f() {\n\tint a = 1;\n\t// the last\n\t// and another\n}\nstruct P {\n\tint x;\n\t// after the field\n}\nfunc g() {\n\tif true {\n\t\tint b = 2;\n\t\t// in the if\n\t}\n\t// after the if\n}\n"},
		{"a variant of a generic enum", "func f() {\n\tMaybe<int> a = Maybe<int>.Just{value: 7};\n\tMaybe<int> b = .Nothing;\n\tbool c = x < y && z > w;\n}\n", "func f() {\n\tMaybe<int> a = Maybe<int>.Just{value: 7};\n\tMaybe<int> b = .Nothing;\n\tbool c = x < y && z > w;\n}\n"},
		{"blocks", "func f() {\nif a{b();}else{c();}\nloop{ d(); }\n}\n", "func f() {\n\tif a { b(); } else { c(); }\n\tloop { d(); }\n}\n"},
		{"an array literal in a header", "func f() {\n\tfor x in [3]int{1,2,3}{ g(x); }\n\tfor y in [2]geo.P{a,b}{ g(y); }\n\tif ok {}\n}\nfunc h() []int {}\n", "func f() {\n\tfor x in [3]int{1, 2, 3} { g(x); }\n\tfor y in [2]geo.P{a, b} { g(y); }\n\tif ok {}\n}\nfunc h() []int {}\n"},
		{"generics", "struct List<T>{ []T items; }\nfunc map<T,U>(List<T> xs,(T)=>U f)List<U>{\n\tList<List<int>> n=List<int>{ count:1 };\n\tint a=first<int>(xs);\n\tgeo.thing<int,string>(1);\n\tbool c=a<b&&b>(a);\n}\n", "struct List<T> { []T items; }\nfunc map<T, U>(List<T> xs, (T) => U f) List<U> {\n\tList<List<int>> n = List<int>{count: 1};\n\tint a = first<int>(xs);\n\tgeo.thing<int, string>(1);\n\tbool c = a < b && b > (a);\n}\n"},
		{"generic methods and value parameters", "struct Ring<T,const int N>{ [N]T data; }\nfunc (*List<T> l) push(T v){}\ntype Names<T> []List<T>;\n", "struct Ring<T, const int N> { [N]T data; }\nfunc (*List<T> l) push(T v) {}\ntype Names<T> []List<T>;\n"},
		{"empty braces", "func f() { }\nstruct S {}\n", "func f() {}\nstruct S {}\n"},
		{"ternary and optional", "func f() {\n\tint a=c?1:2;\n\t?int b=null;\n\t?*P p=null;\n}\n", "func f() {\n\tint a = c ? 1 : 2;\n\t?int b = null;\n\t?*P p = null;\n}\n"},
		{"types", "func f([]int a,[3]*int b,*[2]int c) []int {\n\t[2][2]int g;\n}\n", "func f([]int a, [3]*int b, *[2]int c) []int {\n\t[2][2]int g;\n}\n"},
		{"results", "func f()!int {}\nfunc g() !{}\nfunc h()(int,int) {}\n", "func f() !int {}\nfunc g() ! {}\nfunc h() (int, int) {}\n"},
		{"defer with a bang", "func f() ! {\n\tdefer !g();\n\tdefer!|e| g();\n\tdefer ! |e| h(e);\n\tdefer g()!;\n\tdefer! k()!;\n}\n", "func f() ! {\n\tdefer! g();\n\tdefer! |e| g();\n\tdefer! |e| h(e);\n\tdefer g()!;\n\tdefer! k()!;\n}\n"},
		{"an optional taken out", "func f() {\n\tx=a.?+1;\n\ty=b.?[0];\n\tz=g()!.?;\n}\n", "func f() {\n\tx = a.? + 1;\n\ty = b.?[0];\n\tz = g()!.?;\n}\n"},
		{"propagation", "func f() ! {\n\tg() !;\n\tx=h()!+1;\n\ty=k()!!;\n}\n", "func f() ! {\n\tg()!;\n\tx = h()! + 1;\n\ty = k()!!;\n}\n"},
		{"ranges and slices stay tight", "func f() {\n\tfor i in (0..=n-1).step(2) { a=s[1..n-1]; }\n}\n", "func f() {\n\tfor i in (0..=n - 1).step(2) { a = s[1..n - 1]; }\n}\n"},
		{"labels", "func f() {\n\t:outer for i in 0..=3 { break :outer; }\n}\n", "func f() {\n\t:outer for i in 0..=3 { break :outer; }\n}\n"},
		{"captures", "func f() {\n\tif g()|v|{ h(v); }\n\tloop it.next()|p|{ }\n\tx=y??|e|{ yield 1; };\n}\n", "func f() {\n\tif g() |v| { h(v); }\n\tloop it.next() |p| {}\n\tx = y ?? |e| { yield 1; };\n}\n"},
		{"bit or is not a capture", "func f() {\n\tint a=b|c|d;\n}\n", "func f() {\n\tint a = b | c | d;\n}\n"},
		{"switch arms", "func f() {\nswitch v{\ncase 1=>a();\ncase 2,3=>{\nb();\nc();\n}\ndefault=>d();\n}\n}\n", "func f() {\n\tswitch v {\n\t\tcase 1 => a();\n\t\tcase 2, 3 => {\n\t\t\tb();\n\t\t\tc();\n\t\t}\n\t\tdefault => d();\n\t}\n}\n"},
		{"selector operators", "func<x64&linux>f(){}\nfunc<release|fast>g(){}\nfunc<!windows&(x64|debug)>h(){}\n", "func<x64 & linux> f() {}\nfunc<release | fast> g() {}\nfunc<!windows & (x64 | debug)> h() {}\n"},
		{"target directive operators", "#target linux&x64\nfunc f(){}\n", "#target linux & x64\nfunc f() {}\n"},
		{"interface declarations", "interface Writer{func write([]byte data)!int;func flush();}\ninterface Number{int|int64|float}\ninterface Both{Writer&Closer}\ninterface Mixed{(int|float)&!Small}\n", "interface Writer { func write([]byte data) !int; func flush(); }\ninterface Number { int | int64 | float }\ninterface Both { Writer & Closer }\ninterface Mixed { (int | float) & !Small }\n"},
		{"interface body on several lines", "interface Writer {\nfunc write([]byte data) !int;\n    func flush();\n}\n", "interface Writer {\n\tfunc write([]byte data) !int;\n\tfunc flush();\n}\n"},
		{"implements and constraints", "struct List<T>implements Iterator<T>,Sized{T head;}\nfunc log<Writer W,(int|float) T,const int N>(W w){}\nenum Color implements Printable{Red}\ntype Meters int implements Printable;\n", "struct List<T> implements Iterator<T>, Sized { T head; }\nfunc log<Writer W, (int | float) T, const int N>(W w) {}\nenum Color implements Printable { Red }\ntype Meters int implements Printable;\n"},
		{"selectors and tags", "#test( \"x\" )\nfunc<debug>f(){}\nlet<  debug >x=1;\n", "#test(\"x\")\nfunc<debug> f() {}\nlet<debug> x = 1;\n"},
		{"variadics", "func f(int ...rest) {\n\tg(rest...);\n}\nfunc h(...args) {}\n", "func f(int ...rest) {\n\tg(rest...);\n}\nfunc h(...args) {}\n"},
		{"member chains and dots", "func f() {\n\tx = a . b . c ( );\n\ty = .Red;\n\tz = p@ . x;\n}\n", "func f() {\n\tx = a.b.c();\n\ty = .Red;\n\tz = p@.x;\n}\n"},
		{"blank lines", "func f() {\n\ta();\n\n\n\n\tb();\n}\n", "func f() {\n\ta();\n\n\tb();\n}\n"},
		{"comments", "func f() {   // after the brace\n// alone\n\tint a = 1;// right behind\n\t/* block */ int b = 2;\n}\n", "func f() {  // after the brace\n\t// alone\n\tint a = 1;  // right behind\n\t/* block */ int b = 2;\n}\n"},
		{"comments at the end", "func f() {}\n// the last word\n", "func f() {}\n// the last word\n"},
		{"continuation lines", "func f() {\n\tif a ||\nb &&\nc {\n\t\tg();\n\t}\n\tx = a +\nb;\n}\n", "func f() {\n\tif a ||\n\t\tb &&\n\t\tc {\n\t\tg();\n\t}\n\tx = a +\n\t\tb;\n}\n"},
		{"arguments across lines", "func f() {\n\tg(a,\nb,\nc);\n\th(.{\nx: 1\n});\n}\n", "func f() {\n\tg(a,\n\t\tb,\n\t\tc);\n\th(.{\n\t\tx: 1\n\t});\n}\n"},
		{"imports are sorted", "import \"b\";\nimport \"std/os\";\nimport \"a\";\nimport \"std/fmt\";\n\nfunc f() {}\n", "import \"std/fmt\";\nimport \"std/os\";\nimport \"a\";\nimport \"b\";\n\nfunc f() {}\n"},
		{"imports with a comment stay", "import \"b\";\n// why\nimport \"a\";\n", "import \"b\";\n// why\nimport \"a\";\n"},
		{"raw strings", "func f() {\n\ts = `a\n   b`;\n\tt = \"x  y\";\n}\n", "func f() {\n\ts = `a\n   b`;\n\tt = \"x  y\";\n}\n"},
		{"native blocks keep their lines", "native<x64> n(int x) int r {\n\t\tmov rax,x;\n\t\t  mov r , rax;\n}\n", "native<x64> n(int x) int r {\n\tmov rax,x;\n\t  mov r , rax;\n}\n"},
		{"directive", "#target x64\n\nimport \"std/os\";\n", "#target x64\n\nimport \"std/os\";\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := format.Source([]byte(tc.in))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("formatted wrongly\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
			again, err := format.Source(got)
			if err != nil {
				t.Fatalf("second pass: %v", err)
			}
			if !bytes.Equal(got, again) {
				t.Errorf("not idempotent\n--- first ---\n%s\n--- second ---\n%s", got, again)
			}
		})
	}
}

func TestRefusesSyntaxErrors(t *testing.T) {
	for _, src := range []string{"func f( {\n", "int a = ;\n", "func f() {\n\tx = \"unterminated;\n}\n"} {
		if _, err := format.Source([]byte(src)); err == nil {
			t.Errorf("a file with a syntax error was formatted:\n%s", src)
		}
	}
}

func TestEveryFileOfTheRepositoryIsStable(t *testing.T) {
	root := repoRoot()
	count := 0
	for _, dir := range []string{"std", "tour", "tests/numeric", "benchmark"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".wsp") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			first, err := format.Source(src)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				return nil
			}
			second, err := format.Source(first)
			if err != nil {
				t.Errorf("%s: second pass: %v", path, err)
				return nil
			}
			if !bytes.Equal(first, second) {
				t.Errorf("%s is not stable", path)
			}
			count++
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
	}
	if count < 100 {
		t.Errorf("only %d files were checked", count)
	}
}

func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func run(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

const messy = "func f() {\n    int a=1;\n}\n"
const tidy = "func f() {\n\tint a = 1;\n}\n"

func TestFmtCommandWritesFiles(t *testing.T) {
	dir := project(t, map[string]string{"wisp.toml": "", "a.wsp": messy, "sub/b.wsp": messy})
	out, code := run(t, dir, "fmt")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, name := range []string{"a.wsp", "sub/b.wsp"} {
		got, _ := os.ReadFile(filepath.Join(dir, name))
		if string(got) != tidy {
			t.Errorf("%s was not formatted:\n%s", name, got)
		}
	}
	if out, code := run(t, dir, "fmt", "--check"); code != 0 {
		t.Errorf("--check on formatted files: exit %d: %s", code, out)
	}
}

func TestFmtCheckChangesNothing(t *testing.T) {
	dir := project(t, map[string]string{"wisp.toml": "", "a.wsp": messy, "ok.wsp": tidy})
	out, code := run(t, dir, "fmt", "--check")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(out, "a.wsp") || strings.Contains(out, "ok.wsp") {
		t.Errorf("the list is wrong:\n%s", out)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "a.wsp"))
	if string(got) != messy {
		t.Errorf("--check wrote the file")
	}
}

func TestFmtPathsAndExclude(t *testing.T) {
	dir := project(t, map[string]string{
		"wisp.toml":            "[fmt]\nexclude = [\"generated\", \"skip.wsp\"]\n",
		"a.wsp":                messy,
		"skip.wsp":             messy,
		"generated/g.wsp":      messy,
		"deep/generated/h.wsp": messy,
		"keep/k.wsp":           messy,
	})
	if _, code := run(t, dir, "fmt", "keep"); code != 0 {
		t.Fatal("fmt keep failed")
	}
	for name, want := range map[string]string{"keep/k.wsp": tidy, "a.wsp": messy} {
		got, _ := os.ReadFile(filepath.Join(dir, name))
		if string(got) != want {
			t.Errorf("%s: a path should restrict the run:\n%s", name, got)
		}
	}
	if _, code := run(t, dir, "fmt"); code != 0 {
		t.Fatal("fmt failed")
	}
	for name, want := range map[string]string{"a.wsp": tidy, "skip.wsp": messy, "generated/g.wsp": messy, "deep/generated/h.wsp": messy} {
		got, _ := os.ReadFile(filepath.Join(dir, name))
		if string(got) != want {
			t.Errorf("%s: wrong result:\n%s", name, got)
		}
	}
}

func TestFmtLeavesBrokenFilesAlone(t *testing.T) {
	broken := "func f( {\n"
	dir := project(t, map[string]string{"wisp.toml": "", "a.wsp": messy, "broken.wsp": broken})
	out, code := run(t, dir, "fmt")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(out, "broken.wsp") {
		t.Errorf("the broken file is not named:\n%s", out)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "broken.wsp"))
	if string(got) != broken {
		t.Errorf("a broken file was touched")
	}
	got, _ = os.ReadFile(filepath.Join(dir, "a.wsp"))
	if string(got) != tidy {
		t.Errorf("the good file was not formatted")
	}
}

func TestFmtStdout(t *testing.T) {
	dir := project(t, map[string]string{"wisp.toml": "", "a.wsp": messy})
	out, code := run(t, dir, "fmt", "--stdout", "a.wsp")
	if code != 0 || out != tidy {
		t.Errorf("exit %d, output:\n%s", code, out)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "a.wsp"))
	if string(got) != messy {
		t.Errorf("--stdout wrote the file")
	}
}
