package goschema

type Schema interface {
	Fill([]SchemaEntry)
	Describe() []SchemaEntry
	ID() SchemaID
}

type SchemaID uint16

type SchemaEntry struct {
	Name   string
	Offset uint32
	Type   TypeCode
}

type Reference uint32

const ReferenceSize = 4

func (se SchemaEntry) IsValid() bool {
	return se.Type != InvalidType
}
