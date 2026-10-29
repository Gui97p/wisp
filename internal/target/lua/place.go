package lua

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type placeKind int

const (
	placeText placeKind = iota
	placeView
	placePointerIndex
	placeDeref
)

type placeSpec struct {
	kind  placeKind
	text  string
	base  string
	index string
	typ   analyser.Type
	into  string
}

var simpleText = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*|-?[0-9]+)$`)

func (t *LuaTarget) compilePlace(target ast.Expression) (placeSpec, error) {
	spec, err := t.compilePlaceBase(target)
	if err != nil {
		return spec, err
	}
	if isAggregate(spec.typ) {
		spec.into = t.aggregateFns(spec.typ).intoName
	}
	return spec, nil
}

func (t *LuaTarget) compilePlaceBase(target ast.Expression) (placeSpec, error) {
	typ := t.info.Types[target]

	switch e := target.(type) {
	case *ast.IndexExpr:
		tp := t.info.Types[e.Array]
		if isView(tp) || isPointer(tp) {
			base, err := t.compileExprScratch(e.Array)
			if err != nil {
				return placeSpec{}, err
			}
			index, err := t.compileExprScratch(e.Index)
			if err != nil {
				return placeSpec{}, err
			}
			kind := placeView
			if isPointer(tp) {
				kind = placePointerIndex
			}
			return placeSpec{kind: kind, base: base, index: index, typ: typ}, nil
		}
	case *ast.DerefExpr:
		base, err := t.compileExprScratch(e.Value)
		if err != nil {
			return placeSpec{}, err
		}
		return placeSpec{kind: placeDeref, base: base, typ: typ}, nil
	}

	text, err := t.compileExprScratch(target)
	if err != nil {
		return placeSpec{}, err
	}
	return placeSpec{kind: placeText, text: text, typ: typ}, nil
}

func (t *LuaTarget) stabilize(b *strings.Builder, s *placeSpec) {
	switch s.kind {
	case placeText:
		return
	case placeDeref:
		s.base = t.stableText(b, s.base)
	default:
		s.base = t.stableText(b, s.base)
		s.index = t.stableText(b, s.index)
	}
}

func (t *LuaTarget) stableText(b *strings.Builder, text string) string {
	if simpleText.MatchString(text) {
		return text
	}
	tmp := t.newLabel("place")
	fmt.Fprintf(b, "local %s = %s\n", tmp, text)
	return tmp
}

func (s placeSpec) get() string {
	switch s.kind {
	case placeView:
		return fmt.Sprintf("__wisp_vget(%s, %s)", s.base, s.index)
	case placePointerIndex:
		return fmt.Sprintf("__wisp_pget(%s, %s)", s.base, s.index)
	case placeDeref:
		return fmt.Sprintf("__wisp_get(%s)", s.base)
	}
	return s.text
}

func (s placeSpec) set(value string) string {
	if s.into != "" {
		return fmt.Sprintf("%s(%s, %s)\n", s.into, s.get(), value)
	}
	switch s.kind {
	case placeView:
		return fmt.Sprintf("__wisp_vset(%s, %s, %s)\n", s.base, s.index, value)
	case placePointerIndex:
		return fmt.Sprintf("__wisp_pset(%s, %s, %s)\n", s.base, s.index, value)
	case placeDeref:
		return fmt.Sprintf("__wisp_set(%s, %s)\n", s.base, value)
	}
	return fmt.Sprintf("%s = %s\n", s.text, value)
}
