# Go Schema
`goschema` is a Go serialization library that serializes (almost) arbitrary data into a binary format. A central notion is that of a *schema* that rules how a certain type is serialized. These schemata are generated at compile time for the types that should be serialized.

The schema generator natively supports the serialization of primitive types (except for `complex`) and nested slices, maps, structs, and pointers of those. The binary format is self-describing in the sense that it contains the field names and serialized types for all serialized structs. To keep the overhead of this data small, serialization creates two artifacts: First, the serialized data, and second, the schema descriptors (often just referred to as *schema*). A *schema* is then nothing more but a list of field names with offsets that describe at what point relative to the start of the data of the currently serialized object the data for a specific field can be found.

## Usage
Usage consists of two steps:
1. Using the generator to generate the schemata for whatever types should be serialized.
2. Using the `SchemaWriter` and `SchemaReader` structs to read and write schema data, and the `SchemaDB` and `SchemaDBWriter` structs to write schema descriptors.

## Example
Assume that we want to serialize the following type:
```golang
package subpkg

type TestType struct {
    MyList []string
}
```
Then the following is the Go-program that should be used to generate the schema for `TestType`. Note that the schema is a Go file:
```golang
package main

import (
    "path/filepath"
    "reflect"

    "github.com/chasingcarrots/goschema"
    "github.com/chasingcarrots/goschema/generator"
)

func main() {
    goPath, ok := os.LookupEnv("GOPATH")
    if !ok {
        fmt.Println("Could not find GOPATH in environment")
        return
    }
    join := filepath.Join
    // the path of the go template that is used to generate schema files; it
    // comes with this package
    schemaTemplatePath := join(goPath, "src", "github.com", "chasingcarrots", "goschema", "schemaimpl.got")
    // the path in which the generated schema files should be placed
    outputPath := join(goPath, "src", "github.com/chasingcarrots/schematest/output")
    // the path of the package that the generated schemata should live in
    packagePath := "github.com/chasingcarrots/schematest/output"
    gen := generator.NewContext(
        outputPath,
        packagePath,
        schemaTemplatePath,
        // These two types determine the types of additional context information
        // that is passed into reading and writing methods.
        reflect.TypeOf(map[string]interface{}{}),
        reflect.TypeOf(map[string]interface{}{}),
    )
    // add the default serializers to this schema generator
    gen.AddDefaultSerializers()
    // add the type that we want to serialize
    gen.RequestSchema(reflect.TypeOf(subpkg.TestType{}), "TestType")
    gen.Generate()
}
```

You could run this program with `go generate` to automate the process of generating schemata.

Now you can use the generated schema as follows:
```golang
package main

import (
    "bytes"
    "fmt"

    "github.com/chasingcarrots/gobinary"

    "github.com/chasingcarrots/goschema"
    "github.com/chasingcarrots/schematest/output"
    "github.com/chasingcarrots/schematest/subpkg"
)

func main() {
    test := TestType{
        MyList: []string{"a", "b", "c", "d", "e"},
    }

    // setup a byte buffer to hold the schema-descriptor that will be written out
    var schemaDBBuf gobinary.WriteBuffer
    schemaDBWriter := goschema.MakeSchemaDBWriter(gobinary.NewStreamWriter(&schemaDBBuf))

    // setup a byte buffer for the serialized data
    var schemaDataBuf gobinary.WriteBuffer
    schemaWriter := goschema.MakeSchemaWriter(
        &schemaDBWriter,
        gobinary.MakeStreamWriterView(gobinary.NewStreamWriter(&schemaDataBuf)),
    )
    
    // acquire schema for our type, writing it to the database if required
    testSchema := output.WriteTestTypeSchema(&schemaWriter)
    // perform a single write to the schemaWriter, serializing the `test1` object;
    // the last parameter is an optional context-object -- none of the default 
    // serializers rely on it, but custom made ones may do.
    testSchema.SingleWrite(&schemaWriter, &test1, nil)
    
    // finish writing
    schemaDBWriter.Close()

    // print out the contents of the byte buffers for manual verification
    fmt.Println(schemaDataBuf.Bytes())
    fmt.Println(schemaDBBuf.Bytes())
    
    // read schema back in
    schemaDB := goschema.MakeSchemaDB()
    // use Fill to read schema descriptors
    schemaDB.Fill(bytes.NewReader(schemaDBBuf.Bytes()))
    schemaReader := goschema.MakeSchemaReader(
        &schemaDB,
        gobinary.MakeStreamReaderView(
            gobinary.NewStreamReader(bytes.NewReader(schemaDataBuf.Bytes())),
        ),
    )
    
    // Deserialize the object that was written above
    testDeserialized := subpkg.TestType{}
    output.ReadTestTypeSchema(&schemaReader).SingleRead(&schemaReader, &testDeserialized, nil)
    // print out both objects
    fmt.Println(test1, testDeserialized)
}
```

## Data Types & Serialization Details
`goschema` knows about all the basic data types, slices (= lists), and maps. Structs are serialized via schemata. Serialization always starts with a schema describing a struct. When a schema is written for the first time, it serializes itself using the `SchemaDBWriter`. Subsequent writes with a schema of that type will not cause any more schema descriptors to be written out. The, serialization thus proceeds as follows:

1. First, call `WriteTestTypeSchema`. This tries to acquire the requested schema. If it has already been used before, it will be reused. Otherwise, generate a new instance of the schema and serialize it to the `SchemaDBWriter`. This writes out the offset of any field in the serialized data along with a `TypeCode` that describes what kind of data lives here. There is a `TypeCode` for each supported primitive types, one for lists, one for maps, and one for schema types. Custom data that is serialized in place (i.e. values such as mathematical vectors whose definition is not expected to ever change) can define their own type codes.
Then, independently of whether the schema has been newly generated or found, write the `uint16` index of the schema descriptor in the schema database to the `SchemaWriter`.
2. Write out the length of the data blob that follows. This is of course written after the following step has finished.
3. For each field of the struct that is not marked with `schemaIgnore:""` as a tag, serialize its contents with `SchemaWriter`:
    1. Primitive values of fixed size (numbers, bool) are written out immediately. 
    2. Non-struct types that are structurally equivalent to a primitive type are serialized as such, e.g. `type ID uint32` is serialized as a `uint16`.
    3. Lists, maps, pointers, and schemata store a 32bit reference (= an offset from the beginning of the current schema object) to their actual data, which follows once all fields of this schema have been written. The data for lists is the number of elements in the list, followed by the `TypeCode` of the element types. If that code is the code for schemata, this is followed by the `uint16` index of the schema for the items in the list. For maps, this work similarly but includes two `TypeCode`s. Schemata simply store the index `uint16` of the schema of the type to serialize. Pointers use a 1 byte binary encoding of null-ness instead of a length but otherwise work like lists -- which means that pointers after deserialization, pointers *never* alias, i.e. each pointer points to its own copy of the data!

## Deserialization Details
Deserialization works similarly. The main point is that whenever a schema reference, list, or map of schema typed object is deserialized, the callling code that triggered the deserialization can use the information stored in the schema descriptors to find out whether fields have been removed. Specifically, the calling code always knows what kind of schema it wants to read and that schema can then be filled from the schema descriptors with the offsets of the data that is present in the file. If a required field is not present, reading that fields returns a default value. This ensures a certain degree of backwards-compatibility. More elaborate features to support versioning could be built on top of this.

## Marking Data for Serialization
When a schema is requested for a type, the generator will automatically also generate schemata for all contained types for which it knows how to serialize them.
There are three tags that can be applied to fields in a struct to influence serialization:

 * `schemaIgnore:""` instructs the generator to ignore fields,
 * `schemaName:"your_name_here"` instructs the generator to use a specific name for a field for serialization purposes,
 * `schemaDefault:"default_value"` specifies a default value for a field in case it is not found in the data.


## Custom Serialization
`goschema` supports custom serializers (or rather, custom generators for serializers). When creating a context as in the example above, you can add your own serializers. A common use case would be to add custom primitive types such as a 2-value vector: `type Vector2 struct { x,y float }`. Such values have a known structure and size and can be serialized in place. An easy way to achieve this is to use the `InlineSerializer` that takes a type and a `TypeCode` to use for the serialized primitives:
```golang
gen.AddDefaultSerializers()
gen.AddSerializers(
    generator.NewInlineSerializer(reflect.TypeOf(subtest.Vector2{}), goschema.TypeCode(255)),
)
```
In this example, the type code chosen is 255 (typecodes are 8bit integers). We recommend starting with the highest values and working your way backwards as to not be disturbed by updates to the library that add new lower typecodes (`goschema.NumTypeCodes` is the least `TypeCode` not in use by the basic primitive types, but it may change over time).
To implement custom serializers, take a look at the files in the `generator` directory, especially the `serializer.go` file which contains the interface that serializers need to implement. Your custom serializers can make use of the context value passed to reading and writing methods by using the name `context`. They also have access to all tags declared on fields that they are used on.

Note that the order in which serializers are added to a generator context is very important. When looking for a serializer to use, the least recently added serializers are queried first until a match is found. As such, it is always a good idea to add the default serializers first.

## Why All of This?
This library is motivated by two factors: First, in our use case we are serializing many objects of just a few types. Hence it makes sense to decouple the description of the data (= schema) from the actual contents. Second, during development fields will be introduced, removed, renamed etc., which means that the serialization system needs a way to deal with that gracefully. By letting schemata find the offsets for their data during serialization, the format survives variations in the data layout.

In comparison to flatbuffers, this package does not require any external specifications of the serialized data. As an added bonus, you can also make sense from the data without any information about who will read it. On the downside, the format is neither optimized for size nor for speed.