package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const languageServerName = "wisp-lsp"

func NewHandler() *protocol.Handler {
	handler := &protocol.Handler{}

	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		capabilities := handler.CreateServerCapabilities()

		syncKind := protocol.TextDocumentSyncKindFull
		capabilities.TextDocumentSync = syncKind

		return protocol.InitializeResult{
			Capabilities: capabilities,
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name: languageServerName,
			},
		}, nil
	}

	handler.TextDocumentDidOpen = func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		publish(context, params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentDidSave = func(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
		publish(context, params.TextDocument.URI)
		return nil
	}

	handler.Shutdown = func(context *glsp.Context) error {
		return nil
	}

	handler.Exit = func(context *glsp.Context) error {
		return nil
	}

	return handler
}

func publish(context *glsp.Context, uri protocol.DocumentUri) {
	path := uriToPath(uri)
	diagsByFile := diagnoseFile(path)

	for file, diags := range diagsByFile {
		context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         pathToURI(file),
			Diagnostics: diags,
		})
	}
}
