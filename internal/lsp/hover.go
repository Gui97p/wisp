package lsp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type builtinHelp struct {
	signature string
	doc       string
}

var builtinHelps = map[string]builtinHelp{
	"emit":    {"func emit(string text)", "Writes a string and a newline to standard output."},
	"len":     {"func len(x) int", "The length of an array, slice, string or variadic parameter."},
	"panic":   {"func panic(string message)", "Stops the program with the message and the source position, and exits with code 1."},
	"malloc":  {"func malloc(int size) !*void", "Asks for `size` bytes of heap. Fails with `AllocError.OutOfMemory`, so it is used with `!`, `??` or destructuring. The `*void` converts to any pointer type."},
	"realloc": {"func realloc(*void p, int size) !*void", "Resizes a block and keeps its contents. `realloc(null, n)` is `malloc(n)`. If it fails, the original block is untouched."},
	"free":    {"func free(p, ...)", "Releases any number of pointers, strings or maps. `free(null)` does nothing, and every argument is evaluated before anything is released."},
	"sizeof":  {"func sizeof(x) int", "The size in bytes of a type or of an expression."},
}

var errorFieldDocs = map[string]string{
	"domain":  "The enum the error comes from, qualified with its module, as in `main.MathError`.",
	"code":    "The number of the enum member. It is the exit status when the error ends the program.",
	"message": "The text given when the error was raised, or the member's name.",
	"file":    "The file where the error was raised.",
	"line":    "The line where the error was raised.",
}

const errorDoc = "Creates an error value. The enum gives it its domain and the member's number is its code. The message is optional, and the file and line are recorded where it is raised. `null` means no error, and the fields `domain`, `code`, `message`, `file` and `line` can be read."

func hover(uri protocol.DocumentUri, pos protocol.Position) *protocol.Hover {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	if h := importHover(path, pos); h != nil {
		return h
	}

	program, info := analyseEntry(path)
	if program == nil || info == nil {
		return nil
	}

	if param, owner, idx := paramAt(program, line, col); param != nil {
		text := paramHoverText(param, owner, idx, program, info)
		if text == "" {
			return nil
		}
		return markdownHover(text, paramRange(param))
	}

	var text string
	var rng *protocol.Range

	if t, r := declMemberHover(program, line, col); t != "" {
		return markdownHover(t, r)
	}

	if expr := exprInProgram(program, line, col); expr != nil {
		if _, isCoalesce := expr.(*ast.CoalesceExpr); isCoalesce && isHandlerKeyword(wordUnderCursor(fileBuffer[path], line, col)) {
			return nil
		}
		text = hoverText(expr, program, info, line, col)
		rng = exprRange(expr)
		if conv, ok := expr.(*ast.ConvExpr); ok && refContains(&conv.Type, line, col) {
			if t := typeRefHover(&conv.Type, program, info, line, col); t != "" {
				text, rng = t, refRange(&conv.Type)
			}
		}
	}

	if text == "" {
		if ref := typeRefAt(program, line, col); ref != nil {
			text = typeRefHover(ref, program, info, line, col)
			rng = refRange(ref)
		}
	}

	if text == "" {
		if sym := symbolAt(program, info, fileBuffer[path], line, col); sym != nil {
			text = symbolHover(sym, program)
			rng = nil
		}
	}

	if text == "" {
		return nil
	}
	return markdownHover(text, rng)
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
	"char":    "A single character, stored in one byte.",
	"string":  "Text: a pointer to bytes and a length, compared by content.",
	"map":     "A collection of values found by key.",
}

func typeRefHover(ref *ast.TypeRef, program *ast.Program, info *analyser.Info, line, col int) string {
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
	case *ast.SwitchExpr:
		return ""
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
		if sym := identSymbol(e, info, line, col); sym != nil {
			if text := symbolHover(sym, program); text != "" {
				return text
			}
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
	switch sym.Kind {
	case analyser.FUNC:
		return funcHover(sym, program)
	case analyser.STRUCT:
		if st, ok := sym.Type.(*analyser.StructType); ok {
			return structHover(st, sym.Doc)
		}
	case analyser.TYPE:
		return typeHover(sym)
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

	return renderDoc(analyser.FuncSignature(sym.Name, ft, funcDecl(program, sym)), joinDocs(sym.Doc, doc))
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
		return renderDoc("Error(Enum.Member, \"message\")", errorDoc)
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

func variableHover(sym *analyser.Symbol) string {
	if sym.Type == nil {
		return ""
	}

	if sym.Kind == analyser.CONST && sym.Const != nil {
		value := sym.Const.String()
		if sym.Const.Kind == analyser.ConstString {
			value = strconv.Quote(sym.Const.Str)
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
	if ref.IsMap && ref.MapKey != nil && ref.MapValue != nil {
		return "map[" + refText(*ref.MapKey) + "]" + refText(*ref.MapValue)
	}
	if ref.IsFunc {
		params := make([]string, len(ref.FuncParams))
		for i, p := range ref.FuncParams {
			params[i] = refText(p)
		}
		var text strings.Builder
		text.WriteString("(")
		text.WriteString(strings.Join(params, ", "))
		text.WriteString(")")
		if ref.FuncFallible {
			text.WriteString(" !")
		}
		for i, r := range ref.FuncReturns {
			if i == 0 {
				text.WriteString(" ")
			} else {
				text.WriteString(", ")
			}
			text.WriteString(refText(r))
		}
		return text.String()
	}

	var text strings.Builder
	text.WriteString(strings.Repeat("*", ref.PointerDepth))
	if ref.Module != "" {
		text.WriteString(ref.Module)
		text.WriteString(".")
	}
	text.WriteString(ref.Name)
	for _, dim := range ref.Dims {
		if dim.IsSpan {
			text.WriteString("[]")
		} else if dim.Expr != nil {
			fmt.Fprintf(&text, "[%s]", dim.Text)
		} else {
			fmt.Fprintf(&text, "[%d]", dim.Size)
		}
	}
	return text.String()
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

func isHandlerKeyword(word string) bool {
	return word == "return" || word == "break" || word == "continue"
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
