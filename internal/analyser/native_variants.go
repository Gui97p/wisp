package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
)

func nativeKey(fd *ast.FuncDecl) string {
	if fd.Receiver != nil {
		return fd.Receiver.Type.String() + "." + fd.Name
	}
	return fd.Name
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

func (a *Analyser) selectNativeVariants() {
	groups := map[string][]*ast.FuncDecl{}
	var order []string
	hasNative := false
	for _, d := range a.program.Declarations {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		key := nativeKey(fd)
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], fd)
		if fd.NativeTarget != "" {
			hasNative = true
		}
	}
	if !hasNative {
		return
	}

	active := target.Active()
	drop := map[ast.Declaration]bool{}

	for _, key := range order {
		variants := groups[key]
		natives := 0
		for _, fd := range variants {
			if fd.NativeTarget != "" {
				natives++
			}
		}
		if natives == 0 {
			continue
		}

		first := variants[0]
		for _, fd := range variants[1:] {
			if signatureText(fd) != signatureText(first) || fd.Exported != first.Exported || fd.Reach != first.Reach {
				a.currentFile = a.declFiles[fd]
				at := fd.NameAt
				if at.Line == 0 {
					line, col := fd.Position()
					at = ast.Position{Line: line, Col: col, EndLine: line, EndCol: col}
				}
				a.errorRange(at, "%s must have the same signature and visibility in every version", fd.Name)
			}
		}

		plain := false
		var matching, versions []string
		for _, fd := range variants {
			if fd.NativeTarget == "" {
				plain = true
				continue
			}
			versions = append(versions, fd.NativeTarget)
			if target.MatchesText(active, fd.NativeTarget) {
				matching = append(matching, fd.NativeTarget)
			}
		}
		if len(matching) == 0 && !plain {
			if a.unavailable == nil {
				a.unavailable = map[string][]string{}
			}
			a.unavailable[first.Name] = versions
		}

		dominated := func(text string) bool {
			for _, other := range matching {
				if other != text && target.StrictlyNarrower(other, text) {
					return true
				}
			}
			return false
		}

		for _, fd := range variants {
			switch {
			case fd.NativeTarget == "":
				if len(matching) > 0 {
					drop[fd] = true
				}
			case !target.MatchesText(active, fd.NativeTarget), dominated(fd.NativeTarget):
				drop[fd] = true
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
