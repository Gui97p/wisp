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

	if expr := exprInProgram(program, line, col); expr != nil {
		text = hoverText(expr, program, info, line, col)
		rng = exprRange(expr)
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

func typeText(t analyser.Type) string {
	switch v := t.(type) {
	case *analyser.StructType:
		return v.Name
	case analyser.PointerType:
		return "*" + typeText(v.Element)
	case analyser.ArrayType:
		return fmt.Sprintf("%s[%d]", typeText(v.Element), v.Size)
	case analyser.SpanType:
		return typeText(v.Element) + "[]"
	case analyser.UntypedIntType:
		return "int"
	case analyser.UntypedFloatType:
		return "float"
	}
	return t.String()
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
				return renderDoc(fmt.Sprintf("%s %s", typeText(sym.Type), sym.Name), "Receiver of the method.")
			}
			return symbolHover(sym, program)
		}
	}

	typeStr := param.Type.Name
	if typeStr == "" {
		if types, ok := info.VarTypes[owner]; ok && idx >= 0 && idx < len(types) {
			typeStr = typeText(types[idx])
		}
	}
	if typeStr == "" {
		return ""
	}

	return renderHover(fmt.Sprintf("%s %s", typeStr, param.Name))
}

func hoverText(expr ast.Expression, program *ast.Program, info *analyser.Info, line, col int) string {
	switch e := expr.(type) {
	case *ast.MemberExpr:
		if text := memberHover(e, program, info); text != "" {
			return text
		}
	case *ast.DotIdent:
		if value, ok := info.EnumValues[e]; ok {
			return enumMemberHover(value)
		}
	case *ast.IdentLiteral:
		if sym := identSymbol(e, info, line, col); sym != nil {
			if text := symbolHover(sym, program); text != "" {
				return text
			}
		}
	}

	if t, ok := info.Types[expr]; ok {
		return renderHover(typeText(t))
	}
	return ""
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
			doc = fmt.Sprintf("Field of struct `%s`.", member.Struct.Name)
		}
		return renderDoc(fmt.Sprintf("%s %s", typeText(t), e.Field), doc)
	case analyser.MemberErrorField:
		t, ok := info.Types[e]
		if !ok {
			return ""
		}
		return renderDoc(fmt.Sprintf("%s %s", typeText(t), e.Field), errorFieldDocs[e.Field])
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
			return renderDoc(funcSignature(e.Field, method.Func, decl), joinDocs(doc, "Method."))
		}
	}
	return ""
}

func enumMemberHover(value analyser.EnumValue) string {
	return renderDoc(fmt.Sprintf("%s.%s = %d", value.Type.Name, value.Name, value.Value), fmt.Sprintf("Member of enum `%s`.", value.Type.Name))
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

	return renderDoc(funcSignature(sym.Name, ft, funcDecl(program, sym)), joinDocs(sym.Doc, doc))
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

func funcSignature(name string, ft *analyser.FuncType, decl *ast.FuncDecl) string {
	receiver := ""
	if ft.Receiver != nil {
		receiver = "(" + typeText(ft.Receiver)
		if decl != nil && decl.Receiver != nil && decl.Receiver.Name != "" {
			receiver += " " + decl.Receiver.Name
		}
		receiver += ") "
	}

	params := make([]string, 0, len(ft.Params)+1)
	for i, p := range ft.Params {
		label := typeText(p)
		paramName := ""
		if decl != nil && i < len(decl.Params) {
			paramName = decl.Params[i].Name
		}
		if ft.Variadic && i == len(ft.Params)-1 {
			label += " ..."
		}
		if paramName != "" {
			label += " " + paramName
		}
		params = append(params, label)
	}
	if ft.CVariadic {
		params = append(params, "...")
	}

	return fmt.Sprintf("func %s%s(%s)%s", receiver, name, strings.Join(params, ", "), returnsText(ft))
}

func returnsText(ft *analyser.FuncType) string {
	var b strings.Builder
	if ft.Fallible {
		b.WriteString(" !")
	}
	switch len(ft.Returns) {
	case 0:
	case 1:
		if !ft.Fallible {
			b.WriteString(" ")
		}
		b.WriteString(typeText(ft.Returns[0]))
	default:
		if !ft.Fallible {
			b.WriteString(" ")
		}
		parts := make([]string, len(ft.Returns))
		for i, r := range ft.Returns {
			parts[i] = typeText(r)
		}
		b.WriteString("(")
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(")")
	}
	return b.String()
}

func structHover(st *analyser.StructType, doc string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "struct %s {\n", st.Name)
	for _, field := range st.Order {
		fmt.Fprintf(&b, "  %s %s\n", typeText(st.Fields[field]), field)
	}
	b.WriteString("}")

	names := make([]string, 0, len(st.Methods))
	for name := range st.Methods {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		b.WriteString("\n")
		for _, name := range names {
			fmt.Fprintf(&b, "\n%s", funcSignature(name, st.Methods[name], nil))
		}
	}

	return renderDoc(b.String(), doc)
}

func typeHover(sym *analyser.Symbol) string {
	switch t := sym.Type.(type) {
	case analyser.ErrorType:
		return renderDoc("Error(Enum.Member, \"message\")", errorDoc)
	case analyser.NamedType:
		if t.Enum {
			return renderDoc(enumText(t), joinDocs(sym.Doc, fmt.Sprintf("Enum with %d members.", len(t.MemberOrder))))
		}
		return renderDoc(fmt.Sprintf("type %s %s", t.Name, typeText(t.Underlying)), joinDocs(sym.Doc, "A distinct type: converting to and from it names the type."))
	}
	return renderHover(sym.Type.String())
}

func enumText(t analyser.NamedType) string {
	var b strings.Builder
	fmt.Fprintf(&b, "enum %s {\n", t.Name)
	for _, name := range t.MemberOrder {
		fmt.Fprintf(&b, "  %s = %d,\n", name, t.Members[name])
	}
	b.WriteString("}")
	return b.String()
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
			return funcSignature(name, ft, nil)
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
	return fmt.Sprintf("%s %s", typeText(sym.Type), name)
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
		return renderDoc(fmt.Sprintf("const %s = %s", sym.Name, value), joinDocs(sym.Doc, fmt.Sprintf("Constant of type `%s`.", typeText(sym.Type))))
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

	return renderDoc(fmt.Sprintf("%s %s", typeText(sym.Type), sym.Name), joinDocs(sym.Doc, doc))
}
