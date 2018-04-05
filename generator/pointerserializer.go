package generator

import (
	"bytes"
	"reflect"
	"text/template"

	"github.com/chasingcarrots/goschema"
)

const pointerReadCoreTemplate_A = `{{ .Token }}NonNil := {{ .Reader }}.ReadBool()
if {{ .Token }}NonNil {
	var {{ .Token }} {{ .InnerType }}
`

const pointerReadCoreTemplate_B = `{{ .Dereference }}{{ .PointerValue }} = &{{ .Token }}
} else {
	{{ .Dereference }}{{ .PointerValue }} = nil
}
`

const pointerWriteCoreTemplate_A = `if {{ .PointerValue }} != nil {
	{{ .Writer }}.WriteBool(true)
`

const pointerWriteCoreTemplate_B = `} else {
	{{ .Writer }}.WriteBool(false)
}
`

const basicPointerReadTemplate = "_ = {{ .Reader }}.ReadUInt8() // ignore typecode\n" +
	pointerReadCoreTemplate_A +
	"{{ .InnerReadingCode }}\n" +
	pointerReadCoreTemplate_B

const schemaPointerReadTemplate = "_ = {{ .Reader }}.ReadUInt8() // ignore typecode\n" +
	schemaReadRegisterTemplate +
	readSaveBase +
	pointerReadCoreTemplate_A +
	schemaReadCoreTemplate +
	pointerReadCoreTemplate_B +
	readRestoreBase

const basicPointerWriteTemplate = "{{ .Writer }}.WriteUInt8(uint8(goschema.TypeCode({{ .TypeCode }})))\n" +
	pointerWriteCoreTemplate_A +
	"{{ .InnerWritingCode }}\n" +
	pointerWriteCoreTemplate_B

const schemaPointerWriteTemplate = "{{ .Writer }}.WriteUInt8(uint8(goschema.TypeCode({{ .TypeCode }})))\n" +
	writeSaveBase +
	schemaWriteRegisterTemplate +
	pointerWriteCoreTemplate_A +
	schemaWriteCoreTemplate +
	pointerWriteCoreTemplate_B +
	writeRestoreBase

type PointerSerializer struct {
	readTemplate        *template.Template
	readSchemaTemplate  *template.Template
	writeTemplate       *template.Template
	writeSchemaTemplate *template.Template
}

func NewPointerSerializer() *PointerSerializer {
	return &PointerSerializer{
		readTemplate:        template.Must(template.New("Read").Parse(basicPointerReadTemplate)),
		readSchemaTemplate:  template.Must(template.New("ReadSchema").Parse(schemaPointerReadTemplate)),
		writeTemplate:       template.Must(template.New("Write").Parse(basicPointerWriteTemplate)),
		writeSchemaTemplate: template.Must(template.New("WriteSchema").Parse(schemaPointerWriteTemplate)),
	}
}

func (*PointerSerializer) Initialize(context *Context) {}

func (ls *PointerSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	innerType := TypeTarget(target.Type.Elem())
	serializer := context.FindSerializer(innerType)
	if serializer == nil {
		panic("Could not find serializer")
	}

	token := context.UniqueToken()
	var buf bytes.Buffer
	innerName := token
	if serializer.TypeCode(context, innerType) == goschema.SchemaType {
		schema := context.GetSchema(target.Type.Elem())
		ls.readSchemaTemplate.Execute(&buf,
			Lookup{
				"Token":        token,
				"PointerValue": valueName,
				"SchemaValue":  innerName,
				"Reader":       readerName,
				"InnerType":    context.GetTypeName(innerType.Type),
				"Dereference":  makeDeref(ptrValueTarget),
				"SchemaName":   schema.Name,
				"Reference":    "&",
			},
		)
	} else {
		innerReadingCode := serializer.MakeReadingCode(context, false, innerType, readerName, innerName)
		ls.readTemplate.Execute(&buf,
			Lookup{
				"Token":            token,
				"PointerValue":     valueName,
				"Reader":           readerName,
				"InnerType":        context.GetTypeName(innerType.Type),
				"InnerReadingCode": innerReadingCode,
				"Dereference":      makeDeref(ptrValueTarget),
			},
		)
	}
	return buf.String()
}

func (ls *PointerSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	innerType := TypeTarget(target.Type.Elem())
	serializer := context.FindSerializer(innerType)
	if serializer == nil {
		panic("Could not find serializer")
	}
	token := context.UniqueToken()
	var buf bytes.Buffer
	innerValueName := valueName
	if ptrValueTarget {
		innerValueName = "*" + valueName
	}
	if serializer.TypeCode(context, innerType) == goschema.SchemaType {
		schema := context.GetSchema(target.Type.Elem())
		ls.writeSchemaTemplate.Execute(&buf,
			Lookup{
				"Token":        token,
				"PointerValue": valueName,
				"SchemaValue":  innerValueName,
				"Writer":       writerName,
				"Dereference":  makeDeref(ptrValueTarget),
				"Reference":    "",
				"TypeCode":     goschema.SchemaType,
				"SchemaID":     schema.ID,
				"SchemaName":   schema.Name,
				"HeaderSize":   schema.HeaderSize,
			},
		)
	} else {
		innerWritingCode := serializer.MakeWritingCode(context, true, innerType, writerName, innerValueName)
		ls.writeTemplate.Execute(&buf,
			Lookup{
				"Token":            token,
				"PointerValue":     valueName,
				"Writer":           writerName,
				"TypeCode":         serializer.TypeCode(context, innerType),
				"InnerWritingCode": innerWritingCode,
				"Dereference":      makeDeref(ptrValueTarget),
			},
		)
	}
	return buf.String()
}

func (*PointerSerializer) SizeOf(*Context, Target) uint32 {
	return 4
}

func (*PointerSerializer) CanSerialize(context *Context, target Target) bool {
	if target.Type.Kind() != reflect.Ptr {
		return false
	}
	target.Type = target.Type.Elem()
	return context.FindSerializer(target) != nil
}

func (*PointerSerializer) IsVariableSize(*Context, Target) bool {
	return true
}

func (*PointerSerializer) WriteByValue(*Context, Target) bool {
	return true
}

func (*PointerSerializer) TypeCode(*Context, Target) goschema.TypeCode {
	return goschema.PointerType
}
