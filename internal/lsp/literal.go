package lsp

const (
	stateCode = iota
	stateLineComment
	stateBlockComment
	stateQuoted
	stateRaw
	stateChar
)

func insideLiteralOrComment(text string) bool {
	state, _ := scanState(text)
	return state != stateCode
}

func openStringStart(text string) (int, bool) {
	state, start := scanState(text)
	if state == stateQuoted || state == stateRaw {
		return start, true
	}
	return 0, false
}

func scanState(text string) (int, int) {
	const (
		code         = stateCode
		lineComment  = stateLineComment
		blockComment = stateBlockComment
		quoted       = stateQuoted
		raw          = stateRaw
		char         = stateChar
	)

	state := code
	start := 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		switch state {
		case code:
			switch {
			case c == '/' && next == '/':
				state = lineComment
				i++
			case c == '/' && next == '*':
				state = blockComment
				i++
			case c == '"':
				state = quoted
				start = i
			case c == '`':
				state = raw
				start = i
			case c == '\'':
				state = char
			}
		case lineComment:
			if c == '\n' {
				state = code
			}
		case blockComment:
			if c == '*' && next == '/' {
				state = code
				i++
			}
		case quoted:
			switch {
			case c == '\\' && next == '"':
				i++
			case c == '"' || c == '\n':
				state = code
			}
		case raw:
			switch {
			case c == '\\' && next == '`':
				i++
			case c == '`':
				state = code
			}
		case char:
			switch {
			case c == '\\':
				i++
			case c == '\'' || c == '\n':
				state = code
			}
		}
	}
	return state, start
}
