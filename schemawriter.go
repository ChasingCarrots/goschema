package goschema

import (
	"io"

	"github.com/chasingcarrots/gobinary"
)

type SchemaWriter struct {
	gobinary.StreamWriterView
	schemaData *SchemaDBWriter
}

func MakeSchemaWriter(schemaData *SchemaDBWriter, writer io.WriteSeeker) SchemaWriter {
	return SchemaWriter{
		StreamWriterView: gobinary.NewStreamWriterView(writer, 1024),
		schemaData:       schemaData,
	}
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
