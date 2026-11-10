package analyser

import (
	"errors"
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
		if fd.NativeArch != "" {
			hasNative = true
		}
	}
	if !hasNative {
		return
	}

	arch := target.ActiveArch()
	drop := map[ast.Declaration]bool{}

	for _, key := range order {
		variants := groups[key]
		natives := 0
		for _, fd := range variants {
			if fd.NativeArch == "" {
				continue
			}
			natives++
			a.currentFile = a.declFiles[fd]
			if err := target.ValidateArch(fd.NativeArch); err != nil {
				d := a.errorRange(fd.NativeAt, "unknown architecture %q in a native function", fd.NativeArch)
				if unknown, ok := errors.AsType[*target.UnknownError](err); ok {
					d.WithHelp("%s", unknown.Help)
				}
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

		matches, plain := false, false
		var archs []string
		for _, fd := range variants {
			if fd.NativeArch != "" && fd.NativeArch == arch {
				matches = true
			}
			if fd.NativeArch == "" {
				plain = true
			} else {
				archs = append(archs, fd.NativeArch)
			}
		}
		if !matches && !plain {
			if a.unavailable == nil {
				a.unavailable = map[string][]string{}
			}
			a.unavailable[first.Name] = archs
		}
		for _, fd := range variants {
			switch {
			case fd.NativeArch != "" && fd.NativeArch != arch:
				drop[fd] = true
			case fd.NativeArch == "" && matches:
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
