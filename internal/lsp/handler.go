package lsp

import (
	"path/filepath"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/version"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const languageServerName = "wisp-lsp"

var serverVersion = version.Version()

var fileBuffer = map[string][]byte{}

func NewHandler() *protocol.Handler {
	handler := &protocol.Handler{}
	module.ConfigureTests(true)

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		capabilities := handler.CreateServerCapabilities()

		syncKind := protocol.TextDocumentSyncKindFull
		capabilities.TextDocumentSync = syncKind

		capabilities.CompletionProvider = &protocol.CompletionOptions{TriggerCharacters: []string{".", "{", ",", "[", "=", "\"", "/", " ", ":", "<", "#"}}

		capabilities.SignatureHelpProvider = &protocol.SignatureHelpOptions{TriggerCharacters: []string{"(", ","}}
		capabilities.DocumentSymbolProvider = true
		capabilities.SemanticTokensProvider = &protocol.SemanticTokensOptions{
			Legend: protocol.SemanticTokensLegend{TokenTypes: semanticTokenTypes, TokenModifiers: []string{}},
			Full:   true,
		}
		capabilities.DocumentFormattingProvider = true
		capabilities.ImplementationProvider = true
		capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: boolPtr(true)}
		capabilities.CodeActionProvider = protocol.CodeActionOptions{CodeActionKinds: []protocol.CodeActionKind{protocol.CodeActionKindQuickFix, protocol.CodeActionKindSourceOrganizeImports}}

		return protocol.InitializeResult{
			Capabilities: capabilities,
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name:    languageServerName,
				Version: &serverVersion,
			},
		}, nil
	}

	handler.TextDocumentDidOpen = func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		path := uriToPath(params.TextDocument.URI)
		fileBuffer[path] = []byte(params.TextDocument.Text)
		invalidateAnalysis()

		publish(context, params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		if len(params.ContentChanges) == 0 {
			return nil
		}

		change, ok := params.ContentChanges[len(params.ContentChanges)-1].(protocol.TextDocumentContentChangeEventWhole)
		if !ok {
			return nil
		}

		uri := params.TextDocument.URI
		fileBuffer[uriToPath(uri)] = []byte(change.Text)
		invalidateAnalysis()
		publish(context, uri)
		return nil
	}

	handler.TextDocumentDidSave = func(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
		invalidateAnalysis()
		publish(context, params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		delete(fileBuffer, uriToPath(params.TextDocument.URI))
		invalidateAnalysis()
		return nil
	}

	handler.TextDocumentFormatting = func(context *glsp.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
		return formatting(params.TextDocument.URI), nil
	}

	handler.TextDocumentHover = func(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
		if isConfigURI(params.TextDocument.URI) {
			return configHover(params.TextDocument.URI, params.Position), nil
		}
		return hover(params.TextDocument.URI, params.Position), nil
	}

	handler.TextDocumentDefinition = func(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
		locations := withoutBuiltin(definition(params.TextDocument.URI, params.Position))
		if len(locations) == 0 {
			return nil, nil
		}
		return locations, nil
	}

	handler.TextDocumentImplementation = func(context *glsp.Context, params *protocol.ImplementationParams) (any, error) {
		locations := implementation(params.TextDocument.URI, params.Position)
		if len(locations) == 0 {
			return nil, nil
		}
		return locations, nil
	}

	handler.TextDocumentSemanticTokensFull = semanticTokens

	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		trigger := ""
		if params.Context != nil && params.Context.TriggerCharacter != nil {
			trigger = *params.Context.TriggerCharacter
		}
		if isConfigURI(params.TextDocument.URI) {
			return configCompletion(params.TextDocument.URI, params.Position), nil
		}
		items := completion(params.TextDocument.URI, params.Position, trigger)
		return protocol.CompletionList{IsIncomplete: shortWord(params.TextDocument.URI, params.Position), Items: items}, nil
	}

	handler.TextDocumentDocumentSymbol = func(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
		return documentSymbols(params.TextDocument.URI), nil
	}

	handler.TextDocumentSignatureHelp = func(context *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
		return signatureHelp(params.TextDocument.URI, params.Position), nil
	}

	handler.TextDocumentPrepareRename = func(context *glsp.Context, params *protocol.PrepareRenameParams) (any, error) {
		if r := prepareRename(params.TextDocument.URI, params.Position); r != nil {
			return r, nil
		}
		return nil, nil
	}

	handler.TextDocumentRename = func(context *glsp.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
		return rename(params.TextDocument.URI, params.Position, params.NewName)
	}

	handler.TextDocumentCodeAction = func(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
		return codeActions(params.TextDocument.URI, params.Context.Diagnostics, params.Context.Only), nil
	}

	handler.Shutdown = func(context *glsp.Context) error {
		return nil
	}

	handler.Exit = func(context *glsp.Context) error {
		return nil
	}

	return handler
}

var published = map[string]map[string]bool{}

func publish(context *glsp.Context, uri protocol.DocumentUri) {
	path := uriToPath(uri)
	diagsByFile := diagnoseFile(path)

	root := module.FindProjectRoot(filepath.Dir(path))
	if root == "" {
		root = filepath.Dir(path)
	}
	if published[root] == nil {
		published[root] = map[string]bool{}
	}

	for file := range published[root] {
		if _, ok := diagsByFile[file]; !ok {
			context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
				URI:         pathToURI(file),
				Diagnostics: []protocol.Diagnostic{},
			})
			delete(published[root], file)
		}
	}

	for file, diags := range diagsByFile {
		context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         pathToURI(file),
			Diagnostics: diags,
		})
		published[root][file] = true
	}
}

func boolPtr(b bool) *bool { return &b }
