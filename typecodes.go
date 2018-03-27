package goschema

type TypeCode uint8

const (
	SchemaType   TypeCode = 0x0
	MapType      TypeCode = 0x1
	ListType     TypeCode = 0x2
	UInt8Type    TypeCode = 0x3
	UInt16Type   TypeCode = 0x4
	UInt32Type   TypeCode = 0x5
	UInt64Type   TypeCode = 0x6
	UIntType     TypeCode = 0x7
	Int8Type     TypeCode = 0x8
	Int16Type    TypeCode = 0x9
	Int32Type    TypeCode = 0xA
	Int64Type    TypeCode = 0xB
	IntType      TypeCode = 0xC
	Float32Type  TypeCode = 0xD
	Float64Type  TypeCode = 0xE
	BoolType     TypeCode = 0xF
	StringType   TypeCode = 0x10
	NumTypeCodes TypeCode = 0x11
)
