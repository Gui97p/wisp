package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/target"
)

type variant struct {
	decl      ast.Declaration
	key       string
	names     []string
	kind      string
	selector  string
	at        ast.Position
	exported  bool
	reach     int
	signature string
}

func signatureText(fd *ast.FuncDecl) string {
	var b strings.Builder
	for _, p := range fd.Params {
		fmt.Fprintf(&b, "%s,", p.Type.String())
	}
	b.WriteString("->")
	for _, r := range fd.ReturnTypes {
		fmt.Fprintf(&b, "%s,", r.String())
	}
	if fd.Fallible {
		b.WriteString("!")
	}
	if fd.Receiver != nil {
		fmt.Fprintf(&b, "|%s", fd.Receiver.Type.String())
	}
	return b.String()
}

func declPosition(d ast.Declaration) ast.Position {
	line, col := d.Position()
	endLine, endCol := d.EndPosition()
	return ast.Position{Line: line, Col: col, EndLine: endLine, EndCol: endCol}
}

func describeVariant(d ast.Declaration) (variant, bool) {
	switch decl := d.(type) {
	case *ast.FuncDecl:
		key := "func:" + decl.Name
		if decl.Receiver != nil {
			key = "method:" + decl.Receiver.Type.String() + "." + decl.Name
		}
		at := decl.NameAt
		if at.Line == 0 {
			at = declPosition(d)
		}
		return variant{decl: d, key: key, names: []string{decl.Name}, kind: "function", selector: decl.Selector, at: at, exported: decl.Exported, reach: decl.Reach, signature: signatureText(decl)}, true
	case *ast.StructDecl:
		return variant{decl: d, key: "type:" + decl.Name, names: []string{decl.Name}, kind: "struct", selector: decl.Selector, at: declPosition(d), exported: decl.Exported, reach: decl.Reach}, true
	case *ast.TypeDecl:
		kind := "type"
		if decl.IsEnum {
			kind = "enum"
		}
		return variant{decl: d, key: "type:" + decl.Name, names: []string{decl.Name}, kind: kind, selector: decl.Selector, at: declPosition(d), exported: decl.Exported, reach: decl.Reach}, true
	case *ast.ImportDecl:
		return variant{decl: d, key: "import:" + decl.Alias, names: []string{decl.Alias}, kind: "import", selector: decl.Selector, at: declPosition(d)}, true
	case *ast.ConstDecl:
		names := make([]string, len(decl.Vars))
		for i, v := range decl.Vars {
			names[i] = v.Name
		}
		return variant{decl: d, key: "const:" + strings.Join(names, ","), names: names, kind: "constant", selector: decl.Selector, at: declPosition(d), exported: decl.Exported, reach: decl.Reach}, true
	}
	return variant{}, false
}

func (a *Analyser) selectVariants() {
	groups := map[string][]variant{}
	var order []string
	selected := false
	for _, d := range a.program.Declarations {
		v, ok := describeVariant(d)
		if !ok {
			continue
		}
		if _, seen := groups[v.key]; !seen {
			order = append(order, v.key)
		}
		groups[v.key] = append(groups[v.key], v)
		if v.selector != "" {
			selected = true
		}
	}
	if !selected {
		return
	}

	active := target.Active()
	drop := map[ast.Declaration]bool{}

	for _, key := range order {
		variants := groups[key]
		anySelector := false
		for _, v := range variants {
			if v.selector != "" {
				anySelector = true
			}
		}
		if !anySelector {
			continue
		}

		first := variants[0]
		for _, v := range variants[1:] {
			a.currentFile = a.declFiles[v.decl]
			switch {
			case v.kind != first.kind:
				a.errorRange(v.at, "%s must be the same kind of declaration in every version", first.names[0])
			case v.signature != first.signature:
				a.errorRange(v.at, "%s must have the same signature in every version", first.names[0])
			case v.exported != first.exported || v.reach != first.reach:
				a.errorRange(v.at, "%s must have the same visibility in every version", first.names[0])
			}
		}

		plain := false
		var matching, versions []string
		for _, v := range variants {
			if v.selector == "" {
				plain = true
				continue
			}
			versions = append(versions, v.selector)
			if target.MatchesText(active, v.selector) {
				matching = append(matching, v.selector)
			}
		}
		if len(matching) == 0 && !plain {
			if a.unavailable == nil {
				a.unavailable = map[string][]string{}
			}
			for _, name := range first.names {
				a.unavailable[name] = versions
			}
		}

		dominated := func(text string) bool {
			for _, other := range matching {
				if other != text && target.StrictlyNarrower(other, text) {
					return true
				}
			}
			return false
		}

		for _, v := range variants {
			switch {
			case v.selector == "":
				if len(matching) > 0 {
					drop[v.decl] = true
				}
			case !target.MatchesText(active, v.selector), dominated(v.selector):
				drop[v.decl] = true
			}
		}
	}

	if len(drop) == 0 {
		return
	}
	kept := a.program.Declarations[:0:0]
	for _, d := range a.program.Declarations {
		if !drop[d] {
			kept = append(kept, d)
		}
	}
	a.program.Declarations = kept
}

func (a *Analyser) unavailableHint(d *diag.Diagnostic, name string) bool {
	versions, missing := a.unavailable[name]
	if !missing || d == nil {
		return false
	}
	d.WithHelp("%s only has versions for %s, and this build is for %s", name, target.Enumerate(versions), target.Active())
	return true
}
