package lsp

import (
	"errors"
	"path/filepath"
	"slices"

	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func diagnoseFile(path string) map[string][]protocol.Diagnostic {
	result := map[string][]protocol.Diagnostic{
		path: {},
	}

	if file, diags, ok := configDiagnostics(path); ok {
		result[file] = diags
	}

	if filepath.Base(path) == "wisp.toml" {
		root := filepath.Dir(path)
		for open := range fileBuffer {
			if filepath.Ext(open) != ".wsp" || module.FindProjectRoot(filepath.Dir(open)) != root {
				continue
			}
			for file, diags := range diagnoseFile(open) {
				if file != path {
					result[file] = diags
				}
			}
			break
		}
		return result
	}

	pa, err := analyseProject(path)
	if err != nil || pa == nil {
		return result
	}

	for _, m := range pa.modules {
		mod := m.mod
		for _, f := range mod.FilePaths {
			if _, ok := result[f]; !ok {
				result[f] = []protocol.Diagnostic{}
			}
		}

		if mod.ParseError != nil {
			addDiagnostics(result, mod.ParseError.Path, mod.ParseError.Errors)
		}

		if mod.ParseError == nil && slices.Contains(mod.FilePaths, path) {
			setLastGood(path, analysis{m.merged, m.info})
		}

		if mod.ParseError == nil && len(m.diags) > 0 {
			addDiagnostics(result, "", m.diags)
		}
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
			endCol = uint32(d.EndCol-1) + 1
		}

		severity := protocol.DiagnosticSeverityError
		if d.Severity == diag.SeverityWarning {
			severity = protocol.DiagnosticSeverityWarning
		}
		rng := protocol.Range{
			Start: protocol.Position{Line: line, Character: col},
			End:   protocol.Position{Line: endLine, Character: endCol},
		}
		var related []protocol.DiagnosticRelatedInformation
		for _, n := range d.Notes {
			noteFile := n.File
			if noteFile == "" {
				noteFile = file
			}
			noteLine, noteCol := uint32(0), uint32(0)
			if n.Line > 0 {
				noteLine = uint32(n.Line - 1)
			}
			if n.Col > 0 {
				noteCol = uint32(n.Col - 1)
			}
			related = append(related, protocol.DiagnosticRelatedInformation{
				Location: protocol.Location{
					URI:   pathToURI(noteFile),
					Range: protocol.Range{Start: protocol.Position{Line: noteLine, Character: noteCol}, End: protocol.Position{Line: noteLine, Character: noteCol + 1}},
				},
				Message: n.Message,
			})
		}
		if d.Help != "" {
			related = append(related, protocol.DiagnosticRelatedInformation{
				Location: protocol.Location{URI: pathToURI(file), Range: rng},
				Message:  "help: " + d.Help,
			})
		}
		result[file] = append(result[file], protocol.Diagnostic{
			Range:              rng,
			Severity:           &severity,
			Message:            d.Message,
			RelatedInformation: related,
		})
	}
}

func configDiagnostics(path string) (string, []protocol.Diagnostic, bool) {
	s := loadSettings(path)
	if !s.hasConfig {
		return "", nil, false
	}

	diags := []protocol.Diagnostic{}
	err := s.configErr
	if err == nil {
		err = s.programErr
	}
	if err == nil {
		return s.tomlPath, diags, true
	}

	severity := protocol.DiagnosticSeverityError
	source := "wisp"
	issue := func(line int, message string) protocol.Diagnostic {
		start := uint32(0)
		if line > 0 {
			start = uint32(line - 1)
		}
		return protocol.Diagnostic{
			Range:    protocol.Range{Start: protocol.Position{Line: start}, End: protocol.Position{Line: start, Character: 1000}},
			Severity: &severity,
			Source:   &source,
			Message:  message,
		}
	}

	if configErr, ok := errors.AsType[*module.ConfigError](err); ok {
		for _, i := range configErr.Issues {
			problem := issue(i.Line, i.Message)
			if i.Help != "" {
				problem.RelatedInformation = []protocol.DiagnosticRelatedInformation{{
					Location: protocol.Location{URI: pathToURI(s.tomlPath), Range: problem.Range},
					Message:  "help: " + i.Help,
				}}
			}
			diags = append(diags, problem)
		}
		return s.tomlPath, diags, true
	}

	return s.tomlPath, append(diags, issue(0, err.Error())), true
}

func diagnosticText(d protocol.Diagnostic) string {
	text := d.Message
	for _, related := range d.RelatedInformation {
		text += "\n" + related.Message
	}
	return text
}
