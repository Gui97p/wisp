package analyser

import (
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/arch"
	"github.com/Gui97p/wisp/internal/ast"
)

type NativeRefKind int

const (
	RefRegister NativeRefKind = iota
	RefNumber
	RefVariable
	RefConstPart
	RefLabel
	RefExtern
	RefFunction
)

type NativeRef struct {
	Kind NativeRefKind
	Reg  arch.Register
	Text string
	Sym  *Symbol
	Path []string
	Type Type
}

type NativeTermInfo struct {
	Negative bool
	Scale    string
	Ref      NativeRef
}

type NativeOperandInfo struct {
	Ref   NativeRef
	Addr  bool
	Mem   bool
	Size  string
	Rel   bool
	Terms []NativeTermInfo
}

type NativeInstrInfo struct {
	Operands []NativeOperandInfo
}

type NativeInfo struct {
	Instrs    []NativeInstrInfo
	Registers []string
	Stack     bool
	Vars      []*Symbol
}

var widthChecked = map[string]bool{
	"mov": true, "add": true, "sub": true, "cmp": true, "and": true, "or": true,
	"xor": true, "test": true, "adc": true, "sbb": true, "xchg": true,
}

type nativeBlock struct {
	a       *Analyser
	info    *NativeInfo
	labels  map[string]bool
	externs map[string]bool
	touched []string
	seen    map[*Symbol]bool
}

func (a *Analyser) checkNativeStmt(s *ast.NativeStmt) {
	b := &nativeBlock{
		a: a, info: &NativeInfo{},
		labels: make(map[string]bool), externs: make(map[string]bool),
		seen: make(map[*Symbol]bool),
	}

	for _, in := range s.Instrs {
		if in.Label != "" {
			b.declareLabel(in)
		}
		if strings.EqualFold(in.Mnemonic, "extern") {
			for _, op := range in.Operands {
				if op.Kind != ast.NativeWord || len(op.Path) != 1 {
					a.errorRange(op.At, "extern takes the names of assembly symbols")
					continue
				}
				b.externs[op.Path[0]] = true
			}
		}
	}

	for _, in := range s.Instrs {
		b.info.Instrs = append(b.info.Instrs, b.instr(in))
	}

	b.info.Registers = arch.Canonical(b.touched)
	a.info.Natives[s] = b.info
}

func (b *nativeBlock) declareLabel(in ast.NativeInstr) {
	if _, isRegister := arch.LookupRegister(in.Label); isRegister {
		b.a.errorRange(in.At, "%s is a register name, so it cannot be a label", in.Label)
		return
	}
	if b.labels[in.Label] {
		b.a.errorRange(in.At, "the label %s is declared twice in this native block", in.Label)
		return
	}
	b.labels[in.Label] = true
}

func (b *nativeBlock) instr(in ast.NativeInstr) NativeInstrInfo {
	var out NativeInstrInfo
	if in.Mnemonic == "" {
		return out
	}
	if strings.EqualFold(in.Mnemonic, "extern") {
		out.Operands = make([]NativeOperandInfo, len(in.Operands))
		return out
	}

	for _, op := range in.Operands {
		info := b.operand(op)
		if info.Addr && !strings.EqualFold(in.Mnemonic, "lea") {
			b.a.errorRange(op.At, "%s stands for the memory of the variable, and only lea turns that into an address: lea rax, %s", op.String(), op.String())
		}
		if info.Ref.Kind == RefFunction && !info.Mem {
			switch strings.ToLower(in.Mnemonic) {
			case "call", "jmp", "lea":
			default:
				b.a.errorRange(op.At, "the name of a function is an address that only call, jmp and lea take: lea rax, %s", op.String())
			}
		}
		if info.Ref.Kind == RefVariable && !info.Addr && !info.Mem && nativeWidth(info.Ref.Type) == 0 {
			b.a.errorRange(op.At, "%s has type %s, which does not fit in one operand: name a field of it, or use &%s with lea", op.String(), info.Ref.Type.String(), op.String())
		}
		out.Operands = append(out.Operands, info)
	}

	effects := arch.InstrEffects(in.Prefixes, in.Mnemonic, len(in.Operands))
	b.touched = append(b.touched, effects.Touches...)
	if effects.Stack {
		b.info.Stack = true
	}

	b.checkWidths(in, out)
	return out
}

func (b *nativeBlock) operand(op ast.NativeOperand) NativeOperandInfo {
	switch op.Kind {
	case ast.NativeNumber:
		return NativeOperandInfo{Ref: NativeRef{Kind: RefNumber, Text: op.Number}}
	case ast.NativeWord:
		return NativeOperandInfo{Ref: b.word(op.Path, op.At, false)}
	case ast.NativeAddr:
		ref := b.word(op.Path, op.At, false)
		if ref.Kind != RefVariable {
			b.a.errorRange(op.At, "& needs a variable: %s is not one", strings.Join(op.Path, "."))
		}
		return NativeOperandInfo{Ref: ref, Addr: true}
	}

	out := NativeOperandInfo{Mem: true, Size: op.Size, Rel: op.Rel}
	for _, t := range op.Terms {
		term := NativeTermInfo{Negative: t.Negative, Scale: t.Scale}
		if t.Number != "" {
			term.Ref = NativeRef{Kind: RefNumber, Text: t.Number}
		} else {
			term.Ref = b.word(t.Path, t.At, true)
		}
		out.Terms = append(out.Terms, term)
	}
	return out
}

func (b *nativeBlock) word(path []string, at ast.Position, inMemory bool) NativeRef {
	a := b.a
	name := path[0]

	if reg, ok := arch.LookupRegister(name); ok {
		if len(path) > 1 {
			a.errorRange(at, "%s is a register: it has no field %s", name, path[1])
			return NativeRef{Kind: RefRegister, Reg: reg}
		}
		if sym, shadowed := a.scope.Resolve(name); shadowed && sym.Kind == VAR {
			a.errors.Warn(a.currentFile, at.Line, at.Col, at.EndLine, at.EndCol,
				"%s is a register in a native block, not the variable %s: rename the variable or copy it before the block", name, name)
		}
		if reg.Class == arch.General || reg.Class == arch.Vector {
			b.touched = append(b.touched, reg.Canonical)
		}
		if reg.Canonical == "rsp" || reg.Canonical == "rbp" {
			b.info.Stack = true
		}
		return NativeRef{Kind: RefRegister, Reg: reg}
	}

	if len(path) == 1 {
		if b.labels[name] {
			return NativeRef{Kind: RefLabel, Text: name}
		}
		if b.externs[name] {
			return NativeRef{Kind: RefExtern, Text: name}
		}
		if arch.IsSizeKeyword(name) {
			a.errorRange(at, "%s is a size keyword: it goes before a memory operand, as in %s [rax]", name, name)
			return NativeRef{Kind: RefNumber, Text: "0"}
		}
	}

	sym, rest := NativeConstantRef(a.scope, path)
	if root, ok := a.scope.Resolve(path[0]); ok {
		root.Used = true
	}
	if sym == nil {
		a.errorRange(at, "unknown name %s in the native block: it accepts registers, labels, extern symbols, variables, constants and functions", strings.Join(path, "."))
		return NativeRef{Kind: RefNumber, Text: "0"}
	}
	sym.Used = true

	switch sym.Kind {
	case CONST:
		return b.constant(sym, rest, at)
	case VAR, PARAM:
		return b.variable(sym, rest, at, inMemory)
	case FUNC:
		if len(rest) > 0 {
			a.errorRange(at, "the function %s has no part %s", sym.Name, rest[0])
		}
		return NativeRef{Kind: RefFunction, Sym: sym}
	}
	a.errorRange(at, "%s cannot be used in a native block: only variables, constants and functions can", strings.Join(path, "."))
	return NativeRef{Kind: RefNumber, Text: "0"}
}

func (b *nativeBlock) constant(sym *Symbol, rest []string, at ast.Position) NativeRef {
	a := b.a
	if sym.Const != nil && sym.Const.Kind == ConstString {
		if len(rest) != 1 || (rest[0] != "ptr" && rest[0] != "len") {
			a.errorRange(at, "the string constant %s has the parts ptr and len: write %s.ptr or %s.len", sym.Name, sym.Name, sym.Name)
		}
		return NativeRef{Kind: RefConstPart, Sym: sym, Path: rest}
	}
	if len(rest) != 0 {
		a.errorRange(at, "the constant %s has no part %s", sym.Name, rest[0])
	}
	text, ok := NativeConstant(sym)
	if !ok {
		a.errorRange(at, "the constant %s has no value that fits in an instruction", sym.Name)
		text = "0"
	}
	return NativeRef{Kind: RefNumber, Text: text}
}

func (b *nativeBlock) variable(sym *Symbol, rest []string, at ast.Position, inMemory bool) NativeRef {
	a := b.a
	full := sym.Name
	if len(rest) > 0 {
		full += "." + strings.Join(rest, ".")
	}
	if inMemory {
		a.errorRange(at, "the variable %s cannot be used inside [ ]: load it into a register first, as in mov rax, %s", full, full)
	}

	t := sym.Type
	for _, part := range rest {
		next, ok := nativeField(t, part)
		if !ok {
			a.errorRange(at, "%s has no field %s here", full, part)
			break
		}
		t = next
	}

	if !b.seen[sym] {
		b.seen[sym] = true
		b.info.Vars = append(b.info.Vars, sym)
	}
	return NativeRef{Kind: RefVariable, Sym: sym, Path: rest, Type: t}
}

func nativeField(t Type, part string) (Type, bool) {
	switch v := t.(type) {
	case NamedType:
		return nativeField(v.Underlying, part)
	case *StructType:
		f, ok := v.Fields[part]
		return f, ok
	case SpanType:
		if part == "ptr" || part == "len" {
			return PrimitiveType{Name: "int"}, true
		}
	case PrimitiveType:
		if v.Name == "string" && (part == "ptr" || part == "len") {
			return PrimitiveType{Name: "int"}, true
		}
	}
	return nil, false
}

func nativeWidth(t Type) int {
	switch v := t.(type) {
	case NamedType:
		if _, isFunc := v.Underlying.(*FuncType); isFunc {
			return 8
		}
		return nativeWidth(v.Underlying)
	case PointerType, *FuncType:
		return 8
	case PrimitiveType:
		switch v.Name {
		case "int", "uint", "int64", "uint64", "float", "float64":
			return 8
		case "int32", "uint32", "float32":
			return 4
		case "int16", "uint16":
			return 2
		case "int8", "uint8", "byte", "bool":
			return 1
		}
	}
	return 0
}

func (b *nativeBlock) checkWidths(in ast.NativeInstr, info NativeInstrInfo) {
	if !widthChecked[strings.ToLower(in.Mnemonic)] || len(info.Operands) != 2 {
		return
	}
	for i := range info.Operands {
		reg, other := info.Operands[i], info.Operands[1-i]
		if reg.Mem || reg.Ref.Kind != RefRegister || reg.Ref.Reg.Class != arch.General {
			continue
		}
		if other.Mem || other.Addr || other.Ref.Kind != RefVariable {
			continue
		}
		width := nativeWidth(other.Ref.Type)
		if width == 0 {
			continue
		}
		if width*8 != reg.Ref.Reg.Width {
			at := in.Operands[1-i].At
			b.a.errors.Warn(b.a.currentFile, at.Line, at.Col, at.EndLine, at.EndCol,
				"%s is %d bytes wide but %s is %d bits: the instruction mixes widths", in.Operands[1-i].String(), width, reg.Ref.Reg.Name, reg.Ref.Reg.Width)
		}
	}
}

func NativeConstantRef(scope *Scope, parts []string) (*Symbol, []string) {
	sym, ok := scope.Resolve(parts[0])
	if !ok {
		return nil, nil
	}
	if mt, isModule := sym.Type.(*ModuleType); isModule {
		if len(parts) < 2 {
			return nil, nil
		}
		exported, found := mt.Exports[parts[1]]
		if !found {
			return nil, nil
		}
		return exported, parts[2:]
	}
	return sym, parts[1:]
}

func NativeConstant(sym *Symbol) (string, bool) {
	if sym == nil || sym.Kind != CONST || sym.Const == nil {
		return "", false
	}
	switch sym.Const.Kind {
	case ConstInt:
		return sym.Const.Int.String(), true
	case ConstBool:
		if sym.Const.Bool {
			return "1", true
		}
		return "0", true
	}
	return "", false
}

func (i *NativeInfo) SortedVars() []*Symbol {
	vars := append([]*Symbol(nil), i.Vars...)
	sort.Slice(vars, func(a, b int) bool { return vars[a].Name < vars[b].Name })
	return vars
}
