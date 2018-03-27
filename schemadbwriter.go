package goschema

import (
	"io"

	"github.com/chasingcarrots/gobinary"
)

type SchemaDBWriter struct {
	schemaIndex    map[SchemaID]SchemaDataEntry
	schemaBlob     *gobinary.BufferedStreamWriter
	originalOffset int64
}

func MakeSchemaDBWriter(writer io.WriteSeeker) SchemaDBWriter {
	blob := gobinary.NewBufferedStreamWriter(writer, 1024)
	// reserve 4 bytes for the number of schemas
	blob.WriteUInt32(0)
	return SchemaDBWriter{
		schemaIndex:    make(map[SchemaID]SchemaDataEntry),
		schemaBlob:     blob,
		originalOffset: blob.Offset() - 4,
	}
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
	sd.schemaBlob.WriteUInt16(uint16(len(entries)))
	for i := range entries {
		sd.schemaBlob.WriteUInt16(uint16(len(entries[i].Name)))
		sd.schemaBlob.WriteString(entries[i].Name)
		sd.schemaBlob.WriteUInt8(uint8(entries[i].Type))
		sd.schemaBlob.WriteUInt32(entries[i].Offset)
	}
	idx := len(sd.schemaIndex)
	sd.schemaIndex[schema.ID()] = SchemaDataEntry{
		schema: schema,
		index:  idx,
	}
	return idx
}

func (sd *SchemaDBWriter) Close() {
	sd.schemaBlob.Seek(sd.originalOffset, io.SeekStart)
	sd.schemaBlob.WriteUInt32(uint32(len(sd.schemaIndex)))
	sd.schemaBlob.Flush()
}

func (sd *SchemaDBWriter) Flush() {
	sd.schemaBlob.Flush()
}

type SchemaDataEntry struct {
	schema Schema
	index  int
}

func (sde *SchemaDataEntry) Schema() Schema { return sde.schema }
func (sde *SchemaDataEntry) Index() int     { return sde.index }
