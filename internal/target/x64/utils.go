package x64

import (
	"fmt"
	"strconv"
	"strings"
)

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func decodeEscapes(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			out = append(out, s[i])
			continue
		}

		i++
		if i >= len(s) {
			return nil, fmt.Errorf("unterminated escape sequence")
		}

		switch s[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'v':
			out = append(out, '\v')
		case 'a':
			out = append(out, '\a')
		case '\\':
			out = append(out, '\\')
		case '"':
			out = append(out, '"')
		case '\'':
			out = append(out, '\'')
		case '0':
			out = append(out, 0)

		case 'x':
			if i+2 >= len(s) {
				return nil, fmt.Errorf("invalid hexadecimal escape")
			}

			hi, ok1 := hexValue(s[i+1])
			lo, ok2 := hexValue(s[i+2])
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("invalid hexadecimal escape")
			}

			out = append(out, hi<<4|lo)
			i += 2

		default:
			return nil, fmt.Errorf("unknown escape sequence: \\%c", s[i])
		}
	}

	return out, nil
}

func hexValue(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

func bytesToAsm(b []byte) string {
	var sb strings.Builder

	for i, v := range b {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(strconv.Itoa(int(v)))
	}

	return sb.String()
}
