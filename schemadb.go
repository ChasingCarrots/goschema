package goschema

import (
	"github.com/chasingcarrots/gobinary"
)

type SchemaDB struct {
	rawSchemata map[int][]SchemaEntry
	schemata    map[int]Schema
}

func NewSchemaDB() *SchemaDB {
	return &SchemaDB{
		rawSchemata: make(map[int][]SchemaEntry),
		schemata:    make(map[int]Schema),
	}
}

func (sdb *SchemaDB) FindSchema(schemaIndex int) (Schema, []SchemaEntry) {
	raw, ok := sdb.rawSchemata[schemaIndex]
	if !ok {
		return nil, nil
	}
	schema, ok := sdb.schemata[schemaIndex]
	if !ok {
		return nil, raw
	}
	return schema, raw
}

func (sdb *SchemaDB) RegisterSchema(schemaIndex int, schema Schema) {
	sdb.schemata[schemaIndex] = schema
}

func (sdb *SchemaDB) Fill(reader gobinary.HighLevelReader) {
	n := int(reader.ReadUInt32())
	for s := 0; s < n; s++ {
		length := int(reader.ReadUInt16())
		schema := make([]SchemaEntry, length, length)
		for i := 0; i < length; i++ {
			schema[i].Name = reader.ReadString(int(reader.ReadUInt16()))
			schema[i].Type = TypeCode(reader.ReadUInt8())
			schema[i].Offset = reader.ReadUInt32()
		}
		sdb.rawSchemata[s] = schema
	}
}
