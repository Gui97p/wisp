package target

import (
	"fmt"
	"strings"
	"sync"
)

type selectorKind int

const (
	selAtom selectorKind = iota
	selNot
	selAnd
	selOr
)

type Selector struct {
	kind  selectorKind
	word  string
	left  *Selector
	right *Selector
}

type selectorParser struct {
	text string
	pos  int
}

func ParseSelector(text string) (*Selector, error) {
	p := &selectorParser{text: text}
	sel, err := p.or()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.pos < len(p.text) {
		return nil, fmt.Errorf("unexpected %q in the target expression", p.text[p.pos:])
	}
	return sel, nil
}

func (p *selectorParser) skip() {
	for p.pos < len(p.text) && p.text[p.pos] == ' ' {
		p.pos++
	}
}

func (p *selectorParser) accept(op string) bool {
	p.skip()
	if strings.HasPrefix(p.text[p.pos:], op) {
		p.pos += len(op)
		return true
	}
	return false
}

func (p *selectorParser) or() (*Selector, error) {
	left, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.accept("||") {
		right, err := p.and()
		if err != nil {
			return nil, err
		}
		left = &Selector{kind: selOr, left: left, right: right}
	}
	return left, nil
}

func (p *selectorParser) and() (*Selector, error) {
	left, err := p.not()
	if err != nil {
		return nil, err
	}
	for p.accept("&&") {
		right, err := p.not()
		if err != nil {
			return nil, err
		}
		left = &Selector{kind: selAnd, left: left, right: right}
	}
	return left, nil
}

func (p *selectorParser) not() (*Selector, error) {
	if p.accept("!") {
		inner, err := p.not()
		if err != nil {
			return nil, err
		}
		return &Selector{kind: selNot, left: inner}, nil
	}
	if p.accept("(") {
		inner, err := p.or()
		if err != nil {
			return nil, err
		}
		if !p.accept(")") {
			return nil, fmt.Errorf("a ( in the target expression is never closed")
		}
		return inner, nil
	}

	p.skip()
	start := p.pos
	for p.pos < len(p.text) {
		c := p.text[p.pos]
		if c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			p.pos++
			continue
		}
		break
	}
	if start == p.pos {
		return nil, fmt.Errorf("expected an architecture or a system in the target expression")
	}
	return &Selector{kind: selAtom, word: p.text[start:p.pos]}, nil
}

func (s *Selector) Matches(t Target) bool {
	switch s.kind {
	case selAtom:
		return t.Matches(s.word)
	case selNot:
		return !s.left.Matches(t)
	case selAnd:
		return s.left.Matches(t) && s.right.Matches(t)
	default:
		return s.left.Matches(t) || s.right.Matches(t)
	}
}

func (s *Selector) words(out *[]string) {
	if s.kind == selAtom {
		*out = append(*out, s.word)
		return
	}
	if s.left != nil {
		s.left.words(out)
	}
	if s.right != nil {
		s.right.words(out)
	}
}

var selectorCache sync.Map

func cachedSelector(text string) (*Selector, error) {
	if cached, ok := selectorCache.Load(text); ok {
		entry := cached.(selectorEntry)
		return entry.sel, entry.err
	}
	sel, err := ParseSelector(text)
	selectorCache.Store(text, selectorEntry{sel: sel, err: err})
	return sel, err
}

type selectorEntry struct {
	sel *Selector
	err error
}

func ValidateSelectorText(text string) error {
	sel, err := cachedSelector(text)
	if err != nil {
		return err
	}
	var words []string
	sel.words(&words)
	known := Selectors()
	for _, w := range words {
		found := false
		for _, k := range known {
			if k == w {
				found = true
				break
			}
		}
		if !found {
			return &UnknownError{Name: w, Help: Suggestion(w, known, "architectures and systems")}
		}
	}
	return nil
}

func MatchesText(targetName, text string) bool {
	t, err := Lookup(targetName)
	if err != nil {
		return false
	}
	sel, err := cachedSelector(text)
	if err != nil {
		return false
	}
	return sel.Matches(t)
}

func universe() []Target {
	registered := all()
	universe := append([]Target{}, registered...)

	archs, oses := map[string]bool{}, map[string]bool{"": true}
	for _, t := range registered {
		archs[t.Arch] = true
		oses[t.OS] = true
	}
	for arch := range archs {
		for os := range oses {
			present := false
			for _, t := range registered {
				if t.Arch == arch && t.OS == os {
					present = true
					break
				}
			}
			if !present {
				universe = append(universe, Target{Arch: arch, OS: os})
			}
		}
	}
	return universe
}

func matchedSet(text string, world []Target) ([]bool, bool) {
	sel, err := cachedSelector(text)
	if err != nil {
		return nil, false
	}
	set := make([]bool, len(world))
	for i, t := range world {
		set[i] = sel.Matches(t)
	}
	return set, true
}

func StrictlyNarrower(narrow, wide string) bool {
	world := universe()
	a, okA := matchedSet(narrow, world)
	b, okB := matchedSet(wide, world)
	if !okA || !okB {
		return false
	}
	smaller := false
	for i := range world {
		if a[i] && !b[i] {
			return false
		}
		if !a[i] && b[i] {
			smaller = true
		}
	}
	return smaller
}

func FirstMatching(text string) (string, bool) {
	sel, err := cachedSelector(text)
	if err != nil {
		return "", false
	}
	for _, t := range all() {
		if sel.Matches(t) {
			return t.Name, true
		}
	}
	return "", false
}
