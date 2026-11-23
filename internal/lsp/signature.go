package lsp

import (
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func callContext(prefix string) (callee string, arg int, ok bool) {
	depth := 0
	arg = 0
	for i := len(prefix) - 1; i >= 0; i-- {
		switch prefix[i] {
		case ')', ']', '}':
			depth++
		case '[', '{':
			if depth == 0 {
				return "", 0, false
			}
			depth--
		case ',':
			if depth == 0 {
				arg++
			}
		case '(':
			if depth > 0 {
				depth--
				continue
			}
			end := i
			if end > 0 && prefix[end-1] == '>' {
				angle := 0
				for j := end - 1; j >= 0; j-- {
					if prefix[j] == '>' {
						angle++
					} else if prefix[j] == '<' {
						angle--
						if angle == 0 {
							end = j
							break
						}
					}
				}
			}
			start := end
			for start > 0 && (isIdentByte(prefix[start-1]) || prefix[start-1] == '.') {
				start--
			}
			if start == end {
				return "", 0, false
			}
			return prefix[start:end], arg, true
		}
	}
	return "", 0, false
}

func signatureHelp(uri protocol.DocumentUri, pos protocol.Position) *protocol.SignatureHelp {
	path := uriToPath(uri)
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))

	if insideLiteralOrComment(textBefore(path, int(pos.Line), int(pos.Character))) {
		return nil
	}

	callee, arg, ok := callContext(prefix)
	if !ok {
		return nil
	}

	info := currentInfo(path)
	if info == nil {
		return nil
	}

	scope := scopeAt(info, int(pos.Line)+1, int(pos.Character)+1)

	var ft *analyser.FuncType
	name := callee
	if base, member, dotted := strings.Cut(callee, "."); dotted {
		sym, found := scope.Resolve(base)
		if !found {
			return nil
		}
		name = member
		if mt, isModule := sym.Type.(*analyser.ModuleType); isModule {
			if ex, found := mt.Exports[member]; found {
				if help := genericSignature(ex, member, arg); help != nil {
					return help
				}
				ft, _ = ex.Type.(*analyser.FuncType)
			}
		} else {
			ft = analyser.MethodsOf(sym.Type)[member]
		}
	} else if sym, found := scope.Resolve(callee); found {
		if help, builtin := isBuiltin(sym); builtin {
			return builtinSignature(callee, help, arg)
		}
		if help := genericSignature(sym, callee, arg); help != nil {
			return help
		}
		ft, _ = sym.Type.(*analyser.FuncType)
	}
	if ft == nil {
		return nil
	}

	label := name + "("
	labels := analyser.ParamLabels(ft, nil)
	params := make([]protocol.ParameterInformation, 0, len(labels))
	for i, text := range labels {
		if i > 0 {
			label += ", "
		}
		start := uint32(len([]rune(label)))
		label += text
		params = append(params, protocol.ParameterInformation{Label: [2]uint32{start, uint32(len([]rune(label)))}})
	}
	label += ")"
	rets := make([]string, len(ft.Returns))
	for i, r := range ft.Returns {
		rets[i] = r.String()
	}
	switch {
	case ft.Fallible:
		label += " !" + strings.Join(rets, ", ")
	case len(rets) > 0:
		label += " " + strings.Join(rets, ", ")
	}

	active := uint32(arg)
	if active >= uint32(len(params)) && len(params) > 0 {
		active = uint32(len(params) - 1)
	}

	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{{Label: label, Parameters: params}},
		ActiveSignature: new(uint32),
		ActiveParameter: &active,
	}
}

func builtinSignature(name string, help builtinHelp, arg int) *protocol.SignatureHelp {
	label := name + "("
	params := make([]protocol.ParameterInformation, 0, len(help.params))
	for i, text := range help.params {
		if i > 0 {
			label += ", "
		}
		start := uint32(len([]rune(label)))
		label += text
		params = append(params, protocol.ParameterInformation{Label: [2]uint32{start, uint32(len([]rune(label)))}})
	}
	label += ")"
	if help.returns != "" {
		label += " " + help.returns
	}

	active := uint32(arg)
	if active >= uint32(len(params)) && len(params) > 0 {
		active = uint32(len(params) - 1)
	}
	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{{Label: label, Parameters: params, Documentation: help.doc}},
		ActiveSignature: new(uint32),
		ActiveParameter: &active,
	}
}

func genericSignature(sym *analyser.Symbol, name string, arg int) *protocol.SignatureHelp {
	if sym == nil || sym.Generic == nil || sym.Generic.Func == nil {
		return nil
	}
	if _, isGeneric := sym.Type.(analyser.GenericType); !isGeneric {
		return nil
	}
	fd := sym.Generic.Func

	label := name + typeParamsText(fd.TypeParams) + "("
	params := make([]protocol.ParameterInformation, 0, len(fd.Params))
	for i, p := range fd.Params {
		if i > 0 {
			label += ", "
		}
		text := p.Name
		switch {
		case p.Any:
			text = "..." + p.Name
		case p.Variadic:
			text = refText(p.Type) + " ..." + p.Name
		default:
			text = refText(p.Type) + " " + p.Name
		}
		start := uint32(len([]rune(label)))
		label += text
		params = append(params, protocol.ParameterInformation{Label: [2]uint32{start, uint32(len([]rune(label)))}})
	}
	label += ")"
	results := make([]string, len(fd.ReturnTypes))
	for i := range fd.ReturnTypes {
		results[i] = refText(fd.ReturnTypes[i])
	}
	switch {
	case fd.Fallible:
		label += " !" + strings.Join(results, ", ")
	case len(results) > 0:
		label += " " + strings.Join(results, ", ")
	}

	active := uint32(arg)
	if active >= uint32(len(params)) && len(params) > 0 {
		active = uint32(len(params) - 1)
	}
	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{{Label: label, Parameters: params, Documentation: sym.Doc}},
		ActiveSignature: new(uint32),
		ActiveParameter: &active,
	}
}
