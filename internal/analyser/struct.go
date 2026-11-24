package analyser

import (
	"path/filepath"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) registerStructNames() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		if td, isType := d.(*ast.TypeDecl); isType && len(td.TypeParams) > 0 {
			if td.Data {
				a.registerGenericSum(td)
				continue
			}
			a.registerGenericAlias(td)
			continue
		}
		if td, isEnum := d.(*ast.TypeDecl); isEnum && td.Data {
			a.registerSumName(td)
			continue
		}
		sd, ok := d.(*ast.StructDecl)
		if !ok {
			continue
		}
		if len(sd.TypeParams) > 0 {
			a.registerGenericStruct(sd)
			continue
		}

		st := &StructType{
			Name:      a.emitFor(sd.Name, sd.Reach),
			Fields:    make(map[string]Type),
			Order:     make([]string, 0, len(sd.Members)),
			Methods:   make(map[string]*FuncType),
			FieldDocs: make(map[string]string),
			FieldDirs: make(map[string]string),
			Module:    a.module,
			Dir:       filepath.Dir(a.currentFile),
			Exported:  sd.Exported,
			Packed:    sd.TagNamed("packed") != nil,
			Opaque:    sd.Opaque,
		}
		st.Impl = a.newImplSet(sd.Implements, st, sd, a.moduleScope, nil)

		line, col := sd.Position()
		symbol := &Symbol{Name: sd.Name, Kind: STRUCT, Type: st, Line: line, Col: col, File: a.currentFile, Doc: sd.Doc, Private: sd.Reach == ast.ReachFile}

		if !a.defineTop(symbol) {
			a.errorAlreadyDeclared(sd, STRUCT, sd.Name)
		} else if st.Name != sd.Name {
			sd.Emit = st.Name
		}
	}
}

func (a *Analyser) registerStructFields() {
	a.structDecls = map[string]*ast.StructDecl{}
	a.fieldsStarted = map[*ast.StructDecl]bool{}
	for _, d := range a.program.Declarations {
		if sd, ok := d.(*ast.StructDecl); ok {
			a.structDecls[sd.Name] = sd
		}
	}

	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		if td, isEnum := d.(*ast.TypeDecl); isEnum && len(td.TypeParams) > 0 {
			continue
		}
		if td, isEnum := d.(*ast.TypeDecl); isEnum && td.Data {
			a.registerSumFields(td)
			continue
		}
		if sd, ok := d.(*ast.StructDecl); ok {
			a.registerFieldsOf(sd)
		}
	}
}

func (a *Analyser) registerFieldsOf(sd *ast.StructDecl) {
	if a.fieldsStarted[sd] || len(sd.TypeParams) > 0 {
		return
	}
	a.fieldsStarted[sd] = true

	savedFile := a.currentFile
	a.currentFile = a.declFiles[sd]
	defer func() { a.currentFile = savedFile }()

	symbol, _ := a.scope.Resolve(sd.Name)
	st := symbol.Type.(*StructType)

	for _, member := range sd.Members {
		fieldType := a.resolveTypeRef(sd, a.scope, member.Type)
		if fieldType == nil {
			continue
		}

		if _, ok := st.Fields[member.Name]; ok {
			a.errorf(sd, "duplicated field %s in struct %s", member.Name, sd.Name)
			continue
		}

		st.Fields[member.Name] = fieldType
		st.FieldDocs[member.Name] = member.Doc
		restrictField(st, member, a.currentFile, a.moduleFileSet())
		st.Order = append(st.Order, member.Name)
	}
	st.complete = true
}

func (a *Analyser) checkStructLiteral(expr *ast.StructLiteral) Type {
	if expr.Dot {
		a.pendingDotLits[expr] = true
		return ImplicitEnumType{Name: expr.Name}
	}
	if expr.Variant != "" && len(expr.TypeArgs) > 0 {
		symbol, ok := a.lookupType(a.scope, expr.Module, expr.Name)
		if !ok || symbol.Generic == nil || symbol.Generic.Sum == nil {
			a.errorf(expr, "%s is not a generic enum with data", qualified(expr.Module, expr.Name))
			return InvalidType{}
		}
		a.noteUse(expr, symbol)
		sum, isSum := a.instantiateStructRef(expr, symbol.Generic, expr.TypeArgs, a.scope).(*StructType)
		if !isSum {
			return InvalidType{}
		}
		return a.checkVariantLiteral(expr, sum, expr.Variant)
	}
	if expr.Variant != "" {
		sum, ok := a.sumTypeExpr(&ast.MemberExpr{Object: &ast.IdentLiteral{Value: expr.Module}, Field: expr.Name})
		if !ok {
			a.errorf(expr, "%s.%s is not an enum with data", expr.Module, expr.Name)
			return InvalidType{}
		}
		return a.checkVariantLiteral(expr, sum, expr.Variant)
	}
	if expr.Module != "" {
		if sum, ok := a.sumTypeExpr(&ast.IdentLiteral{Value: expr.Module}); ok {
			return a.checkVariantLiteral(expr, sum, expr.Name)
		}
	}
	symbol, ok := a.lookupType(a.scope, expr.Module, expr.Name)
	if ok && a.privateElsewhere(symbol) {
		a.reportPrivate(expr, symbol)
		return InvalidType{}
	}
	if ok {
		a.noteUse(expr, symbol)
	}
	if ok && symbol.Generic != nil && symbol.Generic.Struct != nil {
		if len(expr.TypeArgs) == 0 {
			a.errorf(expr, "%s is generic: write %s<...>{...}", qualified(expr.Module, expr.Name), qualified(expr.Module, expr.Name)).
				WithHelp("%s", symbol.Type.String())
			return InvalidType{}
		}
		inst := a.instantiateStructRef(expr, symbol.Generic, expr.TypeArgs, a.scope)
		st, isStruct := inst.(*StructType)
		if !isStruct {
			return InvalidType{}
		}
		return a.checkFieldInits(expr, st)
	}
	if ok && len(expr.TypeArgs) > 0 {
		a.errorf(expr, "%s takes no type arguments", qualified(expr.Module, expr.Name))
		return InvalidType{}
	}
	if !ok {
		d := a.errorf(expr, "unknown type %s", qualified(expr.Module, expr.Name))
		if expr.Module == "" {
			a.unavailableHint(d, expr.Name)
		}
		return InvalidType{}
	}

	st, ok := symbol.Type.(*StructType)
	if !ok {
		a.errorf(expr, "%s is not a struct", qualified(expr.Module, expr.Name))
		return InvalidType{}
	}
	if st.Sum != nil {
		a.errorf(expr, "%s has variants: write %s.Name{...} or .Name{...}", st.Name, st.Name)
		return InvalidType{}
	}
	if st.Opaque {
		a.errorf(expr, "%s is an extern type: it has no fields, C is the only one that creates it", st.Name)
		return InvalidType{}
	}

	return a.checkFieldInits(expr, st)
}

func (a *Analyser) checkFieldInits(expr *ast.StructLiteral, st *StructType) Type {
	seen := make(map[string]bool)
	for i, key := range expr.Keys {
		valueType := a.checkExpr(expr.Values[i])

		fieldType, ok := st.Fields[key]
		if !ok {
			a.suggest(a.errorf(expr, "unknown field %s in struct %s", key, st.Name), key, st.Order)
			continue
		}
		if !a.checkFieldReach(expr, st, key) {
			continue
		}
		if seen[key] {
			a.errorf(expr, "duplicated field %s in struct literal", key)
			continue
		}
		seen[key] = true

		if _, invalid := valueType.(InvalidType); invalid {
			continue
		}
		if !a.coerce(&expr.Values[i], valueType, fieldType) {
			a.helpMismatch(a.errorf(expr, "field %s expects %s, got %s", key, fieldType.String(), valueType.String()), valueType, fieldType)
			continue
		}
		a.checkConstFits(expr.Values[i], fieldType)
	}

	for _, field := range st.Order {
		if seen[field] {
			continue
		}
		if why := zeroProblem(st.Fields[field]); why != "" {
			a.errorf(expr, "field %s of %s is missing and has no value to start with: %s", field, st.Name, why).
				WithHelp("set it in the literal, or make the field optional (?T)")
		}
	}

	return st
}

func restrictField(st *StructType, member ast.Param, file string, moduleFiles map[string]bool) {
	switch member.Reach {
	case ast.ReachNone:
		if st.FieldModuleOnly == nil {
			st.FieldModuleOnly = map[string]bool{}
		}
		st.FieldModuleOnly[member.Name] = true
		st.ModuleFiles = moduleFiles
	case ast.ReachFile:
		if st.FieldFiles == nil {
			st.FieldFiles = map[string]string{}
		}
		st.FieldFiles[member.Name] = file
	case ast.ReachProject:
		if dir := reachDir(member.Reach, file); dir != "" {
			st.FieldDirs[member.Name] = dir
		}
	}
}

func (a *Analyser) moduleFileSet() map[string]bool {
	if a.moduleFiles == nil {
		a.moduleFiles = map[string]bool{}
		for _, file := range a.declFiles {
			if file != "" {
				a.moduleFiles[filepath.Clean(file)] = true
			}
		}
	}
	return a.moduleFiles
}

func FieldAccessible(st *StructType, field, file string) bool {
	if owner := st.FieldFiles[field]; owner != "" && file != "" && filepath.Clean(owner) != filepath.Clean(file) {
		return false
	}
	if st.FieldModuleOnly[field] && file != "" && len(st.ModuleFiles) > 0 && !st.ModuleFiles[filepath.Clean(file)] {
		return false
	}
	dir := st.FieldDirs[field]
	return dir == "" || insideDir(file, dir)
}

func (a *Analyser) checkFieldReach(node ast.Node, st *StructType, field string) bool {
	if FieldAccessible(st, field, a.currentFile) {
		return true
	}
	if st.FieldModuleOnly[field] && len(st.ModuleFiles) > 0 && !st.ModuleFiles[filepath.Clean(a.currentFile)] {
		a.errorf(node, "field %s of %s belongs to the module %q", field, st.Name, st.Module).WithHelp("only the files of that module can use it: write public in front of the field to share it with everyone, or internal to share it with the project")
		return false
	}
	if owner := st.FieldFiles[field]; owner != "" {
		name := filepath.Base(owner)
		a.errorf(node, "field %s of %s is private to %s", field, st.Name, name).WithHelp("only %s can use it: mark it internal to share it with the project, or public to share it with everyone", name)
		return false
	}
	project := filepath.Base(st.FieldDirs[field])
	a.errorf(node, "field %s of %s is internal to the project %q", field, st.Name, project).WithHelp("only the files of %s can use it", project)
	return false
}
