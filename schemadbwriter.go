package goschema

import (
	"io"

	"github.com/chasingcarrots/gobinary"
)

type SchemaDBWriter struct {
	schemaIndex    map[SchemaID]SchemaDataEntry
	stream         *gobinary.StreamWriter
	writer         gobinary.HighLevelWriter
	originalOffset int64
}

func MakeSchemaDBWriter(stream *gobinary.StreamWriter) SchemaDBWriter {
	dbWriter := SchemaDBWriter{
		schemaIndex:    make(map[SchemaID]SchemaDataEntry),
		stream:         stream,
		originalOffset: stream.Offset(),
		writer:         gobinary.MakeHighLevelWriter(stream),
	}
	// reserve 2 bytes for the number of schemas
	dbWriter.writer.WriteUInt16(0)
	return dbWriter
}

func (sd *SchemaDBWriter) FindSchema(id SchemaID) (SchemaDataEntry, bool) {
	entry, ok := sd.schemaIndex[id]
	return entry, ok
}

func (sd *SchemaDBWriter) RegisterSchema(schema Schema) int {
	if entry, ok := sd.schemaIndex[schema.ID()]; ok {
		return entry.index
	}
	entries := schema.Describe()
	sd.writer.WriteUInt16(uint16(len(entries)))
	for i := range entries {
		sd.writer.WriteUInt16(uint16(len(entries[i].Name)))
		sd.writer.WriteString(entries[i].Name)
		sd.writer.WriteUInt8(uint8(entries[i].Type))
		sd.writer.WriteUInt32(entries[i].Offset)
	}
	idx := len(sd.schemaIndex)
	sd.schemaIndex[schema.ID()] = SchemaDataEntry{
		schema: schema,
		index:  idx,
	}
	return idx
}

func (sd *SchemaDBWriter) Close() {
	offset := sd.stream.Offset()
	sd.stream.Seek(sd.originalOffset, io.SeekStart)
	sd.writer.WriteUInt16(uint16(len(sd.schemaIndex)))
	sd.stream.Seek(offset, io.SeekStart)
}

type SchemaDataEntry struct {
	schema Schema
	index  int
}

func (sde *SchemaDataEntry) Schema() Schema { return sde.schema }
func (sde *SchemaDataEntry) Index() int     { return sde.index }
