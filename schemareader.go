package goschema

import (
	"io"

	"github.com/chasingcarrots/gobinary"
)

type SchemaReader struct {
	gobinary.HighLevelReader
	gobinary.StreamReaderView
	schemaDB *SchemaDB
}

func MakeSchemaReader(schemaDB *SchemaDB, stream *gobinary.StreamReader) SchemaReader {
	sr := SchemaReader{
		schemaDB:         schemaDB,
		StreamReaderView: gobinary.MakeStreamReaderView(stream),
	}
	sr.HighLevelReader = gobinary.MakeHighLevelReader(stream)
	return sr
}

// ReadReference reads a reference from the current position and seeks to
// the offset denoted by the reference. It returns offset in global coordinates
// that the reader should return to after reading what is referenced.
func (sr *SchemaReader) ReadReference() int64 {
	current := sr.StreamReaderView.Offset()
	ref := sr.ReadUInt32()
	sr.Seek(int64(ref), io.SeekStart)
	return current + ReferenceSize
}

func (sr *SchemaReader) FindSchema(schemaIndex int) (Schema, []SchemaEntry) {
	return sr.schemaDB.FindSchema(schemaIndex)
}

func (sr *SchemaReader) RegisterSchema(schemaIndex int, schema Schema) {
	sr.schemaDB.RegisterSchema(schemaIndex, schema)
}

func (sr *SchemaReader) ReadInt() int {
	return int(sr.ReadInt64())
}

func (sr *SchemaReader) ReadUInt() uint64 {
	return uint64(sr.ReadUInt64())
}
