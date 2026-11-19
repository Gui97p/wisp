package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/parser"
)

func Source(src []byte) ([]byte, error) {
	p := parser.NewParser(lexer.NewLexer(src))
	p.ParseProgram()
	if p.HasErrors() {
		first := p.Errors()[0]
		return nil, fmt.Errorf("%d:%d: %s", first.Line, first.Col, first.Message)
	}

	tokens, err := scan(src)
	if err != nil {
		return nil, err
	}
	gaps, err := readGaps(src, tokens)
	if err != nil {
		return nil, err
	}

	tokens, gaps = sortImports(tokens, gaps)
	roles := classify(tokens)
	pr := &printer{src: src, tokens: tokens, roles: roles, gaps: gaps, natives: nativeBlocks(tokens)}
	out := pr.print()

	if err := verify(out, tokens, gaps); err != nil {
		return nil, fmt.Errorf("the formatter would change the program (%v): this is a bug, the file was left alone", err)
	}
	return out, nil
}

func verify(out []byte, tokens []token, gaps []gap) error {
	again, err := scan(out)
	if err != nil {
		return err
	}
	if len(again) != len(tokens) {
		return fmt.Errorf("%d tokens became %d", len(tokens), len(again))
	}
	for i := range tokens {
		if tokens[i].kind != again[i].kind || tokens[i].text != again[i].text {
			return fmt.Errorf("token %q became %q", tokens[i].text, again[i].text)
		}
	}
	gapsAgain, err := readGaps(out, again)
	if err != nil {
		return err
	}
	var before, after []string
	for _, g := range gaps {
		for _, c := range g.comments {
			before = append(before, c.text)
		}
	}
	for _, g := range gapsAgain {
		for _, c := range g.comments {
			after = append(after, c.text)
		}
	}
	if strings.Join(before, "\x00") != strings.Join(after, "\x00") {
		return fmt.Errorf("the comments changed")
	}
	return nil
}

func sortImports(tokens []token, gaps []gap) ([]token, []gap) {
	type unit struct {
		first, last int
		path        string
	}

	var runs [][]unit
	var current []unit
	flush := func() {
		if len(current) > 1 {
			runs = append(runs, current)
		}
		current = nil
	}

	depth := 0
	for i := 0; i < len(tokens); {
		t := tokens[i]
		switch t.kind {
		case lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET, lexer.TOKEN_LBRACE:
			depth++
		case lexer.TOKEN_RPAREN, lexer.TOKEN_RBRACKET, lexer.TOKEN_RBRACE:
			depth--
		}
		if t.kind != lexer.TOKEN_IMPORT || depth != 0 {
			flush()
			i++
			continue
		}
		j := i
		path := ""
		clean := true
		for j < len(tokens) && tokens[j].kind != lexer.TOKEN_SEMICOLON {
			if tokens[j].kind == lexer.TOKEN_STRING_LITERAL && path == "" {
				path = strings.Trim(tokens[j].text, "\"")
			}
			if j > i && len(gaps[j].comments) > 0 {
				clean = false
			}
			j++
		}
		if j >= len(tokens) {
			flush()
			break
		}
		if len(current) > 0 && len(gaps[i].comments) > 0 {
			flush()
		}
		if !clean {
			flush()
			i = j + 1
			continue
		}
		current = append(current, unit{first: i, last: j, path: path})
		i = j + 1
	}
	flush()

	if len(runs) == 0 {
		return tokens, gaps
	}

	outTokens := make([]token, 0, len(tokens))
	outGaps := make([]gap, 0, len(gaps))
	pos := 0
	for _, run := range runs {
		for pos < run[0].first {
			outTokens = append(outTokens, tokens[pos])
			outGaps = append(outGaps, gaps[pos])
			pos++
		}
		sorted := append([]unit(nil), run...)
		sort.SliceStable(sorted, func(a, b int) bool {
			sa, sb := strings.HasPrefix(sorted[a].path, "std/"), strings.HasPrefix(sorted[b].path, "std/")
			if sa != sb {
				return sa
			}
			return sorted[a].path < sorted[b].path
		})
		for n, u := range sorted {
			for k := u.first; k <= u.last; k++ {
				outTokens = append(outTokens, tokens[k])
				switch {
				case k == u.first && n == 0:
					outGaps = append(outGaps, gaps[run[0].first])
				case k == u.first:
					outGaps = append(outGaps, gap{nlAfter: 1})
				default:
					outGaps = append(outGaps, gaps[k])
				}
			}
		}
		pos = run[len(run)-1].last + 1
	}
	for pos < len(tokens) {
		outTokens = append(outTokens, tokens[pos])
		outGaps = append(outGaps, gaps[pos])
		pos++
	}
	outGaps = append(outGaps, gaps[len(tokens)])
	return outTokens, outGaps
}

func nativeBlocks(tokens []token) map[int]int {
	blocks := map[int]int{}
	for i, t := range tokens {
		if t.kind != lexer.TOKEN_NATIVE {
			continue
		}
		paren := 0
		open := -1
		for j := i + 1; j < len(tokens); j++ {
			switch tokens[j].kind {
			case lexer.TOKEN_LPAREN:
				paren++
			case lexer.TOKEN_RPAREN:
				paren--
			case lexer.TOKEN_SEMICOLON:
				if paren == 0 {
					open = -2
				}
			case lexer.TOKEN_LBRACE:
				if paren == 0 {
					open = j
				}
			}
			if open != -1 {
				break
			}
		}
		if open < 0 {
			continue
		}
		depth := 0
		for j := open; j < len(tokens); j++ {
			switch tokens[j].kind {
			case lexer.TOKEN_LBRACE:
				depth++
			case lexer.TOKEN_RBRACE:
				depth--
				if depth == 0 {
					blocks[open] = j
					j = len(tokens)
				}
			}
		}
	}
	return blocks
}
