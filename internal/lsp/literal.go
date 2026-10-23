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

func maskLiterals(text string) string {
	masked := []byte(text)
	state, start := stateCode, 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		blank := func(from, to int) {
			for j := from; j < to && j < len(masked); j++ {
				if masked[j] != '\n' {
					masked[j] = ' '
				}
			}
		}

		switch state {
		case stateCode:
			switch {
			case c == '/' && next == '/':
				state, start = stateLineComment, i
				i++
			case c == '/' && next == '*':
				state, start = stateBlockComment, i
				i++
			case c == '"':
				state, start = stateQuoted, i
			case c == '`':
				state, start = stateRaw, i
			case c == '\'':
				state, start = stateChar, i
			}
		case stateLineComment:
			if c == '\n' {
				blank(start, i)
				state = stateCode
			}
		case stateBlockComment:
			if c == '*' && next == '/' {
				i++
				blank(start, i+1)
				state = stateCode
			}
		case stateQuoted:
			switch {
			case c == '\\' && next == '"':
				i++
			case c == '"' || c == '\n':
				blank(start, i+1)
				state = stateCode
			}
		case stateRaw:
			switch {
			case c == '\\' && next == '`':
				i++
			case c == '`':
				blank(start, i+1)
				state = stateCode
			}
		case stateChar:
			switch c {
			case '\\':
				i++
			case '\'', '\n':
				blank(start, i+1)
				state = stateCode
			}
		}
	}
	if state != stateCode {
		for j := start; j < len(masked); j++ {
			if masked[j] != '\n' {
				masked[j] = ' '
			}
		}
	}
	return string(masked)
}
