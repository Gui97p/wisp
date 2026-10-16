package lsp

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/module"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func diagnoseFile(path string) (result map[string][]protocol.Diagnostic) {
	result = map[string][]protocol.Diagnostic{
		path: {},
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("analyser panic: %v", r)
		}
	}()

	if file, diags, ok := configDiagnostics(resolveProjectEntry(path)); ok {
		result[file] = diags
	}

	modules, err := buildProjectGraph(resolveProjectEntry(path), fileBuffer)
	if err != nil {
		return result
	}

	exports := map[string]*analyser.ModuleInfo{}

	for _, mod := range modules {
		for _, f := range mod.FilePaths {
			if _, ok := result[f]; !ok {
				result[f] = []protocol.Diagnostic{}
			}
		}

		if mod.ParseError != nil {
			addDiagnostics(result, mod.ParseError.Path, mod.ParseError.Errors)
		}

		merged, declFiles := mod.Merge()
		a := analyser.NewAnalyser(merged, exports, declFiles, mod.Path)
		a.SetImportErrors(mod.ImportErrors)
		info := a.Analyze()
		if mod.ParseError == nil && slices.Contains(mod.FilePaths, path) {
			setLastGood(path, analysis{merged, info})
		}

		if a.HasDiagnostics() {
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
			endCol = uint32(d.EndCol-1) + 1
		}

		severity := protocol.DiagnosticSeverityError
		if d.Severity == diag.SeverityWarning {
			severity = protocol.DiagnosticSeverityWarning
		}
		message := d.Message
		for _, n := range d.Notes {
			message += fmt.Sprintf("\nnote: %s (%d:%d)", n.Message, n.Line, n.Col)
		}
		if d.Help != "" {
			message += "\nhelp: " + d.Help
		}
		result[file] = append(result[file], protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: col},
				End:   protocol.Position{Line: endLine, Character: endCol},
			},
			Severity: &severity,
			Message:  message,
		})
	}
}

func configDiagnostics(entry string) (string, []protocol.Diagnostic, bool) {
	root := module.FindProjectRoot(filepath.Dir(entry))
	tomlPath := filepath.Join(root, "wisp.toml")
	if _, err := os.Stat(tomlPath); err != nil {
		return "", nil, false
	}

	diags := []protocol.Diagnostic{}
	cfg, err := module.LoadConfig(root)
	if err == nil {
		_, _, err = cfg.StdlibSettings(root)
	}
	if err == nil {
		return tomlPath, diags, true
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

	var configErr *module.ConfigError
	if errors.As(err, &configErr) {
		for _, i := range configErr.Issues {
			diags = append(diags, issue(i.Line, i.Message))
		}
		return tomlPath, diags, true
	}

	return tomlPath, append(diags, issue(0, err.Error())), true
}
