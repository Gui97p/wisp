package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const languageServerName = "wisp-lsp"

var fileBuffer = map[string][]byte{}

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
		path := uriToPath(params.TextDocument.URI)
		fileBuffer[path] = []byte(params.TextDocument.Text)

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
		publish(context, uri)
		return nil
	}

	handler.TextDocumentDidSave = func(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
		publish(context, params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		delete(fileBuffer, uriToPath(params.TextDocument.URI))
		return nil
	}

	handler.TextDocumentHover = func(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
		return hover(params.TextDocument.URI, params.Position), nil
	}

	handler.TextDocumentDefinition = func(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
		loc := definition(params.TextDocument.URI, params.Position)
		if loc == nil {
			return nil, nil
		}
		return loc, nil
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
