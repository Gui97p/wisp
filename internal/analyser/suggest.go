package analyser

import (
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/diag"
)

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func closestName(name string, candidates []string) (string, bool) {
	limit := max(1, len([]rune(name))/3)
	best, bestDistance := "", limit+1
	for _, c := range candidates {
		if c == name || c == "" {
			continue
		}
		d := editDistance(strings.ToLower(name), strings.ToLower(c))
		if d < bestDistance || (d == bestDistance && c < best) {
			best, bestDistance = c, d
		}
	}
	return best, bestDistance <= limit
}

func (a *Analyser) suggest(d *diag.Diagnostic, name string, candidates []string) {
	if d == nil {
		return
	}
	if best, ok := closestName(name, candidates); ok {
		d.WithHelp("did you mean %s?", best)
	}
}

func (s *Scope) names(accept func(*Symbol) bool) []string {
	seen := map[string]bool{}
	var out []string
	for scope := s; scope != nil; scope = scope.parent {
		for name, sym := range scope.symbols {
			if seen[name] || (accept != nil && !accept(sym)) {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func exportNames(exports map[string]*Symbol) []string {
	out := make([]string, 0, len(exports))
	for name := range exports {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (a *Analyser) notePrevious(d *diag.Diagnostic, sym *Symbol, what string) {
	if d == nil || sym == nil || sym.Line == 0 {
		return
	}
	d.Note(sym.File, sym.Line, sym.Col, sym.Line, sym.Col+max(len(sym.Name), 1)-1, "%s", what)
}
