# Content Assertions (Draft 7+)

Validate `contentEncoding` and `contentMediaType`:

```go
c := jsonschema.NewCompiler()
c.AssertContent()  // Enable content assertions
sch, err := c.Compile("schema.json")
```

## Custom Content Encoding

```go
c := jsonschema.NewCompiler()

c.RegisterContentEncoding(&jsonschema.Decoder{
    Name:   "hex",
    Decode: hex.DecodeString,
})

c.AssertContent()
```

## Custom Content Media Type

```go
c := jsonschema.NewCompiler()

c.RegisterContentMediaType(&jsonschema.MediaType{
    Name: "application/xml",
    Validate: func(b []byte) error {
        return xml.Unmarshal(b, new(any))
    },
    UnmarshalJSON: nil,
})

c.AssertContent()
```
