package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type inlineFrame struct {
	call       *analyser.InlineCall
	results    []string
	label      string
	deferDepth int
}

func (t *LuaTarget) compileInlineCall(b *strings.Builder, expr *ast.CallExpr, call *analyser.InlineCall) error {
	outer := t.pending
	t.pending = nil

	var inner strings.Builder
	inner.WriteString("do\n")

	var temps []string
	addArgument := func(arg ast.Expression, byValue bool) error {
		text, err := t.compileExprScratch(arg)
		if err != nil {
			return err
		}
		if byValue {
			text = t.valueText(arg, text)
		}
		t.flushPending(&inner)
		temp := t.newLabel("arg")
		fmt.Fprintf(&inner, "local %s = %s\n", temp, text)
		temps = append(temps, temp)
		return nil
	}

	if call.Receiver != nil {
		receiver, byValue := ast.Expression(nil), true
		if mc, ok := t.info.MethodCalls[expr]; ok && mc.Receiver != nil {
			receiver, byValue = mc.Receiver, !mc.PointerReceiver
		} else if member, ok := expr.Name.(*ast.MemberExpr); ok {
			receiver = member.Object
		}
		if err := addArgument(receiver, byValue); err != nil {
			t.pending = outer
			return err
		}
	}
	for _, i := range call.ArgIndex {
		if err := addArgument(expr.Args[i], true); err != nil {
			t.pending = outer
			return err
		}
	}

	var names []string
	if call.Receiver != nil {
		recvName := call.Receiver.Name
		if recvName == "" {
			recvName = "_"
		}
		names = append(names, luaIdent(recvName))
	}
	for _, param := range call.Params {
		names = append(names, luaIdent(param.Name))
	}
	if len(names) > 0 {
		fmt.Fprintf(&inner, "local %s = %s\n", strings.Join(names, ", "), strings.Join(temps, ", "))
	}
	symbols := call.Params
	if call.Receiver != nil {
		symbols = append([]*analyser.Symbol{call.Receiver}, symbols...)
	}
	for _, sym := range symbols {
		if t.isBoxed(sym) {
			fmt.Fprintf(&inner, "local %s = {[0]=%s}\n", luaIdent(sym.Name), luaIdent(sym.Name))
		}
	}

	count := len(call.Returns)
	if call.Fallible {
		count++
	}
	results := make([]string, count)
	for i := range results {
		results[i] = t.newLabel("inl")
	}
	frame := inlineFrame{call: call, results: results, label: t.newLabel("inlend"), deferDepth: len(t.deferScopes)}

	prevFallible, prevPayloads := t.currentFallible, t.currentPayloads
	t.currentFallible, t.currentPayloads = call.Fallible, call.Returns
	t.inlineFrames = append(t.inlineFrames, frame)
	err := t.compileStatement(&inner, call.Body)
	t.inlineFrames = t.inlineFrames[:len(t.inlineFrames)-1]
	t.currentFallible, t.currentPayloads = prevFallible, prevPayloads
	if err != nil {
		t.pending = outer
		return err
	}

	inner.WriteString("end\n")
	fmt.Fprintf(&inner, "::%s::\n", frame.label)

	t.pending = outer
	if len(results) > 0 {
		t.emitPending(fmt.Sprintf("local %s\n", strings.Join(results, ", ")))
	}
	t.emitPending(inner.String())

	b.WriteString(strings.Join(results, ", "))
	return nil
}

func (t *LuaTarget) compileInlineReturn(b *strings.Builder, stmt *ast.ReturnStmt) error {
	frame := t.inlineFrames[len(t.inlineFrames)-1]
	isError := t.info.ErrorReturns[stmt]

	texts := make([]string, 0, len(stmt.Values)+len(frame.results))
	for _, value := range stmt.Values {
		text, err := t.compileExprScratch(value)
		if err != nil {
			return err
		}
		texts = append(texts, t.valueText(value, text))
	}

	switch {
	case isError:
		zeros, err := t.payloadZeros()
		if err != nil {
			return err
		}
		texts = append(zeros, texts...)
	case frame.call.Fallible:
		texts = append(texts, "nil")
	}

	t.flushPending(b)

	if t.hasDefers(frame.deferDepth, isError) {
		packed := t.newLabel("ret")
		b.WriteString("do\n")
		fmt.Fprintf(b, "local %s = __lua_table.pack(%s)\n", packed, strings.Join(texts, ", "))
		if err := t.emitDefers(b, frame.deferDepth, isError, packed+"["+packed+".n]"); err != nil {
			return err
		}
		if len(frame.results) > 0 {
			fmt.Fprintf(b, "%s = __lua_table.unpack(%s, 1, %s.n)\n", strings.Join(frame.results, ", "), packed, packed)
		}
		fmt.Fprintf(b, "goto %s\n", frame.label)
		b.WriteString("end\n")
		return nil
	}

	if len(frame.results) > 0 {
		fmt.Fprintf(b, "%s = %s\n", strings.Join(frame.results, ", "), strings.Join(texts, ", "))
	}
	fmt.Fprintf(b, "goto %s\n", frame.label)
	return nil
}
