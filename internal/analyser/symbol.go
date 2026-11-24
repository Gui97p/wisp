package analyser

import "path/filepath"

type SymbolKind int

const (
	VAR SymbolKind = iota
	CONST
	PARAM
	FUNC
	STRUCT
	TYPE
	MODULE
	INTERFACE
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
	case INTERFACE:
		return "interface"
	default:
		return "unknown"
	}
}

type Symbol struct {
	Name string
	Type Type
	Kind SymbolKind

	Line   int
	Col    int
	File   string
	Module string

	Const  *ConstValue
	Global bool
	CName  string
	Extern bool
	Doc    string

	ScopeDir string
	Private  bool
	Emit     string

	ImportPath string

	MustUse bool
	Used    bool

	Deprecated *string

	Generic *GenericDecl
	Builtin bool
	Phantom string
}

func (s *Symbol) EmitName() string {
	if s.Emit != "" {
		return s.Emit
	}
	return s.Name
}

type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
	lambda  bool

	privates     map[string]map[string]*Symbol
	privateFiles map[string]int
	file         *string
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
	if s.privates != nil && s.file != nil && *s.file != "" {
		if symbol, ok := s.privates[filepath.Clean(*s.file)][name]; ok {
			return symbol, true
		}
	}
	if symbol, ok := s.symbols[name]; ok {
		return symbol, true
	}
	if s.parent != nil {
		return s.parent.Resolve(name)
	}
	return nil, false
}

func (s *Scope) captures(name string, module *Scope) bool {
	crossed := false
	for scope := s; scope != nil; scope = scope.parent {
		if _, ok := scope.symbols[name]; ok {
			return crossed && scope != module
		}
		if scope.lambda {
			crossed = true
		}
	}
	return false
}

func (s *Scope) Depths() map[*Symbol]int {
	depths := map[*Symbol]int{}
	seen := map[string]bool{}
	depth := 0
	for scope := s; scope != nil; scope = scope.parent {
		for name, sym := range scope.symbols {
			if seen[name] {
				continue
			}
			seen[name] = true
			depths[sym] = depth
		}
		depth++
	}
	return depths
}

func (s *Scope) Visible() []*Symbol {
	seen := map[string]bool{}
	var out []*Symbol
	for scope := s; scope != nil; scope = scope.parent {
		for name, sym := range scope.symbols {
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, sym)
		}
	}
	return out
}

func (s *Scope) BindFile(file *string) (restore func()) {
	saved := s.file
	s.file = file
	return func() { s.file = saved }
}

func (s *Scope) hasPrivate(name string) bool {
	return s.privateFiles[name] > 0
}

func (s *Scope) addPrivate(sym *Symbol) {
	if s.privates == nil {
		s.privates = map[string]map[string]*Symbol{}
		s.privateFiles = map[string]int{}
	}
	file := filepath.Clean(sym.File)
	if s.privates[file] == nil {
		s.privates[file] = map[string]*Symbol{}
	}
	s.privates[file][sym.Name] = sym
	s.privateFiles[sym.Name]++
}

func (s *Scope) topLevel(name, file string) (*Symbol, bool) {
	if sym, ok := s.privates[filepath.Clean(file)][name]; ok {
		return sym, true
	}
	sym, ok := s.symbols[name]
	return sym, ok
}
