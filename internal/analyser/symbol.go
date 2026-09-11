package analyser

type SymbolKind int

const (
	VAR SymbolKind = iota
	PARAM
	FUNC
	STRUCT
)

type Symbol struct {
	Name string
	Type Type
	Kind SymbolKind
}

type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

func (s *Scope) Define(symbol *Symbol) bool {
	if _, ok := s.symbols[symbol.Name]; ok {
		return false
	}
	s.symbols[symbol.Name] = symbol
	return true
}

func (s *Scope) Resolve(name string) (*Symbol, bool) {
	if symbol, ok := s.symbols[name]; ok {
		return symbol, true
	}
	if s.parent != nil {
		return s.parent.Resolve(name)
	}
	return nil, false
}
