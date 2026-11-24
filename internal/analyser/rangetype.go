package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

const rangeFile = "<builtin range>"

const rangeCallName = "__wisp_range"

const rangeSource = `
/// The values from start to end, both included, counting by one.
/// Write it as start..end, and give it another step with step.
public struct Range<T> {
	public T start;
	public T end;
	internal int by;
}

/// Walks a range once: what for gives when it takes a range.
public struct RangeIter<T> {
	internal T cur;
	internal T end;
	internal int by;
	internal bool done;
}

public func makeRange<T>(T start, T end) Range<T> {
	return Range<T>{start: start, end: end};
}

/// The same range counting by n: negative to count down, never zero.
public func (Range<T> r) step(int n) Range<T> {
	if n == 0 {
		panic("the step of a range cannot be zero");
	}
	return Range<T>{start: r.start, end: r.end, by: n};
}

func (Range<T> r) stride() int {
	if r.by == 0 {
		return 1;
	}
	return r.by;
}

/// A fresh walk of the range, which for uses to visit its values.
public func (Range<T> r) iter() RangeIter<T> {
	return RangeIter<T>{cur: r.start, end: r.end, by: r.stride()};
}

/// Whether the value is one the range visits, counting its step.
public func (Range<T> r) contains(T v) bool {
	int by = r.stride();
	if by > 0 {
		return v >= r.start && v <= r.end && (truncate(uint64, v) -% truncate(uint64, r.start)) % uint64(by) == 0;
	}
	return v <= r.start && v >= r.end && (truncate(uint64, r.start) -% truncate(uint64, v)) % uint64(-by) == 0;
}

/// How many values the range visits.
public func (Range<T> r) count() int {
	int by = r.stride();
	if by > 0 {
		if r.end < r.start {
			return 0;
		}
		return int((truncate(uint64, r.end) -% truncate(uint64, r.start)) / uint64(by)) + 1;
	}
	if r.end > r.start {
		return 0;
	}
	return int((truncate(uint64, r.start) -% truncate(uint64, r.end)) / uint64(-by)) + 1;
}

/// The next value of the walk, or null when it is over.
public func (*RangeIter<T> it) next() ?T {
	if it.done {
		return null;
	}
	T v = it.cur;
	if it.by > 0 {
		if v > it.end {
			it.done = true;
			return null;
		}
		if truncate(uint64, it.end) -% truncate(uint64, v) < uint64(it.by) {
			it.done = true;
		} else {
			it.cur = truncate(T, truncate(uint64, v) +% uint64(it.by));
		}
		return v;
	}
	if v < it.end {
		it.done = true;
		return null;
	}
	if truncate(uint64, v) -% truncate(uint64, it.end) < uint64(-it.by) {
		it.done = true;
	} else {
		it.cur = truncate(T, truncate(uint64, v) -% uint64(-it.by));
	}
	return v;
}
`

func (a *Analyser) registerRange() {
	program := parser.NewParser(lexer.NewLexer([]byte(rangeSource))).ParseProgram()

	define := func(name string, kind SymbolKind, g *GenericDecl, doc string) {
		symbol := &Symbol{Name: name, Kind: kind, Type: GenericType{Decl: g}, File: rangeFile, Generic: g, Used: true, Doc: doc}
		g.Symbol = symbol
		a.universe.Define(symbol)
	}

	for _, d := range program.Declarations {
		if sd, ok := d.(*ast.StructDecl); ok {
			g := &GenericDecl{Name: sd.Name, File: rangeFile, Scope: a.universe, Params: sd.TypeParams, Struct: sd, Exported: true}
			define(sd.Name, STRUCT, g, sd.Doc)
		}
	}

	for _, d := range program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Receiver == nil {
			if fd.Name == "makeRange" {
				fd.Name = rangeCallName
			}
			g := &GenericDecl{Name: fd.Name, File: rangeFile, Scope: a.universe, Params: fd.TypeParams, Func: fd, Exported: true}
			define(fd.Name, FUNC, g, fd.Doc)
			continue
		}
		symbol, found := a.universe.Resolve(fd.Receiver.Type.Name)
		if found && symbol.Generic != nil {
			symbol.Generic.Methods = append(symbol.Generic.Methods, fd)
		}
	}
}

func rangeElement(t Type) Type {
	st, ok := t.(*StructType)
	if !ok || st.Instance == nil || st.Instance.Decl == nil || st.Instance.Decl.File != rangeFile || st.Instance.Decl.Name != "Range" || len(st.Instance.Args) != 1 {
		return nil
	}
	return st.Instance.Args[0].Type
}

func allUntyped(types []Type) bool {
	for _, t := range types {
		if !isUntyped(t) {
			return false
		}
	}
	return len(types) > 0
}

func fitsUntyped(bound, defaulted Type) bool {
	base := underlyingOf(bound)
	if isInteger(defaulted) {
		return isInteger(base)
	}
	return isFloatType(base) && isFloatType(defaulted)
}

func rangeChain(call *ast.CallExpr) bool {
	if call.Range {
		return true
	}
	member, ok := call.Name.(*ast.MemberExpr)
	if !ok || member.Field != "step" {
		return false
	}
	inner, ok := member.Object.(*ast.CallExpr)
	return ok && inner.Range
}
