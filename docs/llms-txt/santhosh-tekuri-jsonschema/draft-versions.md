# Draft Version Configuration

## Default Draft (2020-12)

By default, schemas without explicit `$schema` use Draft 2020-12:

```go
c := jsonschema.NewCompiler()
sch, err := c.Compile("schema.json")
```

## Specify Draft Version

```go
c := jsonschema.NewCompiler()
c.DefaultDraft(jsonschema.Draft7)  // For schemas without $schema
sch, err := c.Compile("schema.json")
```

## Available Draft Versions

```go
var (
    Draft4    *Draft  // JSON Schema Draft 4
    Draft6    *Draft  // JSON Schema Draft 6
    Draft7    *Draft  // JSON Schema Draft 7
    Draft2019 *Draft  // JSON Schema Draft 2019-09
    Draft2020 *Draft  // JSON Schema Draft 2020-12
)
```

## Per-Schema Draft (Using $schema)

```json
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object"
}
```

The library will automatically use the draft specified in `$schema`.
