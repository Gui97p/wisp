package lsp

import (
	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func diagnoseFile(path string) map[string][]protocol.Diagnostic {
	result := map[string][]protocol.Diagnostic{
		path: {},
	}

	modules, err := module.BuildGraphWithOverrides(resolveProjectEntry(path), fileBuffer)
	if err != nil {
		return result
	}

	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		isEntry := false

		for _, f := range mod.FilePaths {
			if _, ok := result[f]; !ok {
				result[f] = []protocol.Diagnostic{}
			}
		}

		if mod.ParseError != nil {
			addDiagnostics(result, mod.ParseError.Path, mod.ParseError.Errors)
		}

		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, isEntry, exports, declFiles)
		a.Analyze()

		if a.HasErrors() {
			addDiagnostics(result, "", a.Errors())
		}

		exports[mod.Path] = &analyser.ModuleInfo{Exports: a.Exports()}
	}

	return result
}

func addDiagnostics(result map[string][]protocol.Diagnostic, fallbackFile string, diags diag.List) {
	for _, d := range diags {
		file := d.File
		if file == "" {
			file = fallbackFile
		}

		line := uint32(0)
		if d.Line > 0 {
			line = uint32(d.Line - 1)
		}
		col := uint32(0)
		if d.Col > 0 {
			col = uint32(d.Col - 1)
		}

		endLine := line
		if d.EndLine > 0 {
			endLine = uint32(d.EndLine - 1)
		}
		endCol := col + 1
		if d.EndCol > 0 {
			endCol = uint32(d.EndCol - 1) + 1
		}

		severity := protocol.DiagnosticSeverityError
		result[file] = append(result[file], protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: col},
				End:   protocol.Position{Line: endLine, Character: endCol},
			},
			Severity: &severity,
			Message:  d.Message,
		})
	}
}
