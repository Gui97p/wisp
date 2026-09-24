package analyser

type SymbolKind int

const (
	VAR SymbolKind = iota
	CONST
	PARAM
	FUNC
	STRUCT
	TYPE
	MODULE
)

func (s SymbolKind) String() string {
	switch s {
	case VAR:
		return "variable"
	case CONST:
		return "constant"
	case PARAM:
		return "parameter"
	case FUNC:
		return "function"
	case STRUCT:
		return "struct"
	case TYPE:
		return "type"
	case MODULE:
		return "module"
	default:
		return "unknown"
	}
}

type Symbol struct {
	Name string
	Type Type
	Kind SymbolKind
	Line int
	Col  int
}

type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]*Symbol),
	}
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
