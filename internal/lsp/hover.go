package lsp

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type builtinHelp struct {
	signature string
	doc       string
	params    []string
	returns   string
}

var builtinHelps = map[string]builtinHelp{
	"emit":         {"func emit(string text)", "Writes a string and a newline to standard output.", []string{"string text"}, ""},
	"len":          {"func len(x) int", "The length of an array, slice, string or variadic parameter.", []string{"x"}, "int"},
	"panic":        {"func panic(string message)", "Stops the program with the message and the source position, and exits with code 1.", []string{"string message"}, ""},
	"sizeof":       {"func sizeof(x) int", "The size in bytes of a type or of an expression. It is a constant known at compile time, so it can size an array or feed a `staticAssert`.", []string{"x"}, "int"},
	"alignof":      {"func alignof(Type) int", "The alignment in bytes of a type, known at compile time.", []string{"Type"}, "int"},
	"offsetof":     {"func offsetof(Type, field) int", "The offset in bytes of a field inside its struct, known at compile time. A path such as `offsetof(Outer, inner.x)` follows nested structs.", []string{"Type", "field"}, "int"},
	"staticAssert": {"func staticAssert(condition, message)", "Stops the compilation with the message when the condition is false. The condition and the message must be known at compile time. It can be written at the top of a file or inside a function.", []string{"condition", "message"}, ""},
	"truncate": {"func truncate(Type, value) Type", "Converts an integer to a smaller or differently signed integer type keeping the low bits, so a value that does not fit wraps around. A plain conversion `Type(value)` stops the program instead.",
		[]string{"Type", "value"}, "Type"},
	"saturate": {"func saturate(Type, value) Type", "Converts an integer or a float to an integer type, stopping at the limits of the type: a value that is too big becomes the maximum and one that is too small the minimum. A float that is not a number becomes 0.",
		[]string{"Type", "value"}, "Type"},
}

func isBuiltin(sym *analyser.Symbol) (builtinHelp, bool) {
	if sym == nil || sym.File != "" || sym.Kind != analyser.FUNC {
		return builtinHelp{}, false
	}
	help, ok := builtinHelps[sym.Name]
	return help, ok
}

var errorFieldDocs = map[string]string{
	"domain":  "The enum the error comes from, qualified with its module, as in `main.MathError`.",
	"code":    "The number of the enum member. It is the exit status when the error ends the program.",
	"message": "The text given when the error was raised, or the member's name.",
	"file":    "The file where the error was raised.",
	"line":    "The line where the error was raised.",
}

const errorDoc = "Creates an error value. `Error(\"message\")` is a plain error with the code 1 and no enum. `Error(Enum.Member)` gives it the enum as its domain and the member's number as its code, and a message after it is optional. The file and line are recorded where it is raised. `null` means no error, an error compares with null, with enum members and with other errors (the same record), and the fields `domain`, `code`, `message`, `file` and `line` can be read. A message with values is made by `fmt.errorf`."

func hover(uri protocol.DocumentUri, pos protocol.Position) *protocol.Hover {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	if state, _ := scanState(textBefore(path, int(pos.Line), int(pos.Character))); state == stateLineComment || state == stateBlockComment {
		return nil
	}

	if h := importHover(path, pos); h != nil {
		return h
	}

	if h := labelHover(path, pos); h != nil {
		return h
	}

	if h := tagHover(path, pos); h != nil {
		return h
	}

	if onPunctuation(fileBuffer[path], line, col) {
		return nil
	}

	program, info := analyseEntry(path)
	if program == nil || info == nil {
		return nil
	}
	if h := nativeHover(info, path, line, col); h != nil {
		return h
	}
	local := declarationsOf(program, declarationFiles(path), path)

	if hit := typeParamAt(local, line, col); hit != nil {
		return markdownHover(typeParamHover(hit), positionRange(hit.rng))
	}

	if text, rng := interfaceMethodDeclHover(local, line, col); text != "" {
		return markdownHover(text, rng)
	}

	if text, ok := methodDeclHover(local, fileBuffer[path], line, col); ok {
		return markdownHover(text, wordRange(fileBuffer[path], line, col))
	}

	if param, owner, idx := paramAt(local, line, col); param != nil {
		text := paramHoverText(param, owner, idx, program, info)
		if text == "" {
			return nil
		}
		return markdownHover(text, paramRange(param))
	}

	var text string
	var rng *protocol.Range

	if t, r := declMemberHover(local, line, col); t != "" {
		return markdownHover(t, r)
	}

	if expr := exprInProgram(local, line, col); expr != nil {
		if slice, isSlice := expr.(*ast.SliceExpr); isSlice {
			if rng, onDots := sliceDotsAt(fileBuffer[path], line, col); onDots {
				if t, known := info.Types[slice]; known {
					return markdownHover(renderDoc(analyser.TypeText(t), "A view of part of the value, from the start up to the end, which `..` leaves out and `..=` includes. Nothing is copied."), rng)
				}
			}
		}
		if noHoverHere(expr) {
			return nil
		}
		if prop, isPropagate := expr.(*ast.PropagateExpr); isPropagate {
			at, onOperator := propagateOperatorAt(prop, line, col)
			if !onOperator {
				return nil
			}
			return markdownHover(propagateHover(prop, info), positionRange(at))
		}
		if coalesce, isCoalesce := expr.(*ast.CoalesceExpr); isCoalesce {
			switch at, onOperator := coalesceOperatorAt(fileBuffer[path], line, col); {
			case coalesce.ErrorBind != "" && positionContains(coalesce.ErrorBindAt, line, col):
				return markdownHover(renderDoc("Error "+coalesce.ErrorBind, "The error that made the call fail."), positionRange(coalesce.ErrorBindAt))
			case onOperator:
				return markdownHover(hoverText(expr, program, info, line, col), positionRange(at))
			default:
				return nil
			}
		}
		text = hoverText(expr, program, info, line, col)
		rng = exprRange(expr)
		if narrow := literalHoverRange(expr, line, col); narrow != nil {
			rng = narrow
		}
		if block, isBlock := expr.(*ast.BlockExpr); isBlock && block.Static {
			startLine, startCol := block.Position()
			rng = positionRange(ast.Position{Line: startLine, Col: startCol, EndLine: startLine, EndCol: startCol + len("static") - 1})
		}
		if ref := exprTypeRef(expr); ref != nil && refContains(ref, line, col) {
			if t := typeRefHover(ref, program, info, line, col); t != "" {
				text, rng = t, refRange(ref)
			}
		}
	}

	if text == "" {
		if ref := typeRefAt(local, line, col); ref != nil {
			text = typeRefHover(ref, program, info, line, col)
			rng = refRange(ref)
		}
	}

	if text == "" {
		if sym := symbolAt(local, info, path, fileBuffer[path], line, col); sym != nil {
			text = symbolHover(sym, program)
			rng = nil
		}
	}

	if text == "" {
		return textualHover(path, line, col, info, program)
	}
	return markdownHover(text, rng)
}

func sliceDotsAt(src []byte, line, col int) (*protocol.Range, bool) {
	lines := splitLines(src)
	if line < 1 || line > len(lines) || col < 1 || col > len(lines[line-1]) {
		return nil, false
	}
	text := lines[line-1]
	at := col - 1
	if text[at] == '=' && at > 0 && text[at-1] == '.' {
		at--
	}
	if text[at] != '.' {
		return nil, false
	}
	from, to := at, at+1
	for from > 0 && text[from-1] == '.' {
		from--
	}
	for to < len(text) && text[to] == '.' {
		to++
	}
	if to-from < 2 {
		return nil, false
	}
	if to < len(text) && text[to] == '=' {
		to++
	}
	return positionRange(ast.Position{Line: line, Col: from + 1, EndLine: line, EndCol: to}), true
}

func noHoverHere(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.StringLiteral, *ast.IntLiteral, *ast.FloatLiteral, *ast.CharLiteral:
		return true
	case *ast.CallExpr:
		return !e.Range
	case *ast.IndexExpr, *ast.SliceExpr:
		return true
	}
	return false
}

func textualHover(path string, line, col int, info *analyser.Info, program *ast.Program) *protocol.Hover {
	lines := splitLines(fileBuffer[path])
	if line < 1 || line > len(lines) {
		return nil
	}
	text := lines[line-1]
	start, end := col-1, col-1
	for start > 0 && start <= len(text) && isIdentByte(text[start-1]) {
		start--
	}
	for end < len(text) && isIdentByte(text[end]) {
		end++
	}
	if start >= end {
		return nil
	}
	name := text[start:end]

	scope := scopeAt(info, line, col)
	var sym *analyser.Symbol
	if base, qualified := memberBase(text[:end]); qualified {
		owner, found := resolveChain(scope, base)
		if !found {
			return nil
		}
		if mt, isModule := owner.Type.(*analyser.ModuleType); isModule {
			sym = mt.Exports[name]
		} else if ft := analyser.MethodsOf(owner.Type)[name]; ft != nil {
			doc := ""
			if ft.Decl != nil {
				doc = ft.Decl.Doc
			}
			return markdownHover(renderDoc(analyser.FuncSignature(name, ft, nil), joinDocs(doc, "Method.")), wordRange(fileBuffer[path], line, col))
		}
	} else if found, ok := scope.Resolve(name); ok {
		sym = found
		if text := sharedSymbolHover(info, line, col, name); text != "" {
			return markdownHover(text, wordRange(fileBuffer[path], line, col))
		}
	}
	if sym == nil {
		return nil
	}
	content := symbolHover(sym, program)
	if content == "" {
		return nil
	}
	return markdownHover(content, wordRange(fileBuffer[path], line, col))
}

var primitiveDocs = map[string]string{
	"int":     "Signed 64-bit integer.",
	"uint":    "Unsigned 64-bit integer.",
	"int8":    "Signed 8-bit integer.",
	"int16":   "Signed 16-bit integer.",
	"int32":   "Signed 32-bit integer.",
	"int64":   "Signed 64-bit integer. `int` is another name for it.",
	"uint8":   "Unsigned 8-bit integer.",
	"uint16":  "Unsigned 16-bit integer.",
	"uint32":  "Unsigned 32-bit integer.",
	"uint64":  "Unsigned 64-bit integer. `uint` is another name for it.",
	"float":   "64-bit floating point number. Another name for `float64`.",
	"float32": "32-bit floating point number.",
	"float64": "64-bit floating point number.",
	"bool":    "`true` or `false`.",
	"byte":    "An unsigned 8-bit integer, the same type as uint8. A string is made of bytes.",
	"string":  "Text: a pointer to bytes and a length, compared by content.",
}

func layerDoc(ref *ast.TypeRef) string {
	inner := ast.TypeRef{Module: ref.Module, Name: ref.Name, Layers: ref.Layers[1:]}
	switch layer := ref.Layers[0]; {
	case layer.Optional:
		return fmt.Sprintf("`%s` or `null`. Read it with `??`, `?`, or `if x |v|`.", inner.String())
	case layer.Pointer:
		return fmt.Sprintf("A pointer to a `%s`. It is never null: `?*%s` is the pointer that can be.", inner.String(), inner.String())
	case layer.IsSpan:
		return fmt.Sprintf("A slice of `%s`: a pointer and a length.", inner.String())
	}
	return fmt.Sprintf("An array of `%s`, with its size fixed in the type.", inner.String())
}

func typeRefHover(ref *ast.TypeRef, program *ast.Program, info *analyser.Info, line, col int) string {
	if len(ref.Layers) > 0 && !ref.IsFunc {
		return renderDoc(ref.String(), layerDoc(ref))
	}
	if ref.Module == "" {
		if doc, ok := primitiveDocs[ref.Name]; ok {
			return renderDoc(ref.Name, doc)
		}
	}

	if sym := typeRefSymbol(ref, info, line, col); sym != nil {
		return symbolHover(sym, program)
	}
	return ""
}

func markdownHover(text string, rng *protocol.Range) *protocol.Hover {
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: text,
		},
		Range: rng,
	}
}

func exprRange(expr ast.Expression) *protocol.Range {
	sl, sc := expr.Position()
	el, ec := expr.EndPosition()

	return &protocol.Range{
		Start: protocol.Position{Line: uint32(sl - 1), Character: uint32(sc - 1)},
		End:   protocol.Position{Line: uint32(el - 1), Character: uint32(ec)},
	}
}

func paramRange(param *ast.Param) *protocol.Range {
	return &protocol.Range{
		Start: protocol.Position{Line: uint32(param.Line - 1), Character: uint32(param.Col - 1)},
		End:   protocol.Position{Line: uint32(param.EndLine - 1), Character: uint32(param.EndCol)},
	}
}

func renderHover(code string) string {
	return fmt.Sprintf("```wisp\n%s\n```", code)
}

func joinDocs(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n\n")
}

func renderDoc(code, doc string) string {
	if doc == "" {
		return renderHover(code)
	}
	return renderHover(code) + "\n\n" + strings.ReplaceAll(doc, "\n", "  \n")
}

func paramHoverText(param *ast.Param, owner ast.Node, idx int, program *ast.Program, info *analyser.Info) string {
	if idx >= 0 && usesTypeParam(param.Type, typeParamUses(program)) {
		doc := ""
		if _, isFunc := owner.(*ast.FuncDecl); isFunc {
			doc = "Parameter."
		}
		return renderDoc(fmt.Sprintf("%s %s", param.Type.String(), param.Name), doc)
	}
	for _, sym := range info.VarSymbols[owner] {
		if sym.Name == param.Name && sym.Line == param.Line && sym.Col == param.Col {
			if idx == -1 {
				return renderDoc(fmt.Sprintf("%s %s", analyser.TypeText(sym.Type), sym.Name), "Receiver of the method.")
			}
			return symbolHover(sym, program)
		}
	}

	typeStr := param.Type.Name
	if typeStr == "" {
		if types, ok := info.VarTypes[owner]; ok && idx >= 0 && idx < len(types) {
			typeStr = analyser.TypeText(types[idx])
		}
	}
	if typeStr == "" {
		return ""
	}

	return renderHover(fmt.Sprintf("%s %s", typeStr, param.Name))
}

func hoverText(expr ast.Expression, program *ast.Program, info *analyser.Info, line, col int) string {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		if e.Synthetic {
			return ""
		}
	case *ast.BinaryExpr:
		if e.Synthetic {
			return ""
		}
	case *ast.TernaryExpr:
		if e.Synthetic {
			return ""
		}
	case *ast.BlockExpr:
		if e.Static {
			return staticHover(e, info, line, col)
		}
		return ""
	case *ast.SwitchExpr:
		return ""
	}

	if v, ok := info.Variants[expr]; ok {
		return renderDoc(analyser.VariantText(v.Sum, v.Variant), fmt.Sprintf("Variant of `%s`.", v.Sum.Name))
	}

	switch e := expr.(type) {
	case *ast.ConvExpr:
		if t, ok := info.Types[e]; ok {
			return renderDoc(analyser.TypeText(t), "Pointer conversion: the argument becomes this type.")
		}
	case *ast.MemberExpr:
		if text := memberHover(e, program, info); text != "" {
			return text
		}
	case *ast.CoalesceExpr:
		if e.ErrorBind != "" && positionContains(e.ErrorBindAt, line, col) {
			return renderDoc("Error "+e.ErrorBind, "The error that made the call fail.")
		}
		return renderDoc("a ?? handler", "The value of `a`, or what the handler gives when `a` fails. The handler is a value, `|e| value`, or a statement that ends in `return`, `break` or `continue`.")
	case *ast.DotIdent:
		if value, ok := info.EnumValues[e]; ok {
			return enumMemberHover(value)
		}
	case *ast.StructLiteral:
		if text := structLiteralHover(e, program, info, line, col); text != "" {
			return text
		}
	case *ast.IdentLiteral:
		if text := sharedBindingHover(e, info); text != "" {
			return text
		}
		if sym := identSymbol(e, info, line, col); sym != nil {
			if ft, isFunc := sym.Type.(*analyser.FuncType); isFunc && strings.Contains(sym.Name, "wisp_range") && len(ft.Returns) == 1 {
				return renderDoc(analyser.TypeText(ft.Returns[0]), "The values from start to end, both included.")
			}
			if text := declaredTypeHover(sym, program); text != "" {
				return text
			}
			if text := symbolHover(sym, program); text != "" {
				return text
			}
		}
		if doc, ok := primitiveDocs[e.Value]; ok && info.Idents[e] == nil {
			return renderDoc(e.Value, doc)
		}
	}

	if t, ok := info.Types[expr]; ok {
		return renderHover(analyser.TypeText(t))
	}
	return ""
}

func structLiteralHover(e *ast.StructLiteral, program *ast.Program, info *analyser.Info, line, col int) string {
	st, ok := info.Types[e].(*analyser.StructType)
	if !ok {
		return ""
	}

	for i, at := range e.KeyAt {
		if i < len(e.Keys) && positionContains(at, line, col) {
			field, exists := st.Fields[e.Keys[i]]
			if !exists {
				return renderDoc(e.Keys[i], fmt.Sprintf("Struct `%s` has no field `%s`.", st.Name, e.Keys[i]))
			}
			return renderDoc(fmt.Sprintf("%s %s", analyser.TypeText(field), e.Keys[i]), joinDocs(st.FieldDocs[e.Keys[i]], fmt.Sprintf("Field of struct `%s`.", st.Name)))
		}
	}

	if e.Module == "" {
		if sym, found := scopeFor(info, line, col).Resolve(e.Name); found && sym.Kind == analyser.STRUCT {
			return symbolHover(sym, program)
		}
	}
	return structHover(st, "")
}

func positionContains(at ast.Position, line, col int) bool {
	if at.Line == 0 {
		return false
	}
	if line < at.Line || (line == at.Line && col < at.Col) {
		return false
	}
	if line > at.EndLine || (line == at.EndLine && col > at.EndCol) {
		return false
	}
	return true
}

func identSymbol(ident *ast.IdentLiteral, info *analyser.Info, line, col int) *analyser.Symbol {
	if sym, ok := info.Idents[ident]; ok {
		return sym
	}
	if sym, ok := scopeAt(info, line, col).Resolve(ident.Value); ok {
		return sym
	}
	return nil
}

func memberHover(e *ast.MemberExpr, program *ast.Program, info *analyser.Info) string {
	if value, ok := info.EnumValues[e]; ok {
		return enumMemberHover(value)
	}

	member := info.Members[e]
	if member == nil {
		return methodHover(e, program, info)
	}

	switch member.Kind {
	case analyser.MemberField:
		t, ok := info.Types[e]
		if !ok {
			return ""
		}
		doc := ""
		if member.Struct != nil {
			doc = joinDocs(member.Struct.FieldDocs[e.Field], fmt.Sprintf("Field of struct `%s`.", member.Struct.Name))
		}
		return renderDoc(fmt.Sprintf("%s %s", analyser.TypeText(t), e.Field), doc)
	case analyser.MemberErrorField:
		t, ok := info.Types[e]
		if !ok {
			return ""
		}
		return renderDoc(fmt.Sprintf("%s %s", analyser.TypeText(t), e.Field), errorFieldDocs[e.Field])
	case analyser.MemberModule:
		if sym := resolveSymbol(e, info); sym != nil {
			return symbolHover(sym, program)
		}
	}
	return ""
}

func methodHover(e *ast.MemberExpr, program *ast.Program, info *analyser.Info) string {
	for call, method := range info.MethodCalls {
		if call.Name == ast.Expression(e) && method.Func != nil {
			decl := methodDecl(program, e.Field, method.Func)
			doc := ""
			if decl != nil {
				doc = decl.Doc
			} else if method.Func.Decl != nil {
				doc = method.Func.Decl.Doc
			}
			return renderDoc(analyser.FuncSignature(e.Field, method.Func, decl), joinDocs(doc, "Method."))
		}
	}
	return ""
}

func enumMemberHover(value analyser.EnumValue) string {
	return renderDoc(fmt.Sprintf("%s.%s = %d", value.Type.Name, value.Name, value.Value), joinDocs(value.Type.MemberDocs[value.Name], fmt.Sprintf("Member of enum `%s`.", value.Type.Name)))
}

func symbolHover(sym *analyser.Symbol, program *ast.Program) string {
	if sym.Generic != nil {
		if _, isGeneric := sym.Type.(analyser.GenericType); isGeneric {
			return genericHover(sym)
		}
	}
	switch sym.Kind {
	case analyser.FUNC:
		return funcHover(sym, program)
	case analyser.STRUCT:
		if st, ok := sym.Type.(*analyser.StructType); ok {
			return structHover(st, sym.Doc)
		}
	case analyser.TYPE:
		return typeHover(sym)
	case analyser.INTERFACE:
		if it, ok := sym.Type.(*analyser.InterfaceType); ok {
			return interfaceHover(it, sym.Doc, program)
		}
	case analyser.MODULE:
		if mt, ok := sym.Type.(*analyser.ModuleType); ok {
			return moduleHover(sym.Name, mt)
		}
	default:
		return variableHover(sym)
	}
	return ""
}

func funcHover(sym *analyser.Symbol, program *ast.Program) string {
	ft, ok := sym.Type.(*analyser.FuncType)
	if !ok {
		return ""
	}

	if sym.File == "" {
		if help, ok := builtinHelps[sym.Name]; ok {
			return renderDoc(help.signature, help.doc)
		}
	}

	doc := ""
	switch {
	case sym.Extern:
		doc = fmt.Sprintf("Defined in C and linked as `%s`.", sym.CName)
	case sym.CName != "":
		doc = fmt.Sprintf("Exported to C as `%s`.", sym.CName)
	}

	name, decl := sym.Name, funcDecl(program, sym)
	if ft.Decl != nil && ft.Decl.MethodOf != "" && len(ft.Params) > 0 && len(ft.Decl.Params) == len(ft.Params) {
		method := *ft
		method.Receiver = ft.Params[0]
		method.ReceiverName = ft.Decl.Params[0].Name
		method.Params = ft.Params[1:]
		method.ParamNames = nil
		shown := *ft.Decl
		shown.Params = ft.Decl.Params[1:]
		return renderDoc(analyser.FuncSignature(ft.Decl.MethodOf, &method, &shown), joinDocs(sym.Doc, doc))
	}
	if ft.Decl != nil && ft.Decl.Instance != "" && ft.Receiver == nil {
		name, decl = ft.Decl.Instance, ft.Decl
	}

	return renderDoc(analyser.FuncSignature(name, ft, decl), joinDocs(sym.Doc, doc))
}

func funcDecl(program *ast.Program, sym *analyser.Symbol) *ast.FuncDecl {
	if program == nil || sym.File == "" {
		return nil
	}
	for _, d := range program.Declarations {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Receiver == nil && fd.Name == sym.Name {
			return fd
		}
	}
	return nil
}

func methodDecl(program *ast.Program, name string, ft *analyser.FuncType) *ast.FuncDecl {
	if program == nil || ft.Receiver == nil {
		return nil
	}
	owner := receiverName(ft.Receiver)
	for _, d := range program.Declarations {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Receiver != nil && fd.Name == name && fd.Receiver.Type.Name == owner {
			return fd
		}
	}
	return nil
}

func receiverName(t analyser.Type) string {
	if ptr, ok := t.(analyser.PointerType); ok {
		t = ptr.Element
	}
	if st, ok := t.(*analyser.StructType); ok {
		return st.Name
	}
	return t.String()
}

func structHover(st *analyser.StructType, doc string) string {
	return renderDoc(analyser.StructText(st), doc)
}

func typeHover(sym *analyser.Symbol) string {
	switch t := sym.Type.(type) {
	case analyser.ErrorType:
		return renderDoc("Error(\"message\")\nError(Enum.Member, \"message\")", errorDoc)
	case analyser.NamedType:
		if t.Enum {
			return renderDoc(analyser.EnumText(t), joinDocs(sym.Doc, fmt.Sprintf("Enum with %d members.", len(t.MemberOrder))))
		}
		return renderDoc(fmt.Sprintf("type %s %s", t.Name, analyser.TypeText(t.Underlying)), joinDocs(sym.Doc, "A distinct type: converting to and from it names the type."))
	}
	return renderHover(sym.Type.String())
}

func moduleHover(name string, mt *analyser.ModuleType) string {
	names := make([]string, 0, len(mt.Exports))
	for export := range mt.Exports {
		names = append(names, export)
	}
	sort.Strings(names)

	var b strings.Builder
	fmt.Fprintf(&b, "module %s {\n", name)
	for _, export := range names {
		fmt.Fprintf(&b, "  %s\n", exportLine(export, mt.Exports[export]))
	}
	b.WriteString("}")

	return renderDoc(b.String(), "An imported module. Its exports are reached with a dot.")
}

func exportLine(name string, sym *analyser.Symbol) string {
	switch sym.Kind {
	case analyser.FUNC:
		if ft, ok := sym.Type.(*analyser.FuncType); ok {
			return analyser.FuncSignature(name, ft, nil)
		}
	case analyser.STRUCT:
		return "struct " + name
	case analyser.TYPE:
		if nt, ok := sym.Type.(analyser.NamedType); ok && nt.Enum {
			return "enum " + name
		}
		return "type " + name
	case analyser.CONST:
		return "const " + name
	case analyser.MODULE:
		return "module " + name
	}
	return fmt.Sprintf("%s %s", analyser.TypeText(sym.Type), name)
}

func sharedSymbolHover(info *analyser.Info, line, col int, name string) string {
	var types []string
	seen := map[string]bool{}
	for _, scope := range tiedScopes(info, line, col) {
		sym, ok := scope.Resolve(name)
		if !ok || sym.Type == nil || !sym.Shared {
			continue
		}
		text := analyser.TypeText(sym.Type)
		if !seen[text] {
			seen[text] = true
			types = append(types, text)
		}
	}
	if len(types) < 2 {
		return ""
	}
	sort.Strings(types)
	return renderHover(strings.Join(types, " | ") + " " + name)
}

func sharedBindingHover(e *ast.IdentLiteral, info *analyser.Info) string {
	line, col := e.Position()
	var types []string
	seen := map[string]bool{}
	var first *analyser.Symbol
	for other, sym := range info.Idents {
		if sym == nil || other.Value != e.Value || sym.Type == nil || !sym.Shared {
			continue
		}
		ol, oc := other.Position()
		if ol != line || oc != col {
			continue
		}
		text := analyser.TypeText(sym.Type)
		if !seen[text] {
			seen[text] = true
			types = append(types, text)
		}
		if first == nil {
			first = sym
		}
	}
	if len(types) < 2 || first == nil {
		return ""
	}
	sort.Strings(types)
	return renderHover(strings.Join(types, " | ") + " " + first.Name)
}

func variableHover(sym *analyser.Symbol) string {
	if sym.Type == nil {
		return ""
	}

	if sym.Phantom != "" {
		return renderDoc(fmt.Sprintf("const %s %s", analyser.TypeText(sym.Type), sym.Name), "Value parameter: each use of it picks a constant.")
	}

	if sym.Kind == analyser.CONST && sym.Const != nil {
		value := sym.Const.Preview(8)
		if sym.Const.Kind.IsAggregate() {
			return renderDoc(fmt.Sprintf("const %s %s = %s", analyser.TypeText(sym.Type), sym.Name, value), joinDocs(sym.Doc, "Constant stored in read-only data."))
		}
		return renderDoc(fmt.Sprintf("const %s = %s", sym.Name, value), joinDocs(sym.Doc, fmt.Sprintf("Constant of type `%s`.", analyser.TypeText(sym.Type))))
	}

	doc := ""
	switch {
	case sym.Global:
		doc = "Global variable."
	case sym.Kind == analyser.PARAM:
		doc = "Parameter."
	case sym.Kind == analyser.CONST:
		doc = "Constant."
	}
	if _, isError := sym.Type.(analyser.ErrorType); isError && doc == "" {
		doc = "An error value, or `null` when there is no error."
	}

	return renderDoc(fmt.Sprintf("%s %s", analyser.TypeText(sym.Type), sym.Name), joinDocs(sym.Doc, doc))
}

func refText(ref ast.TypeRef) string {
	if ref.IsFunc {
		params := make([]string, len(ref.FuncParams))
		for i, p := range ref.FuncParams {
			params[i] = refText(p)
		}
		results := make([]string, len(ref.FuncReturns))
		for i, r := range ref.FuncReturns {
			results[i] = refText(r)
		}
		text := (&ast.TypeRef{Layers: ref.Layers}).String() + "(" + strings.Join(params, ", ") + ") => "
		if ref.FuncFallible {
			text += "!"
		}
		switch {
		case len(results) > 0:
			text += strings.Join(results, ", ")
		case !ref.FuncFallible:
			text += "()"
		}
		return text
	}

	return ref.String()
}

func declMemberHover(program *ast.Program, line, col int) (string, *protocol.Range) {
	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.StructDecl:
			for i := range d.Members {
				m := &d.Members[i]
				if paramContains(m, line, col) {
					text := renderDoc(fmt.Sprintf("%s %s", refText(m.Type), m.Name), joinDocs(m.Doc, fmt.Sprintf("Field of struct `%s`.", d.Name)))
					return text, paramRange(m)
				}
			}
		case *ast.TypeDecl:
			if !d.IsEnum {
				continue
			}
			for _, m := range d.Members {
				if positionContains(ast.Position{Line: m.Line, Col: m.Col, EndLine: m.EndLine, EndCol: m.EndCol}, line, col) {
					text := renderDoc(fmt.Sprintf("%s.%s = %d", d.Name, m.Name, m.Value), joinDocs(m.Doc, fmt.Sprintf("Member of enum `%s`.", d.Name)))
					r := rangeOf(m.Line, m.Col, m.EndLine, m.EndCol)
					return text, &r
				}
			}
		}
	}
	return "", nil
}

func wordUnderCursor(src []byte, line, col int) string {
	lines := splitLines(src)
	if line < 1 || line > len(lines) {
		return ""
	}
	text := lines[line-1]
	i := col - 1
	if i < 0 || i >= len(text) || !isIdentByte(text[i]) {
		return ""
	}
	start, end := i, i
	for start > 0 && isIdentByte(text[start-1]) {
		start--
	}
	for end < len(text) && isIdentByte(text[end]) {
		end++
	}
	return text[start:end]
}

func coalesceOperatorAt(src []byte, line, col int) (ast.Position, bool) {
	lines := splitLines(src)
	if line < 1 || line > len(lines) {
		return ast.Position{}, false
	}
	text := lines[line-1]
	i := col - 1
	for _, start := range []int{i, i - 1} {
		if start >= 0 && start+1 < len(text) && text[start] == '?' && text[start+1] == '?' {
			return ast.Position{Line: line, Col: start + 1, EndLine: line, EndCol: start + 2}, true
		}
	}
	return ast.Position{}, false
}

func staticHover(e *ast.BlockExpr, info *analyser.Info, line, col int) string {
	if !onStaticKeyword(e, line, col) {
		return ""
	}
	doc := "Runs ordinary code while the program compiles and gives the value of `yield`. It sees numbers, booleans, strings, arrays and structs, and it can call ordinary functions."
	code := "static { ... }"
	if value, ok := info.Statics[e]; ok {
		code = "static { ... } = " + value.Preview(8)
		if tp, typed := info.Types[e]; typed {
			doc += fmt.Sprintf(" Its value here has type `%s`.", analyser.TypeText(tp))
		}
	}
	return renderDoc(code, doc)
}

func usesTypeParam(ref ast.TypeRef, uses map[[2]int]bool) bool {
	found := false
	walkTypeRefs(reflect.ValueOf(ref), func(r ast.TypeRef) {
		if uses[[2]int{r.Line, r.Col}] {
			found = true
		}
	})
	return found
}

func declaredTypeHover(sym *analyser.Symbol, program *ast.Program) string {
	if sym.Line == 0 || (sym.Kind != analyser.VAR && sym.Kind != analyser.PARAM) {
		return ""
	}
	param, owner, idx := paramAt(program, sym.Line, sym.Col)
	if param == nil || idx < 0 || param.Name != sym.Name || !usesTypeParam(param.Type, typeParamUses(program)) {
		return ""
	}
	doc := ""
	if _, isFunc := owner.(*ast.FuncDecl); isFunc {
		doc = "Parameter."
	}
	return renderDoc(fmt.Sprintf("%s %s", param.Type.String(), param.Name), doc)
}

func wordRange(src []byte, line, col int) *protocol.Range {
	lines := splitLines(src)
	text := lines[line-1]
	start, end := col-1, col-1
	for start > 0 && isIdentByte(text[start-1]) {
		start--
	}
	for end < len(text) && isIdentByte(text[end]) {
		end++
	}
	r := rangeOf(line, start+1, line, end)
	return &r
}

func interfaceHover(it *analyser.InterfaceType, doc string, program *ast.Program) string {
	for _, d := range program.Declarations {
		if decl, ok := d.(*ast.InterfaceDecl); ok && decl.Name == it.Name {
			return renderDoc(interfaceDeclText(decl), joinDocs(doc, interfaceKindDoc(it)))
		}
	}
	methods := it.AllMethods()
	var b strings.Builder
	fmt.Fprintf(&b, "interface %s {", it.Name)
	for _, m := range methods {
		fmt.Fprintf(&b, "\n  func %s", m.String())
	}
	b.WriteString("\n}")
	return renderDoc(b.String(), joinDocs(doc, interfaceKindDoc(it)))
}

func interfaceKindDoc(it *analyser.InterfaceType) string {
	if it.ConstraintOnly() {
		return "A constraint: it names types, so it limits a type parameter and is not the type of a value."
	}
	return "A contract. A type declares it with `implements`, a pointer to it becomes a value of the interface (`Writer w = &x;`), and `<" + it.Name + " T>` limits a type parameter to the types that implement it."
}

func literalHoverRange(expr ast.Expression, line, col int) *protocol.Range {
	switch e := expr.(type) {
	case *ast.StructLiteral:
		for _, at := range e.KeyAt {
			if positionContains(at, line, col) {
				return positionRange(at)
			}
		}
		if e.Dot || e.Name == "" {
			return nil
		}
		startLine, startCol := e.Position()
		name := e.Name
		if e.Module != "" {
			name = e.Module + "." + e.Name
		}
		if e.Variant != "" {
			name += "." + e.Variant
		}
		at := ast.Position{Line: startLine, Col: startCol, EndLine: startLine, EndCol: startCol + len(name) - 1}
		if positionContains(at, line, col) {
			return positionRange(at)
		}
	case *ast.AnonLiteral:
		for _, at := range e.KeyAt {
			if positionContains(at, line, col) {
				return positionRange(at)
			}
		}
	}
	return nil
}

func onPunctuation(src []byte, line, col int) bool {
	lines := splitLines(src)
	if line < 1 || line > len(lines) || col < 1 || col > len(lines[line-1]) {
		return false
	}
	switch lines[line-1][col-1] {
	case ',', ';', ':':
		return true
	}
	return false
}

func propagateOperatorAt(prop *ast.PropagateExpr, line, col int) (ast.Position, bool) {
	width := 1
	if prop.Assert {
		width = 2
	}
	endLine, endCol := prop.EndPosition()
	at := ast.Position{Line: endLine, Col: endCol - width + 1, EndLine: endLine, EndCol: endCol}
	return at, positionContains(at, line, col)
}

func propagateHover(prop *ast.PropagateExpr, info *analyser.Info) string {
	value := ""
	if t, ok := info.Types[prop]; ok {
		value = analyser.TypeText(t)
	}
	var text string
	switch {
	case prop.Null:
		text = "Stops the program, naming the place, if the value is null. Otherwise it is the value."
	case prop.Assert:
		text = "Stops the program, naming the place, if the call fails. Otherwise it is the value."
	default:
		text = "Passes the error up: if the call fails, this function fails with the same error. Otherwise it is the value."
	}
	operator := "!"
	switch {
	case prop.Null:
		operator = "?"
	case prop.Assert:
		operator = "!!"
	}
	if value != "" {
		text += "\n\nThe value here is `" + value + "`."
	}
	return renderDoc(operator, text)
}
