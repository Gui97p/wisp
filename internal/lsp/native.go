package lsp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/arch"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var nativeBlockHeader = regexp.MustCompile(`\bnative\s*(?:<[^<>]*>)?(?:\s+\w+\s*\([^)]*\)[^{};]*)?\s*$`)

var nativeLabel = regexp.MustCompile(`(?m)^\s*(\w+)\s*:`)

var nativeWord = regexp.MustCompile(`[\w.&]*$`)

func innermostOpenBrace(text string) int {
	depth := 0
	for i := len(text) - 1; i >= 0; i-- {
		switch text[i] {
		case '}':
			depth++
		case '{':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func nativeItems(path string, pos protocol.Position, info *analyser.Info) ([]protocol.CompletionItem, bool) {
	text := textBefore(path, int(pos.Line), int(pos.Character))
	open := innermostOpenBrace(text)
	if open < 0 || !nativeBlockHeader.MatchString(text[:open]) {
		return nil, false
	}

	items := []protocol.CompletionItem{}
	if info == nil {
		return items, true
	}

	scope := scopeAt(info, int(pos.Line)+1, int(pos.Character)+1)
	word := strings.TrimPrefix(nativeWord.FindString(text), "&")
	parts := strings.Split(word, ".")

	if len(parts) == 1 {
		ctx := nativeContextAt(text, open)
		if ctx.atMnemonic || ctx.mnemonic == "extern" {
			return items, true
		}
		add := func(it protocol.CompletionItem, rank string) {
			it.SortText = ptr(rank + it.Label)
			items = append(items, it)
		}

		jump := strings.HasPrefix(ctx.mnemonic, "j") || strings.HasPrefix(ctx.mnemonic, "loop")
		calls := ctx.mnemonic == "call" || ctx.mnemonic == "jmp" || (ctx.mnemonic == "lea" && ctx.operand == 1)
		values := !jump && ctx.mnemonic != "call" && !ctx.inBracket

		for _, sym := range scope.Visible() {
			switch sym.Kind {
			case analyser.VAR, analyser.PARAM:
				if values && sym.Type != nil {
					add(item(sym.Name, protocol.CompletionItemKindVariable, sym.Type.String()), "0")
				}
			case analyser.CONST:
				if _, isConst := analyser.NativeConstant(sym); isConst && sym.Type != nil && !jump {
					add(item(sym.Name, protocol.CompletionItemKindConstant, sym.Type.String()), "1")
				}
			case analyser.FUNC:
				if calls && !ctx.inBracket && sym.Type != nil && sym.File != "" {
					add(item(sym.Name, protocol.CompletionItemKindFunction, sym.Type.String()), "0")
				}
			case analyser.MODULE:
				if calls && !ctx.inBracket {
					add(item(sym.Name, protocol.CompletionItemKindModule, "module"), "1")
				}
			}
		}
		if jump || ctx.mnemonic == "call" || ctx.mnemonic == "jmp" {
			for _, m := range nativeLabel.FindAllStringSubmatch(text[open:], -1) {
				add(item(m[1], protocol.CompletionItemKindEnumMember, "label"), "0")
			}
		}
		if !jump && ctx.mnemonic != "lea" || ctx.inBracket {
			for _, name := range arch.RegisterNames() {
				add(item(name, protocol.CompletionItemKindKeyword, "register"), "2")
			}
		}
		if !jump && ctx.mnemonic != "lea" && !ctx.inBracket {
			for _, name := range []string{"byte", "word", "dword", "qword"} {
				add(item(name, protocol.CompletionItemKindKeyword, "size"), "3")
			}
		}
		if ctx.inBracket {
			add(item("rel", protocol.CompletionItemKindKeyword, "size"), "3")
		}
		return items, true
	}

	sym, ok := scope.Resolve(parts[0])
	if !ok {
		return items, true
	}

	if module, isModule := sym.Type.(*analyser.ModuleType); isModule && len(parts) == 2 {
		names := make([]string, 0, len(module.Exports))
		for name := range module.Exports {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			member := module.Exports[name]
			switch member.Kind {
			case analyser.FUNC:
				items = append(items, item(name, protocol.CompletionItemKindFunction, member.Type.String()))
			case analyser.CONST:
				if _, isConst := analyser.NativeConstant(member); isConst || member.Const != nil {
					items = append(items, item(name, protocol.CompletionItemKindConstant, member.Type.String()))
				}
			}
		}
		return items, true
	}

	tp := sym.Type
	for _, part := range parts[1 : len(parts)-1] {
		tp = nativeMemberType(tp, part)
		if tp == nil {
			return items, true
		}
	}

	if st := nativeStructOf(tp); st != nil && st.Optional == nil {
		for _, field := range st.Order {
			if hiddenField(field) {
				continue
			}
			items = append(items, item(field, protocol.CompletionItemKindField, st.Fields[field].String()))
		}
	} else if nativeWordPair(tp) {
		items = append(items,
			item("ptr", protocol.CompletionItemKindField, "int"),
			item("len", protocol.CompletionItemKindField, "int"))
	}
	return items, true
}

func nativeMemberType(tp analyser.Type, part string) analyser.Type {
	if st := nativeStructOf(tp); st != nil {
		return st.Fields[part]
	}
	if nativeWordPair(tp) && (part == "ptr" || part == "len") {
		return analyser.PrimitiveType{Name: "int"}
	}
	return nil
}

func nativeStructOf(tp analyser.Type) *analyser.StructType {
	switch v := tp.(type) {
	case *analyser.StructType:
		return v
	case analyser.NamedType:
		return nativeStructOf(v.Underlying)
	}
	return nil
}

func nativeWordPair(tp analyser.Type) bool {
	switch v := tp.(type) {
	case analyser.PrimitiveType:
		return v.Name == "string"
	case analyser.SpanType:
		return true
	case analyser.NamedType:
		return nativeWordPair(v.Underlying)
	}
	return false
}

func positionRange(at ast.Position) *protocol.Range {
	return &protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(at.Line - 1), Character: protocol.UInteger(at.Col - 1)},
		End:   protocol.Position{Line: protocol.UInteger(at.EndLine - 1), Character: protocol.UInteger(at.EndCol)},
	}
}

func positionHas(at ast.Position, line, col int) bool {
	if at.Line == 0 || line < at.Line || line > at.EndLine {
		return false
	}
	if line == at.Line && col < at.Col {
		return false
	}
	return !(line == at.EndLine && col > at.EndCol)
}

var calleeSaved = map[string]bool{"rbx": true, "rbp": true, "r12": true, "r13": true, "r14": true, "r15": true}

func registerHelp(reg arch.Register) string {
	var b strings.Builder
	switch reg.Class {
	case arch.Vector:
		fmt.Fprintf(&b, "```\n%s\n```\n%d-bit vector register", reg.Name, reg.Width)
	case arch.Segment:
		fmt.Fprintf(&b, "```\n%s\n```\nsegment register", reg.Name)
	case arch.Pointer:
		fmt.Fprintf(&b, "```\n%s\n```\ninstruction pointer", reg.Name)
	default:
		fmt.Fprintf(&b, "```\n%s\n```\n%d-bit general register", reg.Name, reg.Width)
	}
	if reg.Name != reg.Canonical {
		fmt.Fprintf(&b, ", part of `%s`", reg.Canonical)
	}
	switch {
	case calleeSaved[reg.Canonical]:
		b.WriteString(".\n\nA function must give it back as it received it: the compiler saves it when a block names it.")
	case reg.Class == arch.General || reg.Class == arch.Vector:
		b.WriteString(".\n\nA call may change it.")
	}
	return b.String()
}

func refHelp(ref analyser.NativeRef) string {
	switch ref.Kind {
	case analyser.RefRegister:
		return registerHelp(ref.Reg)
	case analyser.RefVariable:
		name := ref.Sym.Name
		if len(ref.Path) > 0 {
			name += "." + strings.Join(ref.Path, ".")
		}
		tp := ""
		if ref.Type != nil {
			tp = ref.Type.String()
		}
		return fmt.Sprintf("```wisp\n%s %s\n```\nA Wisp variable. A native block that names it reads and writes it in memory.", tp, name)
	case analyser.RefFunction:
		sig := ""
		if ref.Sym.Type != nil {
			sig = ref.Sym.Type.String()
		}
		if !strings.HasPrefix(sig, ref.Sym.Name) {
			sig = ref.Sym.Name + sig
		}
		return fmt.Sprintf("```wisp\nfunc %s\n```\nThe address of the function. The block follows its calling convention.", sig)
	case analyser.RefLabel:
		return fmt.Sprintf("```\n%s:\n```\nA label of this block.", ref.Text)
	case analyser.RefExtern:
		return fmt.Sprintf("```\n%s\n```\nA symbol of the assembler, declared with `extern`.", ref.Text)
	case analyser.RefConstPart:
		return fmt.Sprintf("```wisp\n%s\n```\nA part of a string constant.", ref.Sym.Name)
	case analyser.RefNumber:
		return fmt.Sprintf("```\n%s\n```\nA number or a constant.", ref.Text)
	}
	return ""
}

func nativeHover(info *analyser.Info, line, col int) *protocol.Hover {
	for stmt, native := range info.Natives {
		for i, instr := range stmt.Instrs {
			if !positionHas(instr.At, line, col) {
				continue
			}
			for j, op := range instr.Operands {
				if i >= len(native.Instrs) || j >= len(native.Instrs[i].Operands) {
					continue
				}
				info := native.Instrs[i].Operands[j]
				if info.Mem {
					for k, term := range op.Terms {
						if positionHas(term.At, line, col) && k < len(info.Terms) {
							return markdownHover(refHelp(info.Terms[k].Ref), positionRange(term.At))
						}
					}
					continue
				}
				if positionHas(op.At, line, col) {
					if text := refHelp(info.Ref); text != "" {
						return markdownHover(text, positionRange(op.At))
					}
				}
			}
			if instr.Mnemonic != "" && line == instr.At.Line {
				word := strings.Join(append(append([]string{}, instr.Prefixes...), instr.Mnemonic), " ")
				if col >= instr.At.Col && col < instr.At.Col+len(word) {
					return mnemonicHover(instr, positionRange(ast.Position{Line: line, Col: instr.At.Col, EndLine: line, EndCol: instr.At.Col + len(word) - 1}))
				}
			}
		}
	}
	return nil
}

func mnemonicHover(instr ast.NativeInstr, rng *protocol.Range) *protocol.Hover {
	effects := arch.InstrEffects(instr.Prefixes, instr.Mnemonic, len(instr.Operands))
	touches := arch.Canonical(effects.Touches)
	var b strings.Builder
	fmt.Fprintf(&b, "```\n%s\n```\nPassed to the assembler as written.", instr.Mnemonic)
	switch {
	case strings.EqualFold(instr.Mnemonic, "call"):
		b.WriteString("\n\nA call may change every register the callee does: all but `rbx`, `rbp` and `r12` to `r15`.")
	case len(touches) > 0:
		fmt.Fprintf(&b, "\n\nAlso uses: `%s`.", strings.Join(touches, "`, `"))
	}
	if effects.Stack {
		b.WriteString("\n\nUses the stack: the function keeps its frame.")
	}
	return markdownHover(b.String(), rng)
}

type nativeContext struct {
	mnemonic   string
	operand    int
	inBracket  bool
	atMnemonic bool
}

var nativeLead = regexp.MustCompile(`^\s*(?:\w+\s*:)?\s*`)

func nativeContextAt(text string, open int) nativeContext {
	stmt := text[open+1:]
	if cut := strings.LastIndexByte(stmt, ';'); cut >= 0 {
		stmt = stmt[cut+1:]
	}
	stmt = stmt[len(nativeLead.FindString(stmt)):]

	words := strings.Fields(stmt)
	for len(words) > 0 && (words[0] == "rep" || words[0] == "repe" || words[0] == "repne" || words[0] == "repz" || words[0] == "repnz" || words[0] == "lock") && strings.ContainsAny(stmt, " \t") {
		stmt = strings.TrimLeft(strings.TrimPrefix(strings.TrimLeft(stmt, " \t"), words[0]), " \t")
		words = words[1:]
	}
	if len(words) == 0 || (len(words) == 1 && !strings.HasSuffix(stmt, " ") && !strings.HasSuffix(stmt, "\t")) {
		return nativeContext{atMnemonic: true}
	}

	ctx := nativeContext{mnemonic: strings.ToLower(words[0])}
	rest := strings.TrimLeft(strings.TrimPrefix(strings.TrimLeft(stmt, " \t"), words[0]), " \t")
	for _, ch := range rest {
		switch ch {
		case '[':
			ctx.inBracket = true
		case ']':
			ctx.inBracket = false
		case ',':
			if !ctx.inBracket {
				ctx.operand++
			}
		}
	}
	return ctx
}

type nativeSite struct {
	line, col int
	sym       *analyser.Symbol
}

func nativeSites(info *analyser.Info) []nativeSite {
	var sites []nativeSite
	add := func(ref analyser.NativeRef, path []string, at ast.Position, addr bool) {
		if ref.Sym == nil || (ref.Kind != analyser.RefVariable && ref.Kind != analyser.RefFunction) {
			return
		}
		col := at.Col
		if addr {
			col++
		}
		if ref.Kind == analyser.RefFunction && len(path) > 1 {
			col += len(strings.Join(path[:len(path)-1], ".")) + 1
		}
		sites = append(sites, nativeSite{at.Line, col, ref.Sym})
	}
	for stmt, native := range info.Natives {
		for i, instr := range stmt.Instrs {
			if i >= len(native.Instrs) {
				break
			}
			for j, op := range instr.Operands {
				if j >= len(native.Instrs[i].Operands) {
					break
				}
				resolved := native.Instrs[i].Operands[j]
				if resolved.Mem {
					for k, term := range op.Terms {
						if k < len(resolved.Terms) {
							add(resolved.Terms[k].Ref, term.Path, term.At, false)
						}
					}
					continue
				}
				add(resolved.Ref, op.Path, op.At, resolved.Addr)
			}
		}
	}
	return sites
}

func nativeSymbolAt(info *analyser.Info, line, col int) *analyser.Symbol {
	for _, s := range nativeSites(info) {
		if s.line == line && col >= s.col && col < s.col+len(s.sym.Name) {
			return s.sym
		}
	}
	return nil
}
