package generator

import (
	"bytes"
	"reflect"
	"text/template"

	"github.com/chasingcarrots/goschema"
)

const listReadCoreTemplate_A = `{{ .Token }}Entries := int({{ .Reader }}.ReadUInt32())
{{ .Token }}Slice := make([]{{ .InnerType }}, {{ .Token }}Entries, {{ .Token }}Entries)
for {{ .Token }}I := 0; {{ .Token }}I < {{ .Token }}Entries; {{ .Token }}I++ {
`

const listReadCoreTemplate_B = `}
{{ .Dereference }}{{ .ListValue }} = {{ .Token }}Slice
`

const listWriteCoreTemplate_A = `{{ .Token }}Length := len({{ .Dereference }}{{ .ListValue }})
{{ .Writer }}.WriteUInt32(uint32({{ .Token }}Length))
for {{ .Token }}I := 0; {{ .Token }}I < {{ .Token }}Length; {{ .Token }}I++ {
`

const basicListReadTemplate = "_ = {{ .Reader }}.ReadUInt8() // ignore typecode\n" +
	listReadCoreTemplate_A +
	"{{ .InnerReadingCode }}\n" +
	listReadCoreTemplate_B

const schemaListReadTemplate = "_ = {{ .Reader }}.ReadUInt8() // ignore typecode\n" +
	schemaReadRegisterTemplate +
	readSaveBase +
	listReadCoreTemplate_A +
	schemaReadCoreTemplate +
	listReadCoreTemplate_B +
	readRestoreBase

const basicListWriteTemplate = "{{ .Writer }}.WriteUInt8(uint8(goschema.TypeCode({{ .TypeCode }})))\n" +
	listWriteCoreTemplate_A +
	"{{ .InnerWritingCode }}\n" +
	"}\n"

const schemaListWriteTemplate = "{{ .Writer }}.WriteUInt8(uint8(goschema.TypeCode({{ .TypeCode }})))\n" +
	writeSaveBase +
	schemaWriteRegisterTemplate +
	listWriteCoreTemplate_A +
	schemaWriteCoreTemplate +
	"}\n" +
	writeRestoreBase

type ListSerializer struct {
	readTemplate        *template.Template
	readSchemaTemplate  *template.Template
	writeTemplate       *template.Template
	writeSchemaTemplate *template.Template
}

func NewListSerializer() *ListSerializer {
	return &ListSerializer{
		readTemplate:        template.Must(template.New("Read").Parse(basicListReadTemplate)),
		readSchemaTemplate:  template.Must(template.New("ReadSchema").Parse(schemaListReadTemplate)),
		writeTemplate:       template.Must(template.New("Write").Parse(basicListWriteTemplate)),
		writeSchemaTemplate: template.Must(template.New("WriteSchema").Parse(schemaListWriteTemplate)),
	}
}

func (ls *ListSerializer) Initialize(context *Context) {}

func (ls *ListSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	innerType := TypeTarget(target.Type.Elem())
	serializer := context.FindSerializer(innerType)
	if serializer == nil {
		panic("Could not find serializer")
	}

	token := context.UniqueToken()
	var buf bytes.Buffer
	innerValueName := token + "Slice[" + token + "I]"
	if serializer.TypeCode(innerType) == goschema.SchemaType {
		schema := context.GetSchema(target.Type.Elem())
		ls.readSchemaTemplate.Execute(&buf,
			Lookup{
				"Token":       token,
				"ListValue":   valueName,
				"SchemaValue": innerValueName,
				"Reader":      readerName,
				"InnerType":   context.GetTypeName(innerType.Type),
				"Dereference": makeDeref(ptrValueTarget),
				"SchemaName":  schema.Name + "Schema",
				"Reference":   "&",
			},
		)
	} else {
		innerReadingCode := serializer.MakeReadingCode(context, false, innerType, readerName, innerValueName)
		ls.readTemplate.Execute(&buf,
			Lookup{
				"Token":            token,
				"ListValue":        valueName,
				"Reader":           readerName,
				"InnerType":        context.GetTypeName(innerType.Type),
				"InnerReadingCode": innerReadingCode,
				"Dereference":      makeDeref(ptrValueTarget),
			},
		)
	}
	return buf.String()
}

func (ls *ListSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	innerType := TypeTarget(target.Type.Elem())
	serializer := context.FindSerializer(innerType)
	if serializer == nil {
		panic("Could not find serializer")
	}
	token := context.UniqueToken()
	var buf bytes.Buffer
	innerValueName := valueName + "[" + token + "I]"
	if serializer.TypeCode(innerType) == goschema.SchemaType {
		schema := context.GetSchema(target.Type.Elem())
		ls.writeSchemaTemplate.Execute(&buf,
			Lookup{
				"Token":       token,
				"ListValue":   valueName,
				"SchemaValue": innerValueName,
				"Writer":      writerName,
				"Dereference": makeDeref(ptrValueTarget),
				"Reference":   "&",
				"TypeCode":    goschema.SchemaType,
				"SchemaID":    schema.ID,
				"SchemaName":  schema.Name + "Schema",
				"HeaderSize":  schema.HeaderSize,
			},
		)
	} else {
		innerWritingCode := serializer.MakeWritingCode(context, false, innerType, writerName, innerValueName)
		ls.writeTemplate.Execute(&buf,
			Lookup{
				"Token":            token,
				"ListValue":        valueName,
				"Writer":           writerName,
				"TypeCode":         serializer.TypeCode(innerType),
				"InnerWritingCode": innerWritingCode,
				"Dereference":      makeDeref(ptrValueTarget),
			},
		)
	}
	return buf.String()
}

func (*ListSerializer) SizeOf() uint32 {
	return 4
}

func (*ListSerializer) CanSerialize(target Target) bool {
	return target.Type.Kind() == reflect.Slice || target.Type.Kind() == reflect.Array
}

func (*ListSerializer) IsVariableSize() bool {
	return true
}

func (*ListSerializer) WriteByValue() bool {
	return true
}

func (*ListSerializer) TypeCode(Target) goschema.TypeCode {
	return goschema.ListType
}
