package lsp

import "strings"

func splitLines(src []byte) []string {
	return strings.Split(strings.ReplaceAll(string(src), "\r", ""), "\n")
}

func bufferLines(path string) ([]string, bool) {
	src, ok := fileBuffer[path]
	if !ok {
		return nil, false
	}
	return splitLines(src), true
}

func splitAtCursor(path string, line, character int) (before, after string, lines []string, ok bool) {
	lines, ok = bufferLines(path)
	if !ok || line < 0 || line >= len(lines) {
		return "", "", nil, false
	}
	runes := []rune(lines[line])
	character = min(max(character, 0), len(runes))
	return string(runes[:character]), string(runes[character:]), lines, true
}

func linePrefix(path string, line, character int) string {
	before, _, _, _ := splitAtCursor(path, line, character)
	return before
}

func lineSuffix(path string, line, character int) string {
	_, after, _, _ := splitAtCursor(path, line, character)
	return after
}

func textBefore(path string, line, character int) string {
	before, _, lines, ok := splitAtCursor(path, line, character)
	if !ok {
		return ""
	}
	return strings.Join(lines[:line], "\n") + "\n" + before
}
