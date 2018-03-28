package goschema

import (
	"github.com/chasingcarrots/gobinary"
)

type SchemaWriter struct {
	gobinary.HighLevelWriter
	gobinary.StreamWriterView
	schemaData *SchemaDBWriter
}

func MakeSchemaWriter(schemaData *SchemaDBWriter, stream *gobinary.StreamWriter) SchemaWriter {
	sw := SchemaWriter{
		schemaData:       schemaData,
		StreamWriterView: gobinary.MakeStreamWriterView(stream),
	}
	sw.HighLevelWriter = gobinary.MakeHighLevelWriter(stream)
	return sw
}

func (sw *SchemaWriter) FindSchema(id SchemaID) (SchemaDataEntry, bool) {
	return sw.schemaData.FindSchema(id)
}

func (sw *SchemaWriter) RegisterSchema(schema Schema) int {
	return sw.schemaData.RegisterSchema(schema)
}

func (sw *SchemaWriter) WriteInt(value int) {
	sw.WriteInt64(int64(value))
}

func (sw *SchemaWriter) WriteUInt(value uint) {
	sw.WriteUInt64(uint64(value))
}
