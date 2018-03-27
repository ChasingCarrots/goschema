package generator

import (
	"bytes"
	"reflect"
	"text/template"

	"github.com/chasingcarrots/goschema"
)

const schemaReadTemplate = schemaReadRegisterTemplate +
	readSaveBase +
	schemaReadCoreTemplate +
	readRestoreBase

const schemaReadRegisterTemplate = "{{ .Token }}Schema := Read{{ .SchemaName }}Schema({{ .Reader }})\n"
const readSaveBase = "{{ .Token }}ViewBase := {{ .Reader }}.Base()\n"
const schemaReadCoreTemplate = "{{ .Token }}Schema.NakedRead({{ .Reader }}, {{ .Reference }}{{ .SchemaValue }}, context)\n"
const readRestoreBase = "{{ .Reader }}.View({{ .Reader }}.Local({{ .Token }}ViewBase))\n"

const schemaWriteTemplate = schemaWriteRegisterTemplate +
	writeSaveBase +
	schemaWriteCoreTemplate +
	writeRestoreBase

const schemaWriteRegisterTemplate = "{{ .Token }}Schema := Write{{ .SchemaName }}Schema({{ .Writer }})\n"
const writeSaveBase = "{{ .Token }}ViewBase := {{ .Writer }}.Base()\n"
const schemaWriteCoreTemplate = "{{ .Token }}Schema.NakedWrite({{ .Writer }}, {{ .Reference }}{{ .SchemaValue }}, context)\n"
const writeRestoreBase = "{{ .Writer }}.View({{ .Writer }}.Local({{ .Token }}ViewBase))\n"

type SchemaSerializer struct {
	readTemplate  *template.Template
	writeTemplate *template.Template
}

func NewSchemaSerializer() *SchemaSerializer {
	return &SchemaSerializer{
		readTemplate:  template.Must(template.New("Read").Parse(schemaReadTemplate)),
		writeTemplate: template.Must(template.New("Write").Parse(schemaWriteTemplate)),
	}
}

func (*SchemaSerializer) Initialize(context *Context) {}

func (ss *SchemaSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	schema := context.GetSchema(target.Type)
	var buf bytes.Buffer
	ss.readTemplate.Execute(&buf,
		Lookup{
			"Token":       context.UniqueToken(),
			"SchemaValue": valueName,
			"Reader":      readerName,
			"SchemaName":  schema.Name,
			"Reference":   makeRef(ptrValueTarget),
		},
	)
	return buf.String()
}

func (ss *SchemaSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	schema := context.GetSchema(target.Type)
	var buf bytes.Buffer
	ss.writeTemplate.Execute(&buf,
		Lookup{
			"Token":       context.UniqueToken(),
			"SchemaValue": valueName,
			"Writer":      writerName,
			"Reference":   makeRef(ptrValueTarget),
			"SchemaID":    schema.ID,
			"SchemaName":  schema.Name,
			"HeaderSize":  schema.HeaderSize,
		},
	)
	return buf.String()
}

func (*SchemaSerializer) SizeOf() uint32 {
	return 4
}

func (*SchemaSerializer) CanSerialize(context *Context, target Target) bool {
	return target.Type.Kind() == reflect.Struct
}

func (*SchemaSerializer) IsVariableSize() bool {
	return true
}

func (*SchemaSerializer) WriteByValue() bool {
	return false
}

func (*SchemaSerializer) TypeCode(Target) goschema.TypeCode {
	return goschema.SchemaType
}
