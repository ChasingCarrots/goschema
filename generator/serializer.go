package generator

import (
	"reflect"

	"github.com/chasingcarrots/goschema"
)

type Lookup = map[string]interface{}

type TypeSerializer interface {
	IsVariableSize(*Context, Target) bool
	WriteByValue(*Context, Target) bool
	MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string
	MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string
	SizeOf(*Context, Target) uint32
	CanSerialize(*Context, Target) bool
	TypeCode(*Context, Target) goschema.TypeCode
	Initialize(*Context)
}

type Target struct {
	Type reflect.Type
	Tags reflect.StructTag
}

func TypeTarget(typ reflect.Type) Target {
	return Target{Type: typ}
}
