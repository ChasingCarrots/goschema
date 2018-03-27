package generator

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"

	"github.com/chasingcarrots/goschema"
)

const mapReadCoreTemplate = `{{ .Token }}Entries := int({{ .Reader }}.ReadUInt32())
var {{ .MapKeyName }} {{ .MapKeyType }}
var {{ .MapValueName }} {{ .MapValueType }}
{{ .Token }}Map := make(map[{{ .MapKeyType }}]{{ .MapValueType}})
for {{ .Token }}I := 0; {{ .Token }}I < {{ .Token }}Entries; {{ .Token }}I++ {
	{{ .MapKeyReadingCode }}
	{{ .MapValueReadingCode }}
	{{ .Token }}Map[{{ .MapKeyName }}] = {{ .MapValueName }}
}
{{ .Dereference }}{{ .MapValue }} = {{ .Token }}Map
`

const mapReadTemplate = mapReadCoreTemplate

const mapSchemaReadTemplate = readSaveBase +
	mapReadCoreTemplate +
	readRestoreBase

const mapWriteCoreTemplate = `{{ .Writer }}.WriteUInt32(uint32(len({{ .Dereference }}{{ .MapValue }})))
for {{ .MapKeyName }}, {{ .MapValueName }} := range {{ .Dereference }}{{ .MapValue }} {
	{{ .MapKeyWritingCode }}
	{{ .MapValueWritingCode }}
}
`

const mapWriteTemplate = mapWriteCoreTemplate

const mapSchemaWriteTemplate = writeSaveBase +
	mapWriteCoreTemplate +
	writeRestoreBase

type MapSerializer struct {
	readTemplate               *template.Template
	readSchemaTemplate         *template.Template
	readSchemaCoreTemplate     *template.Template
	readSchemaRegisterTemplate *template.Template

	writeTemplate               *template.Template
	writeSchemaTemplate         *template.Template
	writeSchemaCoreTemplate     *template.Template
	writeSchemaRegisterTemplate *template.Template
}

func NewMapSerializer() *MapSerializer {
	return &MapSerializer{
		readTemplate:                template.Must(template.New("Read").Parse(mapReadTemplate)),
		readSchemaTemplate:          template.Must(template.New("Read").Parse(mapSchemaReadTemplate)),
		readSchemaCoreTemplate:      template.Must(template.New("Read").Parse(schemaReadCoreTemplate)),
		readSchemaRegisterTemplate:  template.Must(template.New("Read").Parse(schemaReadRegisterTemplate)),
		writeTemplate:               template.Must(template.New("Write").Parse(mapWriteTemplate)),
		writeSchemaTemplate:         template.Must(template.New("Write").Parse(mapSchemaWriteTemplate)),
		writeSchemaCoreTemplate:     template.Must(template.New("Write").Parse(schemaWriteCoreTemplate)),
		writeSchemaRegisterTemplate: template.Must(template.New("Write").Parse(schemaWriteRegisterTemplate)),
	}
}

func (*MapSerializer) Initialize(context *Context) {}

func (ms *MapSerializer) makeReadingProlog(context *Context, buf *bytes.Buffer, typ reflect.Type, readerName, valueName string) (string, bool) {
	target := TypeTarget(typ)
	serializer := context.FindSerializer(target)
	if serializer == nil {
		panic("Could not find serializer")
	}
	var readingCode string
	buf.WriteString("_ = ")
	buf.WriteString(readerName)
	buf.WriteString(".ReadUInt8() // ignore typecode\n")
	isSchema := serializer.TypeCode(target) == goschema.SchemaType
	if isSchema {
		schema := context.GetSchema(typ)
		token := context.UniqueToken()
		ms.readSchemaRegisterTemplate.Execute(buf, Lookup{
			"Token":      token,
			"Reader":     readerName,
			"SchemaName": schema.Name,
		})
		var readingCodeBuf bytes.Buffer
		ms.readSchemaCoreTemplate.Execute(&readingCodeBuf, Lookup{
			"Token":       token,
			"Reader":      readerName,
			"SchemaValue": valueName,
			"Reference":   "&",
		})
		readingCode = readingCodeBuf.String()
	} else {
		readingCode = serializer.MakeReadingCode(context, false, target, readerName, valueName)
	}
	return readingCode, isSchema
}

func (ms *MapSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	if ms.readTemplate == nil {
		ms.readTemplate = template.Must(template.New("Read").Parse(mapReadTemplate))
	}
	var buf bytes.Buffer
	token := context.UniqueToken()

	mapKeyName := token + "Key"
	keyReadingCode, keyIsSchema := ms.makeReadingProlog(
		context, &buf, target.Type.Key(),
		readerName, mapKeyName,
	)

	mapValueName := token + "Value"
	valueReadingCode, valueIsSchema := ms.makeReadingProlog(
		context, &buf, target.Type.Elem(),
		readerName, mapValueName,
	)
	tmpl := ms.readTemplate
	if keyIsSchema || valueIsSchema {
		tmpl = ms.readSchemaTemplate
	}

	tmpl.Execute(&buf,
		Lookup{
			"Token":               token,
			"MapValue":            valueName,
			"Reader":              readerName,
			"MapValueName":        mapValueName,
			"MapKeyName":          mapKeyName,
			"MapValueType":        context.GetTypeName(target.Type.Elem()),
			"MapKeyType":          context.GetTypeName(target.Type.Key()),
			"MapValueReadingCode": valueReadingCode,
			"MapKeyReadingCode":   keyReadingCode,
			"Dereference":         makeDeref(ptrValueTarget),
		},
	)
	return buf.String()
}

func (ms *MapSerializer) makeWritingProlog(context *Context, buf *bytes.Buffer, typ reflect.Type, writerName, valueName string) (string, bool) {
	token := context.UniqueToken()
	target := TypeTarget(typ)
	serializer := context.FindSerializer(target)
	if serializer == nil {
		panic("Could not find serializer")
	}

	var writingCode string
	typeCode := serializer.TypeCode(target)
	isSchema := typeCode == goschema.SchemaType
	buf.WriteString(writerName)
	buf.WriteString(".WriteUInt8(uint8(goschema.TypeCode(")
	buf.WriteString(fmt.Sprintf("%v", typeCode))
	buf.WriteString(")))\n")
	if isSchema {
		schema := context.GetSchema(typ)
		ms.writeSchemaRegisterTemplate.Execute(buf, Lookup{
			"Token":      token,
			"SchemaName": schema.Name,
			"SchemaID":   schema.ID,
			"Writer":     writerName,
		})
		var writingCodeBuf bytes.Buffer
		ms.writeSchemaCoreTemplate.Execute(&writingCodeBuf, Lookup{
			"Token":       token,
			"Writer":      writerName,
			"HeaderSize":  schema.HeaderSize,
			"SchemaValue": valueName,
			"Reference":   "&",
		})
		writingCode = writingCodeBuf.String()
	} else {
		writingCode = serializer.MakeWritingCode(context, false, target, writerName, valueName)
	}
	return writingCode, isSchema
}

func (ms *MapSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	if ms.writeTemplate == nil {
		ms.writeTemplate = template.Must(template.New("Write").Parse(mapWriteTemplate))
	}

	var buf bytes.Buffer
	token := context.UniqueToken()
	mapKeyName := token + "Key"
	mapValueName := token + "Value"
	keyWritingCode, keyIsSchema := ms.makeWritingProlog(
		context, &buf, target.Type.Key(),
		writerName, mapKeyName,
	)
	valueWritingCode, valueIsSchema := ms.makeWritingProlog(
		context, &buf, target.Type.Elem(),
		writerName, mapValueName,
	)

	tmpl := ms.writeTemplate
	if keyIsSchema || valueIsSchema {
		tmpl = ms.writeSchemaTemplate
	}

	tmpl.Execute(&buf,
		Lookup{
			"Token":               token,
			"MapValue":            valueName,
			"Writer":              writerName,
			"MapValueName":        mapValueName,
			"MapKeyName":          mapKeyName,
			"MapValueType":        context.GetTypeName(target.Type.Elem()),
			"MapKeyType":          context.GetTypeName(target.Type.Key()),
			"MapValueWritingCode": valueWritingCode,
			"MapKeyWritingCode":   keyWritingCode,
			"Dereference":         makeDeref(ptrValueTarget),
		},
	)
	return buf.String()
}

func (*MapSerializer) SizeOf() uint32 {
	return 4
}

func (*MapSerializer) CanSerialize(target Target) bool {
	return target.Type.Kind() == reflect.Map
}

func (*MapSerializer) IsVariableSize() bool {
	return true
}

func (*MapSerializer) WriteByValue() bool {
	return true
}

func (*MapSerializer) TypeCode(Target) goschema.TypeCode {
	return goschema.MapType
}
