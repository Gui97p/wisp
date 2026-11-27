package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

type TagSpec struct {
	Name    string
	Usage   string
	Doc     string
	Targets []string
	Args    int
}

var Tags = []TagSpec{
	{
		Name: "naked", Usage: "#naked",
		Doc:     "A native function with no frame: the compiler emits the block and nothing else. The parameters stay in the registers of the calling convention (rdi, rsi, rdx, rcx, r8, r9, and xmm0 to xmm7 for floats), the results are written as types only, and the block ends in `ret` or `jmp`.",
		Targets: []string{"native function"},
	},
	{
		Name: "packed", Usage: "#packed",
		Doc:     "A struct without padding: the fields follow each other and the alignment is 1.",
		Targets: []string{"struct"},
	},
	{
		Name: "deprecated", Usage: "#deprecated(\"message\")",
		Doc:     "Every use of the declaration gets a warning with the message. The message is optional.",
		Targets: []string{"function", "native function", "struct", "enum", "enum with data", "type", "constant", "variable"},
		Args:    1,
	},
	{
		Name: "must_use", Usage: "#must_use(\"message\")",
		Doc:     "A function whose result matters, so a call that throws the result away gets a warning, with the message if there is one. It is for functions that do nothing but compute their result; to ignore a result on purpose, keep it in a variable whose name starts with an underscore.",
		Targets: []string{"function", "method", "native function"},
		Args:    1,
	},
	{
		Name: "test", Usage: "#test(\"name\")",
		Doc:     "A test that `wisp test` runs, in a process of its own. It takes no parameters and gives no result, or only an error (`!`). It passes when it returns, and fails when it returns an error or stops the program, as a panic does. The name between quotes is optional and is what the report shows. The function is left out of every other build.",
		Targets: []string{"function"},
		Args:    1,
	},
	{
		Name: "expect_abort", Usage: "#expect_abort(\"text\")",
		Doc:     "A test that must stop the program, as a panic or a failed bounds check does, instead of returning. It passes when the program stops and what it printed contains the text, and fails when it returns. Without a text any stop passes. It goes with #test, on the same function.",
		Targets: []string{"function"},
		Args:    1,
	},
	{
		Name: "symbol", Usage: "#symbol(\"name\")",
		Doc:     "The name the linker sees. On an extern function it is the C function that is called, and the Wisp name is the one written in the declaration. On a public function it is the name C and other languages use to call it, which does not depend on the path of the module; a library with `shared = true` shows only these names.",
		Targets: []string{"function", "extern function"},
		Args:    1,
	},
	{
		Name: "fails", Usage: "#fails(result < 0, errors.Kind.Invalid)",
		Doc:     "Turns the result of an extern function into an error. The declaration says `!T`, and the first argument says when the call failed, with the name `result` for what C returned: `#fails(result < 0, errors.Kind.Invalid)`. For a function that returns a pointer, `#fails(null, errors.Kind.NotFound)` makes a null result the failure, and the result is then a pointer that is never null. The second argument is the kind of the error, or an expression that gives an `Error`, such as a call to a function of yours that chooses the error from `result`: `#fails(result != 0, statusError(result))`. The function is called through a Wisp function that makes the test, so C never sees the error.",
		Targets: []string{"extern function"},
		Args:    2,
	},
	{
		Name: "format", Usage: "#format",
		Doc:     "A function that takes a format string and the values to write in it: its last parameters are `string format, ...args`. A call whose format is a string literal may write expressions in it, `{sum}`, `{items[i]:X}`, `{a + b:>8}`: the compiler takes each one out of the text, leaves `{}` or `{:spec}` in its place and passes the expression as the next argument. `{}` and `{:spec}` still take the arguments written after the string, in order, and `{{` and `}}` write a brace. An if that gives a value works too, `{if ok => 1 : 2}`, and a second `:` after it starts the spec: `{if ok => 1 : 2:>4}`.",
		Targets: []string{"function", "method"},
	},
	{
		Name: "dynamic", Usage: "#dynamic",
		Doc:     "An interface whose values can be told apart at run time. A value that is passed as an untyped variadic argument (`...args`) and whose type declares `implements` this interface can be picked out with `Interface v =>` in a `match type`, and `v` is then the interface value. The interface has to be declared with #dynamic for that: without it an arm cannot name it. It takes nothing for the types that do not implement it, and cannot be generic.",
		Targets: []string{"interface"},
	},
	{
		Name: "caller", Usage: "#caller",
		Doc:     "A function that wants to know where it was called from. Its first parameter has the type `Here`, and a call does not write it: the compiler fills it with the file, the line and the code of the call, as an error with no kind. `at.file`, `at.line` and `at.block` can be read, and `at` can be the base of an error made at run time.",
		Targets: []string{"function", "method"},
	},
	{
		Name: "noreturn", Usage: "#noreturn",
		Doc:     "A function that never returns, such as one that ends the program. Its body must not return, and the code after a call to it is unreachable.",
		Targets: []string{"function", "method", "native function"},
	},
}

func tagSpecOf(name string) (TagSpec, bool) {
	for _, spec := range Tags {
		if spec.Name == name {
			return spec, true
		}
	}
	return TagSpec{}, false
}

func tagNames() string {
	names := make([]string, len(Tags))
	for i, spec := range Tags {
		names[i] = "#" + spec.Name
	}
	return strings.Join(names, ", ")
}

func declTarget(decl ast.Declaration) string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		switch {
		case d.Extern:
			return "extern function"
		case d.Native:
			return "native function"
		case d.Receiver != nil:
			return "method"
		}
		return "function"
	case *ast.StructDecl:
		return "struct"
	case *ast.InterfaceDecl:
		return "interface"
	case *ast.TypeDecl:
		switch {
		case d.Data:
			return "enum with data"
		case d.IsEnum:
			return "enum"
		}
		return "type"
	case *ast.ConstDecl:
		return "constant"
	case *ast.VarDecl:
		return "variable"
	}
	return ""
}

func (a *Analyser) checkTags() {
	for _, d := range a.program.Declarations {
		if _, ok := d.(ast.Tags); !ok {
			continue
		}
		a.currentFile = a.declFiles[d]
		target := declTarget(d)
		seen := map[string]bool{}

		var tags []ast.Tag
		switch decl := d.(type) {
		case *ast.FuncDecl:
			tags = decl.Tags
		case *ast.StructDecl:
			tags = decl.Tags
		case *ast.InterfaceDecl:
			tags = decl.Tags
		case *ast.TypeDecl:
			tags = decl.Tags
		case *ast.ConstDecl:
			tags = decl.Tags
		case *ast.VarDecl:
			tags = decl.Tags
		}

		for i := range tags {
			tag := &tags[i]
			spec, known := tagSpecOf(tag.Name)
			if !known {
				diag := a.errorRange(tag.At, "unknown tag #%s", tag.Name)
				if tag.Name == "target" {
					diag.WithHelp("#target is a directive: it goes at the top of the file")
				} else {
					diag.WithHelp("the known tags are %s", tagNames())
				}
				continue
			}
			if seen[tag.Name] {
				a.errorRange(tag.At, "#%s is given twice", tag.Name)
				continue
			}
			seen[tag.Name] = true

			if !contains(spec.Targets, target) {
				a.errorRange(tag.At, "#%s does not apply to a %s", tag.Name, target).
					WithHelp("it applies to: %s", strings.Join(spec.Targets, ", "))
				continue
			}
			if len(tag.Args) > spec.Args {
				if spec.Args == 0 {
					a.errorRange(tag.At, "#%s takes no arguments", tag.Name)
				} else {
					a.errorRange(tag.At, "#%s takes at most %d argument", tag.Name, spec.Args)
				}
				continue
			}
			a.checkTagArgs(spec, tag, d)
		}
	}
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func (a *Analyser) checkTagArgs(spec TagSpec, tag *ast.Tag, decl ast.Declaration) {
	switch spec.Name {
	case "expect_abort":
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.TagNamed("test") == nil {
			a.errorRange(tag.At, "#expect_abort belongs to a test: mark %s with #test too", fd.Name)
		}
		if len(tag.Args) == 1 {
			if _, isText := tag.Args[0].(*ast.StringLiteral); !isText {
				a.errorf(tag.Args[0], "the text of #expect_abort is a text between quotes")
			}
		}
	case "test":
		if len(tag.Args) == 1 {
			if text, isText := tag.Args[0].(*ast.StringLiteral); !isText {
				a.errorf(tag.Args[0], "the name of a test is a text between quotes")
			} else if strings.TrimSpace(text.Value) == "" {
				a.errorf(tag.Args[0], "the name of a test cannot be empty")
			}
		}
		if fd, ok := decl.(*ast.FuncDecl); ok {
			switch {
			case len(fd.Params) > 0:
				a.errorRange(tag.At, "a test takes no parameters, and %s takes %d", fd.Name, len(fd.Params))
			case len(fd.ReturnTypes) > 0:
				a.errorRange(tag.At, "a test gives no result, and %s gives one: it passes or fails", fd.Name)
			case fd.Inline:
				a.errorRange(tag.At, "a test cannot be inline")
			case isGenericDecl(fd):
				a.errorRange(tag.At, "a test cannot be generic: %s would have no type arguments to run with", fd.Name)
			}
		}
	case "fails":
		if fd, ok := decl.(*ast.FuncDecl); ok {
			switch {
			case len(tag.Args) != 2:
				a.errorRange(tag.At, "#fails needs the condition and the error kind: #fails(result < 0, errors.Kind.Invalid)")
			case !fd.Fallible || len(fd.ReturnTypes) != 1:
				a.errorRange(tag.At, "#fails tests the result of the call: declare %s with a result that can fail, as in !c_int", fd.Name)
			case fd.CVariadic:
				a.errorRange(tag.At, "#fails cannot be used on a C variadic function yet")
			default:
				if _, onNull := tag.Args[0].(*ast.NullLiteral); onNull && !pointerRef(fd.ReturnTypes[0]) {
					a.errorRange(tag.At, "#fails(null, ...) is for a function that returns a pointer, and %s does not", fd.Name)
				}
			}
		}
	case "dynamic":
		if decl, ok := decl.(*ast.InterfaceDecl); ok && len(decl.TypeParams) > 0 {
			a.errorRange(tag.At, "a generic interface cannot be #dynamic")
		}
	case "caller":
		if fd, ok := decl.(*ast.FuncDecl); ok && !isHereParam(fd) {
			a.errorRange(tag.At, "#caller needs the first parameter of %s to be a `Here`", fd.Name)
		}
	case "format":
		if fd, ok := decl.(*ast.FuncDecl); ok && formatParam(fd) < 0 {
			a.errorRange(tag.At, "#format needs the parameters `string format, ...args` at the end of %s", fd.Name)
		}
	case "symbol":
		if len(tag.Args) == 0 {
			a.errorRange(tag.At, "#symbol needs the name: #symbol(\"name\")")
		} else if text, isText := tag.Args[0].(*ast.StringLiteral); !isText || strings.TrimSpace(text.Value) == "" {
			a.errorf(tag.Args[0], "the name of #symbol is a text between quotes, and cannot be empty")
		}
	case "deprecated", "must_use":
		if len(tag.Args) == 1 {
			if _, isText := tag.Args[0].(*ast.StringLiteral); !isText {
				a.errorf(tag.Args[0], "the message of #%s is a text between quotes", spec.Name)
			}
		}
		if fd, ok := decl.(*ast.FuncDecl); ok && spec.Name == "must_use" && len(fd.ReturnTypes) == 0 {
			a.errorRange(tag.At, "#must_use is for a function that gives a result, and %s gives none", fd.Name)
		}
	}
}

func (a *Analyser) applyTagSymbols() {
	for _, d := range a.program.Declarations {
		var tags []ast.Tag
		var names []string
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Receiver != nil {
				continue
			}
			tags, names = decl.Tags, []string{decl.Name}
		case *ast.StructDecl:
			tags, names = decl.Tags, []string{decl.Name}
		case *ast.TypeDecl:
			tags, names = decl.Tags, []string{decl.Name}
		case *ast.ConstDecl:
			tags = decl.Tags
			for _, v := range decl.Vars {
				names = append(names, v.Name)
			}
		case *ast.VarDecl:
			tags = decl.Tags
			for _, v := range decl.Vars {
				names = append(names, v.Name)
			}
		default:
			continue
		}
		for i := range tags {
			tag := &tags[i]
			for _, name := range names {
				sym, found := a.scope.symbols[name]
				if !found {
					continue
				}
				switch tag.Name {
				case "deprecated":
					message := ""
					if len(tag.Args) == 1 {
						if text, ok := tag.Args[0].(*ast.StringLiteral); ok {
							message = text.Value
						}
					}
					sym.Deprecated = &message
				}
			}
		}
	}
}

func (a *Analyser) deprecationWarning(sym *Symbol) (string, bool) {
	if sym == nil || sym.Deprecated == nil {
		return "", false
	}
	if *sym.Deprecated == "" {
		return fmt.Sprintf("%s is deprecated", sym.Name), true
	}
	return fmt.Sprintf("%s is deprecated: %s", sym.Name, *sym.Deprecated), true
}

func (a *Analyser) noteUse(node ast.Node, sym *Symbol) {
	if message, ok := a.deprecationWarning(sym); ok {
		a.warnf(node, "%s", message)
	}
}

func (a *Analyser) noteUseRef(ref ast.TypeRef, sym *Symbol) {
	if message, ok := a.deprecationWarning(sym); ok && ref.Line > 0 {
		a.errors.Warn(a.currentFile, ref.Line, ref.Col, ref.EndLine, ref.EndCol, "%s", message)
	}
}

func (a *Analyser) calleeFunc(call *ast.CallExpr) *FuncType {
	if mc, ok := a.info.MethodCalls[call]; ok && mc.Func != nil {
		return mc.Func
	}
	var sym *Symbol
	switch name := call.Name.(type) {
	case *ast.IdentLiteral:
		sym = a.info.Idents[name]
		if sym == nil {
			sym, _ = a.scope.Resolve(name.Value)
		}
	case *ast.MemberExpr:
		if module, ok := name.Object.(*ast.IdentLiteral); ok {
			if owner, found := a.scope.Resolve(module.Value); found {
				if mt, isModule := owner.Type.(*ModuleType); isModule {
					sym = mt.Exports[name.Field]
				}
			}
		}
	}
	if sym == nil {
		return nil
	}
	ft, _ := sym.Type.(*FuncType)
	return ft
}

func (a *Analyser) calleeNoReturn(call *ast.CallExpr) bool {
	ft := a.calleeFunc(call)
	return ft != nil && ft.Decl != nil && ft.Decl.TagNamed("noreturn") != nil
}

func (a *Analyser) discardedCall(expr ast.Expression) *ast.CallExpr {
	for {
		switch e := expr.(type) {
		case *ast.CallExpr:
			return e
		case *ast.PropagateExpr:
			expr = e.Value
		case *ast.CoalesceExpr:
			expr = e.Left
		default:
			return nil
		}
	}
}

func (a *Analyser) warnMustUse(stmt *ast.ExpressionStmt) {
	call := a.discardedCall(stmt.Expr)
	if call == nil {
		return
	}
	ft := a.calleeFunc(call)
	if ft == nil || ft.Decl == nil {
		return
	}
	tag := ft.Decl.TagNamed("must_use")
	if tag == nil {
		return
	}
	name := ft.label()
	if path, ok := fieldPath(call.Name); ok {
		name = joinPath(path)
	}
	d := a.warnf(call, "the result of %s is not used", name)
	if len(tag.Args) == 1 {
		if text, ok := tag.Args[0].(*ast.StringLiteral); ok && text.Value != "" {
			d.Note(a.currentFile, tag.At.Line, tag.At.Col, tag.At.EndLine, tag.At.EndCol, "%s", text.Value)
		}
	}
	d.WithHelp("use it, or keep it in a variable whose name starts with an underscore")
}

func (a *Analyser) callNeverReturns(expr ast.Expression) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	return a.isPanicCall(expr) || a.calleeNoReturn(call)
}

func nakedRegisters(ft *FuncType, fd *ast.FuncDecl) map[string]string {
	ints := []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
	floats := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7"}
	out := map[string]string{}
	for i, p := range fd.Params {
		if i >= len(ft.Params) {
			break
		}
		var regs []string
		take := func(pool *[]string) string {
			if len(*pool) == 0 {
				return "the stack"
			}
			r := (*pool)[0]
			*pool = (*pool)[1:]
			return r
		}
		switch t := underlying(ft.Params[i]).(type) {
		case PrimitiveType:
			switch {
			case isFloatName(t.Name):
				regs = append(regs, take(&floats))
			case t.Name == "string":
				regs = append(regs, take(&ints), take(&ints))
			default:
				regs = append(regs, take(&ints))
			}
		case SpanType:
			regs = append(regs, take(&ints), take(&ints))
		default:
			regs = append(regs, take(&ints))
		}
		out[p.Name] = strings.Join(regs, " and ")
	}
	return out
}

func (a *Analyser) checkNaked(fd *ast.FuncDecl) {
	var native *ast.NativeStmt
	for _, stmt := range fd.Body.Statements {
		if n, ok := stmt.(*ast.NativeStmt); ok {
			native = n
		}
	}
	if native == nil || len(native.Instrs) == 0 {
		return
	}
	last := native.Instrs[len(native.Instrs)-1]
	mnemonic := strings.ToLower(last.Mnemonic)
	if mnemonic != "ret" && mnemonic != "jmp" {
		a.errorRange(native.BodyAt, "a naked function must end with ret or jmp: nothing else returns for it")
	}
}

func (a *Analyser) checkNoReturn(fd *ast.FuncDecl) {
	returned := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncLiteral:
			return false
		case *ast.ReturnStmt:
			if !returned {
				a.errorf(v, "a #noreturn function cannot return")
				returned = true
			}
		}
		return true
	})
	if !returned && !a.blockTerminates(fd.Body) {
		a.errorAtEnd(fd.Body, "the function %s is #noreturn but it can end", fd.Name).
			WithHelp("end it with panic, a call to another #noreturn function, or a loop with no end")
	}
}

func pointerRef(ref ast.TypeRef) bool {
	return len(ref.Layers) > 0 && ref.Layers[0].Pointer
}
