package generator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/chasingcarrots/gotransform"

	"github.com/chasingcarrots/goschema"
)

type SchemaMetaData struct {
	Type                 reflect.Type
	Name                 string
	ID                   int
	HeaderSize           uint32
	Imports              map[string]struct{}
	ready, inPreparation bool
}

type Context struct {
	outputPath   string
	packagePath  string // path of the package for the output files
	serializers  []TypeSerializer
	tokenCounter int

	schemaMetaData map[reflect.Type]*SchemaMetaData
	schemaTemplate *template.Template
	schemaStack    []*SchemaMetaData

	writeMethod, readMethod   *template.Template
	writeContext, readContext reflect.Type
}

func NewContext(outputPath, packagePath, schemaTemplatePath string, writeContext, readContext reflect.Type) *Context {
	return &Context{
		schemaTemplate: template.Must(template.ParseFiles(schemaTemplatePath)),
		writeMethod:    template.Must(template.New("WriteMethod").Parse(writingMethodSchema)),
		readMethod:     template.Must(template.New("ReadMethod").Parse(readingMethodSchema)),
		outputPath:     outputPath,
		packagePath:    packagePath,
		schemaMetaData: make(map[reflect.Type]*SchemaMetaData),
		writeContext:   writeContext,
		readContext:    readContext,
	}
}

func (c *Context) RequestSchema(typ reflect.Type, name string) {
	c.schemaMetaData[typ] = &SchemaMetaData{
		Type:    typ,
		Name:    name,
		ID:      len(c.schemaMetaData),
		Imports: make(map[string]struct{}),
	}
}

func (c *Context) AddDefaultSerializers() {
	c.AddSerializers(NewSchemaSerializer(),
		NewBaseSerializer(reflect.TypeOf(int(0))),
		NewBaseSerializer(reflect.TypeOf(int8(0))),
		NewBaseSerializer(reflect.TypeOf(int16(0))),
		NewBaseSerializer(reflect.TypeOf(int32(0))),
		NewBaseSerializer(reflect.TypeOf(int64(0))),
		NewBaseSerializer(reflect.TypeOf(uint(0))),
		NewBaseSerializer(reflect.TypeOf(uint8(0))),
		NewBaseSerializer(reflect.TypeOf(uint16(0))),
		NewBaseSerializer(reflect.TypeOf(uint32(0))),
		NewBaseSerializer(reflect.TypeOf(uint64(0))),
		NewBaseSerializer(reflect.TypeOf(float32(0))),
		NewBaseSerializer(reflect.TypeOf(float64(0))),
		NewBaseSerializer(reflect.TypeOf(false)),
		NewStringSerializer(),
		NewListSerializer(),
		NewMapSerializer(),
	)
}

func (c *Context) AddSerializers(ts ...TypeSerializer) {
	c.serializers = append(c.serializers, ts...)
}

func (c *Context) Generate() error {
	for _, s := range c.serializers {
		s.Initialize(c)
	}
	for _, v := range c.schemaMetaData {
		if !v.ready {
			if err := c.generateSchema(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Context) UniqueToken() string {
	token := c.tokenCounter
	c.tokenCounter++
	return fmt.Sprintf("v%v", token)
}

func (c *Context) FindSerializer(target Target) TypeSerializer {
	n := len(c.serializers)
	for i := n - 1; i >= 0; i-- {
		if c.serializers[i].CanSerialize(c, target) {
			return c.serializers[i]
		}
	}
	return nil
}

// GetTypeName returns the name of the given type in the current context.
// Its name may depend on the package name, hence this abstraction.
func (c *Context) GetTypeName(typ reflect.Type) string {
	paths := ImportPaths(typ)
	top := c.schemaStack[len(c.schemaStack)-1]
	imports := top.Imports
	for _, p := range paths {
		if p != c.packagePath {
			imports[p] = struct{}{}
		}
	}
	return TypeName(typ, c.packagePath)
}

func (c *Context) packageName() string {
	idx := strings.LastIndex(c.packagePath, "/")
	if idx == -1 {
		return c.packagePath
	}
	return c.packagePath[idx+1 : len(c.packagePath)]
}

// GetSchema returns the schema meta data associated to the given type.
func (c *Context) GetSchema(typ reflect.Type) *SchemaMetaData {
	data, ok := c.schemaMetaData[typ]
	if !ok {
		c.RequestSchema(typ, typ.Name()+"AutoGen")
		data = c.schemaMetaData[typ]
	}
	if !data.inPreparation && !data.ready {
		c.generateSchema(data)
	}
	return data
}

type schemaField struct {
	Name      string            // name used for serialization
	FieldName string            // name of the field in the struct
	Offset    uint32            // offset of the field in the schema
	TypeCode  goschema.TypeCode // typecode in the schema
	Reference string            // "&" when writing should proceed by pointer
}

const writingMethodSchema = `func (schema *{{ .SchemaName }}Schema) Write{{ .Name }}(writer *goschema.SchemaWriter, value {{ .WritingType }}, context {{ .WritingContextType }}) {
	offset := writer.Offset()
	writer.Seek(int64(schema.{{ .Name }}Offset), io.SeekStart)
{{ if .InPlace -}}
	{{ .WriteCode }}
	writer.Seek(offset, io.SeekStart)
{{ else -}}
	writer.WriteUInt32(uint32(offset))
	writer.Seek(offset, io.SeekStart)
	{{ .WriteCode }}
{{- end -}}
}

`

const readingMethodSchema = `func (schema *{{ .SchemaName }}Schema) Read{{ .Name }}Into(reader *goschema.SchemaReader, value *{{ .ReadingType }}, context {{ .ReadingContextType }}) {
	if schema.{{ .Name }}Offset == -1 {
{{- if .Default }}
		*value = {{ .ReadingType }}({{ .Default }})
{{- else }}
		var tmp {{ .ReadingType }}
		*value = tmp
{{- end }}
		return
	}
	offset := reader.Offset()
	reader.Seek(int64(schema.{{ .Name }}Offset), io.SeekStart)
{{- if .InPlace -}}
{{ else }}
	fieldOffset := reader.ReadUInt32()
	reader.Seek(int64(fieldOffset), io.SeekStart)
{{- end }}
	{{ .ReadCode }}
	reader.Seek(offset, io.SeekStart)
}

`

func (c *Context) generateSchema(data *SchemaMetaData) error {
	data.inPreparation = true
	c.schemaStack = append(c.schemaStack, data)
	size := uint32(0)
	n := data.Type.NumField()
	schemaFields := make([]schemaField, 0, n)

	writingContextType := c.GetTypeName(c.writeContext)
	readingContextType := c.GetTypeName(c.readContext)

	var methodBuf bytes.Buffer
	for i := 0; i < n; i++ {
		field := data.Type.Field(i)
		_, ignore := field.Tag.Lookup("schemaIgnore")
		if ignore {
			continue
		}
		target := Target{Type: field.Type, Tags: field.Tag}
		serializer := c.FindSerializer(target)
		if serializer == nil {
			fmt.Printf("Ignoring field %v of %v because there is no serializer for its type %v\n", field.Name, data.Type.String(), field.Type.String())
			continue
		}

		writeByValue := serializer.WriteByValue()
		writeCode := serializer.MakeWritingCode(c, !writeByValue, target, "writer", "value")
		readCode := serializer.MakeReadingCode(c, true, target, "reader", "value")

		serializedName := tag(field.Tag, "schemaName", field.Name)
		writingType := c.GetTypeName(field.Type)
		reference := ""
		if !writeByValue {
			writingType = "*" + writingType
			reference = "&"
		}

		defaultValue := tag(field.Tag, "schemaDefault", "")
		readingType := c.GetTypeName(field.Type)
		isInPlace := "yes"
		if serializer.IsVariableSize() {
			isInPlace = ""
		}

		c.writeMethod.Execute(&methodBuf,
			Lookup{
				"SchemaName":         data.Name,
				"Name":               serializedName,
				"WritingType":        writingType,
				"WritingContextType": writingContextType,
				"WriteCode":          writeCode,
				"InPlace":            isInPlace,
			},
		)
		c.readMethod.Execute(&methodBuf,
			Lookup{
				"SchemaName":         data.Name,
				"Name":               serializedName,
				"ReadingType":        readingType,
				"ReadCode":           readCode,
				"Default":            defaultValue,
				"ReadingContextType": readingContextType,
				"InPlace":            isInPlace,
			},
		)

		schemaFields = append(schemaFields,
			schemaField{
				Name:      serializedName,
				FieldName: field.Name,
				Offset:    size,
				TypeCode:  serializer.TypeCode(target), // TODO compute type code
				Reference: reference,
			},
		)

		size += serializer.SizeOf()
	}
	data.HeaderSize = size

	targetTypeName := c.GetTypeName(data.Type)
	var imports []string
	for k := range data.Imports {
		imports = append(imports, `"`+k+`"`)
	}

	var buf bytes.Buffer
	c.schemaTemplate.Execute(&buf,
		Lookup{
			"SchemaName":         data.Name,
			"SchemaSize":         data.HeaderSize,
			"Fields":             schemaFields,
			"NumFields":          len(schemaFields),
			"WritingContextType": writingContextType,
			"ReadingContextType": readingContextType,
			"TargetType":         targetTypeName,
			"Imports":            imports,
			"Package":            c.packageName(),
			"ID":                 data.ID,
		},
	)

	buf.WriteRune('\n')
	buf.WriteRune('\n')
	methodBuf.WriteTo(&buf)

	data.ready = true

	c.schemaStack = c.schemaStack[0 : len(c.schemaStack)-1]
	data.inPreparation = false

	return gotransform.WriteGoFile(filepath.Join(c.outputPath, data.Name+"_schema_gen.go"), &buf)
}

func tag(tags reflect.StructTag, key, defaultValue string) string {
	value, ok := tags.Lookup(key)
	if !ok {
		return defaultValue
	}
	return value
}
