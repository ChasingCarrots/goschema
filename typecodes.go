package goschema

type TypeCode uint8

const (
	InvalidType TypeCode = iota
	MapType
	ListType
	UInt8Type
	UInt16Type
	UInt32Type
	UInt64Type
	UIntType
	Int8Type
	Int16Type
	Int32Type
	Int64Type
	IntType
	Float32Type
	Float64Type
	BoolType
	StringType
	SchemaType
	NumTypeCodes
)
