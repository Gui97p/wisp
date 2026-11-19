package parse

import (
	"slices"
	"strings"
	"testing"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

func parse(t *testing.T, src string) (*ast.Program, []string) {
	t.Helper()
	p := parser.NewParser(lexer.NewLexer([]byte(src)))
	program := p.ParseProgram()
	var errors []string
	for _, e := range p.Errors() {
		errors = append(errors, e.Message)
	}
	return program, errors
}

func collect[T ast.Node](program *ast.Program) []T {
	var found []T
	for _, d := range program.Declarations {
		ast.Inspect(d, func(n ast.Node) bool {
			if v, ok := n.(T); ok {
				found = append(found, v)
			}
			return true
		})
	}
	return found
}

func TestTypeParameters(t *testing.T) {
	program, errors := parse(t, `struct List<T> { []T items; }
struct Ring<T, const int N> { [N]T data; }
func map<T, U>(List<T> xs) List<U> {}
func (*List<T> l) push(T v) {}
type Names<T> []List<T>;
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	var funcs []*ast.FuncDecl
	var structs []*ast.StructDecl
	for _, d := range program.Declarations {
		switch d := d.(type) {
		case *ast.FuncDecl:
			funcs = append(funcs, d)
		case *ast.StructDecl:
			structs = append(structs, d)
		case *ast.TypeDecl:
			if len(d.TypeParams) != 1 || d.TypeParams[0].Name != "T" {
				t.Errorf("type alias parameters: %+v", d.TypeParams)
			}
		}
	}
	if len(structs[0].TypeParams) != 1 || structs[0].TypeParams[0].Name != "T" || structs[0].TypeParams[0].Const {
		t.Errorf("List: %+v", structs[0].TypeParams)
	}
	ring := structs[1].TypeParams
	if len(ring) != 2 || ring[1].Name != "N" || !ring[1].Const || ring[1].Type.Name != "int" {
		t.Errorf("Ring: %+v", ring)
	}
	if len(funcs[0].TypeParams) != 2 || funcs[0].TypeParams[1].Name != "U" {
		t.Errorf("map: %+v", funcs[0].TypeParams)
	}
	if funcs[1].Receiver == nil || funcs[1].Receiver.Type.String() != "*List<T>" || len(funcs[1].TypeParams) != 0 {
		t.Errorf("push: receiver %v", funcs[1].Receiver)
	}
}

func TestTypeArguments(t *testing.T) {
	program, errors := parse(t, `func f() {
	List<int> a;
	List<List<int>> b;
	Ring<byte, 8> c;
	geo.Box<*int> d;
	[]List<string> e;
	List<int> g = List<int>{count: 1};
	first<int>(a);
	geo.thing<int, string>(1);
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	want := map[string]string{"a": "List<int>", "b": "List<List<int>>", "c": "Ring<uint8, 8>", "d": "geo.Box<*int>", "e": "[]List<string>", "g": "List<int>"}
	for _, v := range collect[*ast.VarStmt](program) {
		for _, p := range v.Vars {
			if got := p.Type.String(); got != want[p.Name] {
				t.Errorf("%s has the type %s, want %s", p.Name, got, want[p.Name])
			}
		}
	}
	generics := collect[*ast.GenericExpr](program)
	if len(generics) != 2 {
		t.Fatalf("%d generic references, want 2", len(generics))
	}
	if ast.TypeArgsText(generics[0].Args) != "<int>" || ast.TypeArgsText(generics[1].Args) != "<int, string>" {
		t.Errorf("arguments: %s %s", ast.TypeArgsText(generics[0].Args), ast.TypeArgsText(generics[1].Args))
	}
	literals := collect[*ast.StructLiteral](program)
	if len(literals) != 1 || literals[0].Name != "List" || ast.TypeArgsText(literals[0].TypeArgs) != "<int>" {
		t.Errorf("literals: %+v", literals)
	}
	ring := collect[*ast.VarStmt](program)[2].Vars[0].Type.Args
	if ring[0].Type == nil || ring[1].Expr == nil {
		t.Errorf("a type argument and a value argument: %+v", ring)
	}
}

func TestComparisonsStayComparisons(t *testing.T) {
	program, errors := parse(t, `func f(int a, b, c) {
	bool x = a < b;
	bool y = a < b && b > (c);
	bool z = a < b || c > a;
	int w = a < b ? 1 : 2;
	for i in 0..c { if i < a { } }
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	if n := len(collect[*ast.GenericExpr](program)); n != 0 {
		t.Errorf("%d comparisons were read as generics", n)
	}
}

func TestGenericFunctionAsValue(t *testing.T) {
	program, errors := parse(t, `func f(int a, b) {
	let g = first<int>;
	h(first<int>, second<string>);
	bool x = a < b;
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	var withArgs []string
	for _, id := range collect[*ast.IdentLiteral](program) {
		if len(id.TypeArgs) > 0 {
			withArgs = append(withArgs, id.Value+ast.TypeArgsText(id.TypeArgs))
		}
	}
	if len(withArgs) != 3 || withArgs[0] != "first<int>" || withArgs[2] != "second<string>" {
		t.Errorf("names with type arguments: %v", withArgs)
	}
}

func TestBoundsAreReserved(t *testing.T) {
	_, errors := parse(t, "func f<T: Number>(T a) {}\n")
	if len(errors) == 0 || !strings.Contains(errors[0], "no bound yet") {
		t.Errorf("errors: %v", errors)
	}
}

func TestBadTypeArguments(t *testing.T) {
	for _, src := range []string{"List<int x;\n", "func f<>() {}\n", "func f<1>() {}\n", "struct S<T,> {}\n"} {
		if _, errors := parse(t, src); len(errors) == 0 {
			t.Errorf("no error for %q", src)
		}
	}
}

func TestGenericTypeAfterAComma(t *testing.T) {
	program, errors := parse(t, "func fold<T>(T a, T b, Op<T> f, List<T> items, int n) T { return a; }\n")
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	var got []string
	for _, p := range program.Declarations[0].(*ast.FuncDecl).Params {
		got = append(got, p.Type.String()+" "+p.Name)
	}
	want := []string{"T a", "T b", "Op<T> f", "List<T> items", "int n"}
	if !slices.Equal(got, want) {
		t.Errorf("params %v, want %v", got, want)
	}
}

func TestGenericEnumsAndVariantLiterals(t *testing.T) {
	program, errors := parse(t, `enum Maybe<T> {
	Nothing,
	Just { T value; }
}

func f() {
	Maybe<int> a = Maybe<int>.Just{value: 1};
	Maybe<int> b = Maybe<int>.Nothing;
	switch type x {
		case Id<int> i => return;
		case *Maybe<int> m => return;
	}
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	enum := program.Declarations[0].(*ast.TypeDecl)
	if len(enum.TypeParams) != 1 || enum.TypeParams[0].Name != "T" || !enum.Data {
		t.Errorf("enum: %+v", enum)
	}
	literals := collect[*ast.StructLiteral](program)
	if len(literals) != 1 || literals[0].Name != "Maybe" || literals[0].Variant != "Just" || ast.TypeArgsText(literals[0].TypeArgs) != "<int>" {
		t.Errorf("variant literals: %+v", literals)
	}
	if n := len(collect[*ast.TypeLabel](program)); n != 2 {
		t.Errorf("%d type labels, want 2", n)
	}
}

func TestGenericEnumNeedsData(t *testing.T) {
	_, errors := parse(t, "enum Plain<T> { A, B }\n")
	if len(errors) == 0 || !strings.Contains(errors[0], "carry data") {
		t.Errorf("errors: %v", errors)
	}
}

func TestSelectorOperators(t *testing.T) {
	program, errors := parse(t, "func<x64 & linux> a() {}\nfunc<release | fast> b() {}\nfunc<!windows & (x64 | linux)> c() {}\n")
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	var selectors []string
	for _, d := range program.Declarations {
		selectors = append(selectors, d.(*ast.FuncDecl).Selector)
	}
	want := []string{"x64 & linux", "release | fast", "! windows & ( x64 | linux )"}
	if !slices.Equal(selectors, want) {
		t.Errorf("selectors %q, want %q", selectors, want)
	}
}

func TestOldSelectorOperatorsAreRejected(t *testing.T) {
	for _, src := range []string{"func<x64 && linux> a() {}\n", "func<release || fast> a() {}\n", "#target linux && x64\n"} {
		_, errors := parse(t, src)
		if len(errors) == 0 || !strings.Contains(errors[0], "in the target expression") {
			t.Errorf("%q: errors %v", src, errors)
		}
	}
}
