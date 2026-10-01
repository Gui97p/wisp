package ast

func NativeBackends(s Statement) []string {
	switch n := s.(type) {
	case *NativeStmt:
		return []string{n.Backend}
	case *BlockStmt:
		return nativeBackendsOf(n.Statements)
	case *GroupStmt:
		return nativeBackendsOf(n.Statements)
	case *IfStmt:
		backends := NativeBackends(n.Then)
		if n.Else != nil {
			backends = append(backends, NativeBackends(n.Else)...)
		}
		return backends
	case *ForStmt:
		return NativeBackends(n.Body)
	case *LoopStmt:
		return NativeBackends(n.Body)
	}

	return nil
}

func nativeBackendsOf(stmts []Statement) []string {
	var backends []string
	for _, s := range stmts {
		backends = append(backends, NativeBackends(s)...)
	}

	return backends
}
