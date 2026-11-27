package lsp

import (
	"regexp"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var typeCaseTyping = regexp.MustCompile(`^\s*(?:[A-Za-z0-9_.*]+\s*,\s*)*[A-Za-z0-9_]*$`)

var switchTypeHeader = regexp.MustCompile(`\bmatch\s+type\b[^{}]*$`)

func insideTypeSwitch(text string) bool {
	depth := 0
	for i := len(text) - 1; i >= 0; i-- {
		switch text[i] {
		case '}':
			depth++
		case '{':
			if depth == 0 {
				start := strings.LastIndex(text[:i], "\n") + 1
				return switchTypeHeader.MatchString(text[start:i])
			}
			depth--
		}
	}
	return false
}

func typeGroupItems() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		item("signed", protocol.CompletionItemKindClass, "group: every signed integer, v is an int"),
		item("unsigned", protocol.CompletionItemKindClass, "group: every unsigned integer, v is a uint"),
	}
}
