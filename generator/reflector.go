package generator

import (
	"reflect"
)

// TypeName returns the name of the given type as it would appear in a source file in
// the specified package. This assumes that no imports have been renamed.
func TypeName(typ reflect.Type, pkgPath string) string {
	name := ""
	for {
		switch typ.Kind() {
		case reflect.Ptr:
			name += "*"
			typ = typ.Elem()
		case reflect.Slice:
			name += "[]"
			typ = typ.Elem()
		case reflect.Array:
			panic("Arrays not implemented")
		case reflect.Map:
			name += "map["
			name += TypeName(typ.Key(), pkgPath)
			name += "]"
			name += TypeName(typ.Elem(), pkgPath)
			return name
		case reflect.Chan:
			panic("Channels not implemented")
		case reflect.Func:
			panic("Functions not implemented")
		default:
			path := typ.PkgPath()
			if path == pkgPath {
				return name + typ.Name()
			}
			return name + typ.String()
		}
	}
}

// importPaths collect all import paths that are required to use a type.
func ImportPaths(typ reflect.Type) []string {
	stack := []reflect.Type{typ}
	paths := make(map[string]struct{})
	for len(stack) > 0 {
		next := stack[len(stack)-1]
		stack = stack[0 : len(stack)-1]
		switch next.Kind() {
		case reflect.Struct, reflect.Interface:
			paths[next.PkgPath()] = struct{}{}
		case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Chan:
			stack = append(stack, next.Elem())
		case reflect.Map:
			stack = append(stack, next.Elem(), next.Key())
		case reflect.Func:
			for i := 0; i < next.NumIn(); i++ {
				stack = append(stack, next.In(i))
			}
			for i := 0; i < next.NumOut(); i++ {
				stack = append(stack, next.Out(i))
			}
		default:
			path := next.PkgPath()
			if len(path) > 0 {
				paths[path] = struct{}{}
			}
		}
	}
	output := make([]string, 0, len(paths))
	for k := range paths {
		output = append(output, k)
	}
	return output
}
