package generator

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/chasingcarrots/goschema"
)

type InlineSerializer struct {
	Type   reflect.Type
	size   uint32
	fields []inlineSerializerEntry
	active bool
	code   goschema.TypeCode
}

type inlineSerializerEntry struct {
	serializer TypeSerializer
	field      reflect.StructField
}

func NewInlineSerializer(typ reflect.Type, code goschema.TypeCode) *InlineSerializer {
	return &InlineSerializer{
		Type: typ,
		code: code,
	}
}

func (is *InlineSerializer) Initialize(context *Context) {
	n := is.Type.NumField()
	size := uint32(0)
	for i := 0; i < n; i++ {
		field := is.Type.Field(i)
		if field.Anonymous {
			fmt.Printf("Skipping anonymous field %v on %v\n", field.Type, is.Type.Name())
			continue
		}
		serializer := context.FindSerializer(Target{
			Type: field.Type,
			Tags: field.Tag,
		})
		if serializer == nil {
			fmt.Printf("Ignoring field %v on %v because there is no serializer for its type %v\n", field.Name, is.Type.String(), field.Type)
			continue
		}
		size += serializer.SizeOf()
		is.fields = append(is.fields, inlineSerializerEntry{
			serializer: serializer,
			field:      field,
		})
	}
	is.size = size
}

func (is *InlineSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	if is.active {
		panic(fmt.Sprintf("inline serializer cannot be used for self-referential data such as %v", target.Type))
	}
	is.active = true
	var buf bytes.Buffer
	for _, entry := range is.fields {
		fieldTarget := valueName + "." + entry.field.Name
		readingCode := entry.serializer.MakeReadingCode(context, false, Target{Type: entry.field.Type, Tags: entry.field.Tag}, readerName, fieldTarget)
		buf.WriteString(readingCode)
		buf.WriteRune('\n')
	}
	is.active = false
	return buf.String()
}

func (is *InlineSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	if is.active {
		panic(fmt.Sprintf("inline serializer cannot be used for self-referential data such as %v", target.Type))
	}
	is.active = true
	var buf bytes.Buffer
	for _, entry := range is.fields {
		fieldTarget := valueName + "." + entry.field.Name
		writingCode := entry.serializer.MakeWritingCode(context, false, Target{Type: entry.field.Type, Tags: entry.field.Tag}, writerName, fieldTarget)
		buf.WriteString(writingCode)
		buf.WriteRune('\n')
	}
	is.active = false
	return buf.String()
}

func (is *InlineSerializer) SizeOf() uint32 {
	return is.size
}

func (is *InlineSerializer) CanSerialize(context *Context, target Target) bool {
	return is.Type == target.Type
}

func (*InlineSerializer) IsVariableSize() bool {
	return false
}

func (is *InlineSerializer) WriteByValue() bool {
	return is.size <= 8
}

func (is *InlineSerializer) TypeCode(Target) goschema.TypeCode {
	return is.code
}
