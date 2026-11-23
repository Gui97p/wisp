package parse

import (
	"slices"
	"strings"
	"testing"

	"github.com/Gui97p/wisp/internal/ast"
)

func TestInterfaceDeclarations(t *testing.T) {
	program, errors := parse(t, `interface Writer {
	func write([]byte data) !int;
	func flush();
}

interface Closer { func close() !; }

public interface WriteCloser { Writer & Closer }

interface Number { int | int64 | float | float32 }

interface Signed { Integer & !Small }

interface Iterator<T> {
	func next() ?T;
	func reset();
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	decls := collect[*ast.InterfaceDecl](program)
	if len(decls) != 6 {
		t.Fatalf("%d interfaces, want 6", len(decls))
	}

	writer := decls[0]
	if writer.Name != "Writer" || len(writer.Methods) != 2 || writer.Constraint != nil {
		t.Errorf("Writer: %+v", writer)
	}
	if m := writer.Methods[0]; m.Name != "write" || len(m.Params) != 1 || len(m.ReturnTypes) != 1 || !m.Fallible {
		t.Errorf("write: %+v", m)
	}
	if m := writer.Methods[1]; len(m.Params) != 0 || len(m.ReturnTypes) != 0 || m.Fallible {
		t.Errorf("flush: %+v", m)
	}
	if m := decls[1].Methods[0]; len(m.ReturnTypes) != 0 || !m.Fallible {
		t.Errorf("close: %+v", m)
	}

	writeCloser := decls[2]
	if !writeCloser.Exported || writeCloser.Constraint == nil || writeCloser.Constraint.String() != "Writer & Closer" {
		t.Errorf("WriteCloser: %+v", writeCloser)
	}
	if got := decls[3].Constraint.String(); got != "int | int64 | float | float32" {
		t.Errorf("Number: %q", got)
	}
	if got := decls[4].Constraint.String(); got != "Integer & !Small" {
		t.Errorf("Signed: %q", got)
	}
	if iter := decls[5]; len(iter.TypeParams) != 1 || iter.TypeParams[0].Name != "T" || len(iter.Methods) != 2 {
		t.Errorf("Iterator: %+v", iter)
	}
}

func TestConstraintPrecedence(t *testing.T) {
	cases := map[string]string{
		"A | B & C":             "A | B & C",
		"(A | B) & C":           "(A | B) & C",
		"A & !B | C":            "A & !B | C",
		"!(A | B)":              "!(A | B)",
		"A | (B | C)":           "A | B | C",
		"mod.Writer & Box<int>": "mod.Writer & Box<int>",
	}
	for src, want := range cases {
		program, errors := parse(t, "interface X { "+src+" }\n")
		if len(errors) > 0 {
			t.Errorf("%q: errors %v", src, errors)
			continue
		}
		if got := collect[*ast.InterfaceDecl](program)[0].Constraint.String(); got != want {
			t.Errorf("%q printed as %q, want %q", src, got, want)
		}
	}

	program, _ := parse(t, "interface X { A | B & C }\n")
	c := collect[*ast.InterfaceDecl](program)[0].Constraint
	if c.Kind != ast.ConstraintOr || c.Right.Kind != ast.ConstraintAnd {
		t.Errorf("& must bind tighter than |: %s", c)
	}
}

func TestImplementsClauses(t *testing.T) {
	program, errors := parse(t, `struct Buffer implements Writer, Closer {
	int length;
}

struct List<T> implements Iterator<T>, mod.Sized {
	T head;
}

type Meters int implements Printable;

enum Color implements Printable {
	Red,
	Green
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	names := func(refs []ast.TypeRef) string {
		parts := make([]string, len(refs))
		for i := range refs {
			parts[i] = refs[i].String()
		}
		return strings.Join(parts, ", ")
	}
	structs := collect[*ast.StructDecl](program)
	if names(structs[0].Implements) != "Writer, Closer" || names(structs[1].Implements) != "Iterator<T>, mod.Sized" {
		t.Errorf("structs: %q %q", names(structs[0].Implements), names(structs[1].Implements))
	}
	types := collect[*ast.TypeDecl](program)
	if names(types[0].Implements) != "Printable" || names(types[1].Implements) != "Printable" {
		t.Errorf("types: %q %q", names(types[0].Implements), names(types[1].Implements))
	}
}

func TestConstrainedTypeParameters(t *testing.T) {
	program, errors := parse(t, `func log<Writer W>(W w) {}
func sum<Number T, const int N>([N]T items) T { return items[0]; }
func scale<(int | float) T, U>(T x) T { return x; }
func put<mod.Writer W, Iterator<int> I>(W w, I it) {}
struct Pipe<Reader R, Writer W> { R source; W target; }
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}
	funcs := collect[*ast.FuncDecl](program)
	type want struct{ name, constraint string }
	check := func(params []ast.TypeParam, wants []want) {
		t.Helper()
		if len(params) != len(wants) {
			t.Fatalf("%d params, want %d", len(params), len(wants))
		}
		for i, w := range wants {
			got := ""
			if params[i].Constraint != nil {
				got = params[i].Constraint.String()
			}
			if params[i].Name != w.name || got != w.constraint {
				t.Errorf("param %d: %s bound by %q, want %s bound by %q", i, params[i].Name, got, w.name, w.constraint)
			}
		}
	}
	check(funcs[0].TypeParams, []want{{"W", "Writer"}})
	check(funcs[1].TypeParams, []want{{"T", "Number"}, {"N", ""}})
	check(funcs[2].TypeParams, []want{{"T", "int | float"}, {"U", ""}})
	check(funcs[3].TypeParams, []want{{"W", "mod.Writer"}, {"I", "Iterator<int>"}})
	check(collect[*ast.StructDecl](program)[0].TypeParams, []want{{"R", "Reader"}, {"W", "Writer"}})
}

func TestInterfaceErrors(t *testing.T) {
	cases := map[string]string{
		"interface X { }\n":                        "is empty",
		"interface X { A & B; C | D }\n":           "one type expression",
		"interface X { func m<T>(T v); }\n":        "cannot take type parameters",
		"interface X implements Y { func m(); }\n": "does not implement another",
		"interface X { func m() }\n":               "expected ';'",
		"func f<T: Writer>(T w) {}\n":              "goes before its name",
		"struct S implements { int a; }\n":         "",
	}
	for src, want := range cases {
		_, errors := parse(t, src)
		if len(errors) == 0 {
			t.Errorf("%q: no error", src)
			continue
		}
		if !slices.ContainsFunc(errors, func(e string) bool { return strings.Contains(e, want) }) {
			t.Errorf("%q: errors %v, want one with %q", src, errors, want)
		}
	}
}

func TestInterfaceMethodMissingSemicolon(t *testing.T) {
	_, errors := parse(t, "interface AST { func node() }\n")
	if len(errors) == 0 || !strings.Contains(errors[0], "expected ';'") {
		t.Fatalf("errors: %v", errors)
	}
}
