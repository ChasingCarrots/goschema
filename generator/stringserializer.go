package generator

import (
	"bytes"
	"reflect"
	"text/template"

	"github.com/chasingcarrots/goschema"
)

const stringReadCoreTemplate = `{{ .Token }}Length := {{ .Reader }}.ReadUInt32()
{{ .Dereference }}{{ .Value }} = {{ .Cast -}} ( {{- .Reader -}} .ReadString(int({{ .Token }}Length)))
`

const stringReadTemplate = stringReadCoreTemplate

const stringWriteCoreTemplate = `{{ .Writer }}.WriteUInt32(uint32(len({{ .Dereference }}{{ .Value }})))
{{ .Writer }}.WriteString({{ .Dereference }}{{ .Value }})
`

const stringWriteTemplate = stringWriteCoreTemplate

type StringSerializer struct {
	readTemplate  *template.Template
	writeTemplate *template.Template
}

func NewStringSerializer() *StringSerializer {
	return &StringSerializer{
		readTemplate:  template.Must(template.New("Read").Parse(stringReadTemplate)),
		writeTemplate: template.Must(template.New("Write").Parse(stringWriteTemplate)),
	}
}

func (*StringSerializer) Initialize(context *Context) {}

func (ss *StringSerializer) MakeReadingCode(context *Context, ptrValueTarget bool, target Target, readerName, valueName string) string {
	var buf bytes.Buffer
	ss.readTemplate.Execute(&buf,
		Lookup{
			"Token":       context.UniqueToken(),
			"Value":       valueName,
			"Reader":      readerName,
			"Cast":        context.GetTypeName(target.Type),
			"Dereference": makeDeref(ptrValueTarget),
		},
	)
	return buf.String()
}

func (ss *StringSerializer) MakeWritingCode(context *Context, ptrValueTarget bool, target Target, writerName, valueName string) string {
	var buf bytes.Buffer
	ss.writeTemplate.Execute(&buf,
		Lookup{
			"Token":       context.UniqueToken(),
			"Value":       valueName,
			"Writer":      writerName,
			"Cast":        context.GetTypeName(target.Type),
			"Dereference": makeDeref(ptrValueTarget),
		},
	)
	return buf.String()
}

func (*StringSerializer) SizeOf() uint32 {
	return 4
}

func (*StringSerializer) CanSerialize(context *Context, target Target) bool {
	return target.Type.Kind() == reflect.String
}

func (*StringSerializer) IsVariableSize() bool {
	return true
}

func (*StringSerializer) WriteByValue() bool {
	return true
}

func (*StringSerializer) TypeCode(Target) goschema.TypeCode {
	return goschema.StringType
}
