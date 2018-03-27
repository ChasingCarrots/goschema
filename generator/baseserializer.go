package generator

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"

	"github.com/chasingcarrots/goschema"
)

func makeDeref(ptrValueTarget bool) string {
	if ptrValueTarget {
		return "*"
	}
	return ""
}

func makeRef(ptrValueTarget bool) string {
	if ptrValueTarget {
		return ""
	}
	return "&"
}

const baseReadTemplate = "{{ .Dereference }}{{ .Value }} = {{ .Cast -}} ( {{- .Reader -}} .Read {{- .Method -}} ())"
const baseWriteTemplate = "{{ .Writer -}} .Write {{- .Method -}} ( {{- .Type -}} ( {{- .Dereference }}{{ .Value -}} ))"

type BaseSerializer struct {
	Type          reflect.Type
	methodName    string
	readTemplate  *template.Template
	writeTemplate *template.Template
}

func NewBaseSerializer(typ reflect.Type) *BaseSerializer {
	return &BaseSerializer{
		Type:          typ,
		methodName:    getMethodName(typ),
		readTemplate:  template.Must(template.New("Read").Parse(baseReadTemplate)),
		writeTemplate: template.Must(template.New("Write").Parse(baseWriteTemplate)),
	}
}

func getMethodName(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "Bool"
	case reflect.Int:
		return "Int"
	case reflect.Int8:
		return "Int8"
	case reflect.Int16:
		return "Int16"
	case reflect.Int32:
		return "Int32"
	case reflect.Int64:
		return "Int64"
	case reflect.Uint:
		return "UInt"
	case reflect.Uint8:
		return "UInt8"
	case reflect.Uint16:
		return "UInt16"
	case reflect.Uint32:
		return "UInt32"
	case reflect.Uint64:
		return "UInt64"
	case reflect.Float32:
		return "Float32"
	case reflect.Float64:
		return "Float64"
	default:
		panic(fmt.Sprintf("Invalid basic type %v", typ))
	}
}

func (is *BaseSerializer) Initialize(context *Context) {}

func (b *BaseSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	if b.readTemplate == nil {
		b.readTemplate = template.Must(template.New("Read").Parse(baseReadTemplate))
	}
	var buf bytes.Buffer
	b.readTemplate.Execute(&buf,
		Lookup{
			"Value":       valueName,
			"Reader":      readerName,
			"Cast":        context.GetTypeName(target.Type),
			"Type":        b.Type.Name(),
			"Dereference": makeDeref(ptrValueTarget),
			"Method":      b.methodName,
		},
	)
	return buf.String()
}

func (b *BaseSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	if b.writeTemplate == nil {
		b.writeTemplate = template.Must(template.New("Write").Parse(baseWriteTemplate))
	}
	var buf bytes.Buffer
	b.writeTemplate.Execute(&buf,
		Lookup{
			"Value":       valueName,
			"Writer":      writerName,
			"Type":        b.Type.Name(),
			"Dereference": makeDeref(ptrValueTarget),
			"Method":      b.methodName,
		},
	)
	return buf.String()
}

func (b *BaseSerializer) SizeOf() uint32 {
	return uint32(b.Type.Size())
}

func (b *BaseSerializer) CanSerialize(target Target) bool {
	return b.Type.Kind() == target.Type.Kind()
}

func (*BaseSerializer) IsVariableSize() bool {
	return false
}

func (*BaseSerializer) WriteByValue() bool {
	return true
}

func (*BaseSerializer) TypeCode(target Target) goschema.TypeCode {
	switch target.Type.Kind() {
	case reflect.Bool:
		return goschema.BoolType
	case reflect.Int:
		return goschema.IntType
	case reflect.Int8:
		return goschema.Int8Type
	case reflect.Int16:
		return goschema.Int16Type
	case reflect.Int32:
		return goschema.Int32Type
	case reflect.Int64:
		return goschema.Int64Type
	case reflect.Uint:
		return goschema.UIntType
	case reflect.Uint8:
		return goschema.UInt8Type
	case reflect.Uint16:
		return goschema.UInt16Type
	case reflect.Uint32:
		return goschema.UInt32Type
	case reflect.Uint64:
		return goschema.UInt64Type
	case reflect.Float32:
		return goschema.Float32Type
	case reflect.Float64:
		return goschema.Float64Type
	default:
		panic(fmt.Sprintf("Invalid basic type %v", target.Type))
	}
}
