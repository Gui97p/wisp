package analyser

import (
	"testing"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

func prim(name string) Type {
	return PrimitiveType{Name: name}
}

func analyse(t *testing.T, src string) (*ast.Program, *Info) {
	t.Helper()

	p := parser.NewParser(lexer.NewLexer([]byte(src)))
	program := p.ParseProgram()
	if p.HasErrors() {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	a := NewAnalyser(program, nil, nil, "test")
	info := a.Analyze()
	if a.HasErrors() {
		t.Fatalf("analysis errors: %v", a.Errors())
	}
	return program, info
}

func TestLosslessWidening(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"int8", "int16", true},
		{"int8", "int64", true},
		{"int16", "int8", false},
		{"uint8", "uint16", true},
		{"uint8", "int16", true},
		{"uint8", "int8", false},
		{"uint32", "int64", true},
		{"uint32", "int32", false},
		{"uint64", "int64", false},
		{"int8", "uint16", false},
		{"int", "int64", true},
		{"int64", "int", true},
		{"uint", "uint64", true},
		{"char", "int", true},
		{"char", "uint8", false},
		{"uint8", "char", false},
		{"float32", "float64", true},
		{"float64", "float32", false},
		{"int32", "float64", false},
		{"bool", "int", false},
		{"int8", "int8", false},
	}

	for _, c := range cases {
		if got := LosslessWidening(prim(c.from), prim(c.to)); got != c.want {
			t.Errorf("LosslessWidening(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestCommonType(t *testing.T) {
	cases := []struct {
		l, r, want string
	}{
		{"int8", "int8", "int8"},
		{"int8", "int16", "int16"},
		{"int64", "int8", "int64"},
		{"uint8", "int8", "int16"},
		{"uint8", "int16", "int16"},
		{"uint16", "int8", "int32"},
		{"uint32", "int32", "int64"},
		{"uint32", "int64", "int64"},
		{"uint64", "int64", ""},
		{"uint64", "int8", ""},
		{"float32", "float64", "float64"},
		{"int32", "float64", ""},
	}

	for _, c := range cases {
		got, ok := commonType(prim(c.l), prim(c.r))
		if c.want == "" {
			if ok {
				t.Errorf("commonType(%s, %s) = %s, want none", c.l, c.r, got)
			}
			continue
		}
		if !ok || got.String() != c.want {
			t.Errorf("commonType(%s, %s) = %v (%v), want %s", c.l, c.r, got, ok, c.want)
		}
	}
}

func TestFitsInteger(t *testing.T) {
	cases := []struct {
		neg    bool
		mag    uint64
		bits   int
		signed bool
		want   bool
	}{
		{false, 127, 8, true, true},
		{false, 128, 8, true, false},
		{true, 128, 8, true, true},
		{true, 129, 8, true, false},
		{false, 255, 8, false, true},
		{false, 256, 8, false, false},
		{true, 1, 8, false, false},
		{true, 0, 8, false, true},
		{false, 9223372036854775807, 64, true, true},
		{false, 9223372036854775808, 64, true, false},
		{true, 9223372036854775808, 64, true, true},
		{false, 18446744073709551615, 64, false, true},
	}

	for _, c := range cases {
		if got := fitsInteger(c.neg, c.mag, c.bits, c.signed); got != c.want {
			t.Errorf("fitsInteger(%v, %d, %d, %v) = %v, want %v", c.neg, c.mag, c.bits, c.signed, got, c.want)
		}
	}
}

func TestNoUntypedLeftAfterAnalysis(t *testing.T) {
	_, info := analyse(t, `
const Limit = 100;

func take(int8 v) int8 {
    return v;
}

func main() {
    let x = 5;
    let f = 2.5;
    int8 a = 1 + 2;
    int16 b = 7;
    int64 c = Limit;
    float32 g = 1;
    int[3] arr = [1, 2, 3];
    let list = [4, 5, 6];
    int8[2] small = [7, 8];
    int r = take(3) + 4 * 5;
    uint8 sh = 1 << 3;
    emitf("%d %f", x, f);
}
`)

	for expr, tp := range info.Types {
		if containsUntyped(tp) {
			t.Errorf("%T still has type %s", expr, tp)
		}
	}
}

func TestImplicitCastsAreInserted(t *testing.T) {
	program, info := analyse(t, `
func main() {
    int8 a = 100;
    int16 b = a;
    int64 total = a + b;
}
`)

	var fn *ast.FuncDecl
	for _, d := range program.Declarations {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name == "main" {
			fn = f
		}
	}
	if fn == nil {
		t.Fatal("main not found")
	}

	stmts := fn.Body.Statements
	assign := stmts[1].(*ast.VarStmt)
	cast, ok := assign.Values[0].(*ast.CastExpr)
	if !ok || !cast.Implicit {
		t.Fatalf("int16 b = a should wrap the value in an implicit cast, got %T", assign.Values[0])
	}
	if got := info.Types[cast].String(); got != "int16" {
		t.Errorf("cast target = %s, want int16", got)
	}

	sum := stmts[2].(*ast.VarStmt)
	bin, ok := sum.Values[0].(*ast.CastExpr)
	if !ok {
		t.Fatalf("int64 total = a + b should convert the int16 result, got %T", sum.Values[0])
	}
	inner := bin.Value.(*ast.BinaryExpr)
	if got := info.Types[inner].String(); got != "int16" {
		t.Errorf("a + b type = %s, want int16", got)
	}
	if _, ok := inner.Left.(*ast.CastExpr); !ok {
		t.Errorf("left operand a (int8) should be widened to int16, got %T", inner.Left)
	}
}

func TestLetDefaultsToConcreteTypes(t *testing.T) {
	program, info := analyse(t, `
func main() {
    let x = 5;
    let f = 2.5;
}
`)

	fn := program.Declarations[0].(*ast.FuncDecl)
	x := fn.Body.Statements[0].(*ast.VarStmt)
	f := fn.Body.Statements[1].(*ast.VarStmt)
	if got := info.VarTypes[x][0].String(); got != "int" {
		t.Errorf("let x = 5 has type %s, want int", got)
	}
	if got := info.VarTypes[f][0].String(); got != "float64" {
		t.Errorf("let f = 2.5 has type %s, want float64", got)
	}
}
