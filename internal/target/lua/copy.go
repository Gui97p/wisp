package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type aggregateFns struct {
	copyName string
	intoName string
	copyCode string
	intoCode string
}

func typeKey(tp analyser.Type) string {
	switch v := underlying(tp).(type) {
	case *analyser.StructType:
		return fmt.Sprintf("s%p", v)
	case analyser.ArrayType:
		return fmt.Sprintf("a%d:%s", v.Size, typeKey(v.Element))
	}
	return ""
}

func (t *LuaTarget) aggregateFns(tp analyser.Type) *aggregateFns {
	tp = underlying(tp)
	key := typeKey(tp)
	if fns, ok := t.aggregates[key]; ok {
		return fns
	}
	if t.aggregates == nil {
		t.aggregates = map[string]*aggregateFns{}
	}

	n := len(t.aggregates)
	fns := &aggregateFns{
		copyName: fmt.Sprintf("__wisp_copy_%d", n),
		intoName: fmt.Sprintf("__wisp_into_%d", n),
	}
	t.aggregates[key] = fns
	t.aggregateOrder = append(t.aggregateOrder, fns)

	var copyBody, intoBody strings.Builder
	switch v := tp.(type) {
	case *analyser.StructType:
		var fields []string
		for _, name := range v.Order {
			ft := v.Fields[name]
			access := fieldAccess(name)
			if isAggregate(ft) {
				inner := t.aggregateFns(ft)
				fields = append(fields, fieldKey(name)+inner.copyName+"(x"+access+")")
				fmt.Fprintf(&intoBody, "%s(dst%s, src%s)\n", inner.intoName, access, access)
			} else {
				fields = append(fields, fieldKey(name)+"x"+access)
				fmt.Fprintf(&intoBody, "dst%s = src%s\n", access, access)
			}
		}
		fmt.Fprintf(&copyBody, "return {%s}\n", strings.Join(fields, ", "))
	case analyser.ArrayType:
		element := "v.t[v.o + i]"
		store := "dst.t[dst.o + i] = src.t[src.o + i]"
		if isAggregate(v.Element) {
			inner := t.aggregateFns(v.Element)
			element = inner.copyName + "(v.t[v.o + i])"
			store = fmt.Sprintf("%s(dst.t[dst.o + i], src.t[src.o + i])", inner.intoName)
		}
		fmt.Fprintf(&copyBody, "local t = {}\nfor i = 0, v.n - 1 do\nt[i] = %s\nend\nreturn {t = t, o = 0, n = v.n}\n", element)
		fmt.Fprintf(&intoBody, "for i = 0, src.n - 1 do\n%s\nend\n", store)
	}

	param := "x"
	if _, isArray := tp.(analyser.ArrayType); isArray {
		param = "v"
	}
	fns.copyCode = fmt.Sprintf("%s = function(%s)\n%send\n", fns.copyName, param, copyBody.String())
	fns.intoCode = fmt.Sprintf("%s = function(dst, src)\n%send\n", fns.intoName, intoBody.String())
	return fns
}

func (t *LuaTarget) aggregateCode() string {
	if len(t.aggregateOrder) == 0 {
		return ""
	}

	var names []string
	for _, fns := range t.aggregateOrder {
		names = append(names, fns.copyName, fns.intoName)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "local %s\n", strings.Join(names, ", "))
	for _, fns := range t.aggregateOrder {
		b.WriteString(fns.copyCode)
		b.WriteString(fns.intoCode)
	}
	return b.String()
}

func isStorageExpr(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IdentLiteral, *ast.MemberExpr, *ast.IndexExpr, *ast.DerefExpr, *ast.TernaryExpr, *ast.SwitchExpr, *ast.CoalesceExpr:
		return true
	case *ast.CastExpr:
		return isStorageExpr(e.Value)
	}
	return false
}

func (t *LuaTarget) valueText(expr ast.Expression, text string) string {
	tp := t.info.Types[expr]
	if tp == nil || !isAggregate(tp) || !isStorageExpr(expr) {
		return text
	}
	return t.aggregateFns(tp).copyName + "(" + text + ")"
}
