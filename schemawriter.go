package goschema

import (
	"io"

	"github.com/chasingcarrots/gobinary"
)

type SchemaWriter struct {
	gobinary.StreamWriterView
	schemaData *SchemaDBWriter
	globalEnd  int64
}

func NewSchemaWriter(schemaData *SchemaDBWriter, writer io.WriteSeeker) *SchemaWriter {
	return &SchemaWriter{
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

// WriteReference writes a reference to the end of the current stream, seeks to
// said end of the stream, and returns the offset to return to.
func (sw *SchemaWriter) WriteReference() int64 {
	current := sw.Offset()
	end := sw.Local(sw.globalEnd)
	sw.WriteUInt32(uint32(end))
	sw.Seek(end, io.SeekStart)
	return current + ReferenceSize
}

func (sw *SchemaWriter) WriteInt(value int) {
	sw.WriteInt64(int64(value))
}

func (sw *SchemaWriter) WriteUInt(value uint) {
	sw.WriteUInt64(uint64(value))
}
