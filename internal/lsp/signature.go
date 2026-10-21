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
				ft, _ = ex.Type.(*analyser.FuncType)
			}
		} else {
			ft = analyser.MethodsOf(sym.Type)[member]
		}
	} else if sym, found := scope.Resolve(callee); found {
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
	if ft.Fallible {
		label += " !"
	}
	if len(ft.Returns) > 0 {
		rets := make([]string, len(ft.Returns))
		for i, r := range ft.Returns {
			rets[i] = r.String()
		}
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
