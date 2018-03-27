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

const readSaveBase = "{{ .Token }}ViewBase := {{ .Reader }}.Base()\n"
const readRestoreBase = "{{ .Reader }}.View({{ .Reader }}.Local({{ .Token }}ViewBase))\n"

const schemaReadCoreTemplate = `{{ .Token }}Length := int64({{ .Reader }}.ReadUInt32())
{{ .Token }}NextOffset := {{ .Reader }}.GlobalOffset() + {{ .Token }}Length
{{ .Reader }}.ViewHere()
{{ .Token }}SchemaProper.ReadFrom({{ .Reader }}, {{ .Reference }}{{ .SchemaValue }}, context)
{{ .Reader }}.Seek({{ .Reader }}.Local({{ .Token }}NextOffset), io.SeekStart)
`

const schemaReadRegisterTemplate = `{{ .Token }}SchemaIdx := {{ .Reader }}.ReadUInt32()
{{ .Token }}Schema, {{ .Token }}SchemaEntries := {{ .Reader }}.FindSchema(int({{ .Token }}SchemaIdx))
{{ .Token }}SchemaProper, ok := {{ .Token }}Schema.(*{{ .SchemaName }})
if !ok || {{ .Token }}Schema == nil {
	{{ .Token }}SchemaProper = New{{ .SchemaName }}()
	{{ .Token }}SchemaProper.Fill({{ .Token }}SchemaEntries)
	{{ .Reader }}.RegisterSchema(int({{ .Token }}SchemaIdx), {{ .Token }}SchemaProper)
}
`

const schemaWriteTemplate = schemaWriteRegisterTemplate +
	writeSaveBase +
	schemaWriteCoreTemplate +
	writeRestoreBase

const writeSaveBase = "{{ .Token }}ViewBase := {{ .Writer }}.Base()\n"
const writeRestoreBase = "{{ .Writer }}.View({{ .Writer }}.Local({{ .Token }}ViewBase))\n"

const schemaWriteCoreTemplate = `{{ .Writer }}.WriteUInt32(0) // reserved for size
{{ .Writer }}.ViewHere()
{{ .Token }}StartOffset := {{ .Writer }}.GlobalOffset()
{{ .Writer }}.Seek({{ .HeaderSize }}, io.SeekCurrent)
{{ .Token }}Schema.WriteTo({{ .Writer }}, {{ .Reference }}{{ .SchemaValue }}, context)
{{ .Token }}EndOffset := {{ .Writer }}.GlobalOffset()
{{ .Writer }}.Seek({{ .Writer }}.Local({{ .Token }}StartOffset - 4), io.SeekStart)
{{ .Writer }}.WriteUInt32(uint32({{ .Token }}EndOffset - {{ .Token }}StartOffset))
{{ .Writer }}.Seek({{ .Writer }}.Local({{ .Token }}EndOffset), io.SeekStart)
`

const schemaWriteRegisterTemplate = `{{ .Token }}SchemaEntry, ok := {{ .Writer }}.FindSchema(goschema.SchemaID({{ .SchemaID }}))
{{ .Token }}SchemaIdx := {{ .Token }}SchemaEntry.Index()
{{ .Token }}Schema, ok := {{ .Token }}SchemaEntry.Schema().(*{{ .SchemaName }})
if !ok {
	{{ .Token }}Schema = New{{ .SchemaName }}()
	{{ .Token }}SchemaIdx = {{ .Writer }}.RegisterSchema({{ .Token }}Schema)
}
{{ .Writer }}.WriteUInt32(uint32({{ .Token }}SchemaIdx))
`

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
			"SchemaName":  schema.Name + "Schema",
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
			"SchemaName":  schema.Name + "Schema",
			"HeaderSize":  schema.HeaderSize,
		},
	)
	return buf.String()
}

func (*SchemaSerializer) SizeOf() uint32 {
	return 4
}

func (*SchemaSerializer) CanSerialize(target Target) bool {
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
