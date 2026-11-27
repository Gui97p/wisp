package ast

import "reflect"

type CloneOptions struct {
	Replace func(Node) (Node, bool)
	At      *Position
	Remap   func(line, col int) (int, int)
}

func remapPositions(v reflect.Value, remap func(line, col int) (int, int)) {
	direct := func(name string) reflect.Value {
		if field, ok := v.Type().FieldByName(name); ok && len(field.Index) == 1 {
			return v.FieldByIndex(field.Index)
		}
		return reflect.Value{}
	}
	pair := func(lineName, colName string) {
		line, col := direct(lineName), direct(colName)
		if !line.IsValid() || !col.IsValid() || line.Kind() != reflect.Int || col.Kind() != reflect.Int || line.Int() == 0 {
			return
		}
		newLine, newCol := remap(int(line.Int()), int(col.Int()))
		line.SetInt(int64(newLine))
		col.SetInt(int64(newCol))
	}
	pair("Line", "Col")
	pair("EndLine", "EndCol")
}

func Clone[T Node](node T, options CloneOptions) T {
	copied := cloneValue(reflect.ValueOf(node), options)
	return copied.Interface().(T)
}

var nodePosType = reflect.TypeOf(NodePos{})
var nodeType = reflect.TypeOf((*Node)(nil)).Elem()

func cloneValue(v reflect.Value, options CloneOptions) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		copied := reflect.New(v.Type().Elem())
		copied.Elem().Set(cloneValue(v.Elem(), options))
		return copied
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		if options.Replace != nil && v.Elem().Kind() == reflect.Pointer && v.Elem().Type().Implements(nodeType) && !v.Elem().IsNil() {
			if replacement, ok := options.Replace(v.Elem().Interface().(Node)); ok {
				replaced := reflect.ValueOf(replacement)
				if replaced.Type().AssignableTo(v.Type()) {
					result := reflect.New(v.Type()).Elem()
					result.Set(replaced)
					return result
				}
			}
		}
		copied := cloneValue(v.Elem(), options)
		result := reflect.New(v.Type()).Elem()
		result.Set(copied)
		return result
	case reflect.Struct:
		copied := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			copied.Field(i).Set(cloneValue(v.Field(i), options))
		}
		if options.Remap != nil {
			remapPositions(copied, options.Remap)
		}
		if v.Type() == nodePosType && options.At != nil {
			copied.FieldByName("Line").SetInt(int64(options.At.Line))
			copied.FieldByName("Col").SetInt(int64(options.At.Col))
			copied.FieldByName("EndLine").SetInt(int64(options.At.EndLine))
			copied.FieldByName("EndCol").SetInt(int64(options.At.EndCol))
		}
		return copied
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		copied := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			copied.Index(i).Set(cloneValue(v.Index(i), options))
		}
		return copied
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		copied := reflect.MakeMapWithSize(v.Type(), v.Len())
		for _, key := range v.MapKeys() {
			copied.SetMapIndex(key, cloneValue(v.MapIndex(key), options))
		}
		return copied
	}
	return v
}

func Inspect(node Node, visit func(Node) bool) {
	inspectValue(reflect.ValueOf(node), visit)
}

func inspectValue(v reflect.Value, visit func(Node) bool) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		if v.Type().Implements(nodeType) {
			if !visit(v.Interface().(Node)) {
				return
			}
		}
		inspectValue(v.Elem(), visit)
	case reflect.Interface:
		if !v.IsNil() {
			inspectValue(v.Elem(), visit)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				inspectValue(v.Field(i), visit)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			inspectValue(v.Index(i), visit)
		}
	}
}

var typeRefType = reflect.TypeOf(TypeRef{})

// Mentions adds to into every name that node uses as a value and the module
// part of every qualified type it names, which is where a module alias shows up.
func Mentions(node Node, into map[string]bool) {
	mentionsOf(reflect.ValueOf(node), into)
}

func mentionsOf(v reflect.Value, into map[string]bool) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		if ident, ok := v.Interface().(*IdentLiteral); ok {
			into[ident.Value] = true
		}
		mentionsOf(v.Elem(), into)
	case reflect.Interface:
		if !v.IsNil() {
			mentionsOf(v.Elem(), into)
		}
	case reflect.Struct:
		if v.Type() == typeRefType {
			if module := v.FieldByName("Module").String(); module != "" {
				into[module] = true
			}
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				mentionsOf(v.Field(i), into)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			mentionsOf(v.Index(i), into)
		}
	}
}
