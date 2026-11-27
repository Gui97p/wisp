package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/parser"
)

const castPlaceholder = "__WispCastType"

const optionalCastTemplate = "{ ?*" + castPlaceholder + " wispcast; match type wispsubject { *" + castPlaceholder + " wispfound => wispcast = wispfound; _ => {} } yield wispcast; }"

const assertedCastTemplate = "{ ?*" + castPlaceholder + " wispcast; match type wispsubject { *" + castPlaceholder + " wispfound => wispcast = wispfound; _ => {} } yield wispcast?; }"

func (a *Analyser) lowerInterfaceCast(expr *ast.ConvExpr, arg ast.Expression, held *InterfaceType, pointer PointerType, result Type) (Type, bool) {
	if isVoidPointer(pointer) {
		return nil, false
	}
	if _, concrete := pointer.Element.(*StructType); !concrete && !isNamedConcrete(pointer.Element) {
		return nil, false
	}

	plain := PointerType{Element: pointer.Element}
	if reason := a.satisfiesInterface(plain, held); reason != "" {
		a.errorf(expr, "%s can never be what %s holds", plain, held.Name).WithHelp("%s", reason)
		return result, true
	}

	template := assertedCastTemplate
	if pointer.Nullable {
		template = optionalCastTemplate
	}
	parsed, problem := parser.ParseExpressionText(template)
	if parsed == nil {
		a.errorf(expr, "internal error lowering a conversion from %s: %s", held.Name, problem)
		return result, true
	}

	hidden := map[string]string{"wispcast": "__wisp_cast", "wispfound": "__wisp_found", "wispsubject": "__wisp_subject"}
	ast.Inspect(parsed, func(n ast.Node) bool {
		a.at(expr, n)
		switch node := n.(type) {
		case *ast.IdentLiteral:
			if name, ok := hidden[node.Value]; ok {
				node.Value = name
			}
		case *ast.VarStmt:
			for i := range node.Vars {
				if name, ok := hidden[node.Vars[i].Name]; ok {
					node.Vars[i].Name = name
				}
			}
		case *ast.BinaryExpr:
			if node.Bind != nil {
				if name, ok := hidden[node.Bind.Name]; ok {
					node.Bind.Name = name
				}
			}
		}
		return true
	})

	element := expr.Type
	element.Layers = nil
	for _, layer := range expr.Type.Layers {
		if layer.Optional {
			continue
		}
		element.Layers = append(element.Layers, layer)
	}
	optional := expr.Type
	optional.Layers = append([]ast.TypeLayer{{Optional: true}}, element.Layers...)

	lowered := ast.Clone(parsed, ast.CloneOptions{Replace: func(n ast.Node) (ast.Node, bool) {
		if id, ok := n.(*ast.IdentLiteral); ok && id.Value == "__wisp_subject" {
			return ast.Clone(arg, ast.CloneOptions{}), true
		}
		return nil, false
	}})

	ast.Inspect(lowered, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeLabel:
			if node.Type.Name == castPlaceholder {
				node.Type.Module, node.Type.Name, node.Type.Args, node.Type.Layers = element.Module, element.Name, element.Args, element.Layers
			}
		case *ast.VarStmt:
			for i := range node.Vars {
				if node.Vars[i].Type.Name == castPlaceholder {
					node.Vars[i].Type.Module, node.Vars[i].Type.Name, node.Vars[i].Type.Args, node.Vars[i].Type.Layers = optional.Module, optional.Name, optional.Args, optional.Layers
				}
			}
		}
		return true
	})

	got := a.checkLowered(lowered)
	expr.Lowered = lowered
	if isInvalidType(got) {
		return InvalidType{}, true
	}
	return result, true
}
