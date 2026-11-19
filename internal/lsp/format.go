package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"unicode/utf16"

	"github.com/Gui97p/wisp/internal/format"
	"github.com/Gui97p/wisp/internal/module"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func formatting(uri protocol.DocumentUri) []protocol.TextEdit {
	path := uriToPath(uri)
	if filepath.Ext(path) != ".wsp" {
		return nil
	}
	src, ok := fileBuffer[path]
	if !ok {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		src = data
	}

	root := module.FindProjectRoot(filepath.Dir(path))
	if cfg, err := module.LoadConfig(root); err == nil && format.Excluded(root, cfg.Fmt.Exclude, path) {
		return nil
	}

	out, err := format.Source(src)
	if err != nil || bytes.Equal(out, src) {
		return nil
	}
	return []protocol.TextEdit{{Range: wholeDocument(src), NewText: string(out)}}
}

func wholeDocument(src []byte) protocol.Range {
	line := bytes.Count(src, []byte("\n"))
	last := src[bytes.LastIndexByte(src, '\n')+1:]
	character := len(utf16.Encode([]rune(string(last))))
	return protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(character)},
	}
}
