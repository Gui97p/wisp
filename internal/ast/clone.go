package ast

import "reflect"

type CloneOptions struct {
	Replace func(Node) (Node, bool)
	At      *Position
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
