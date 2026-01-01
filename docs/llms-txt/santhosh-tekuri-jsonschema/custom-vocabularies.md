# Custom Vocabularies (Draft 2019-09+)

```go
vocab := &jsonschema.Vocabulary{
    URL:    "http://example.com/meta/custom",
    Schema: compiledSchema,
    Subschemas: []jsonschema.SchemaPath{
        {jsonschema.Prop("customKeyword")},
    },
    Compile: func(ctx *jsonschema.CompilerContext,
                  obj map[string]any) (jsonschema.SchemaExt, error) {
        // Custom compilation logic
        return nil, nil
    },
}

c := jsonschema.NewCompiler()
c.AssertVocabs()
c.RegisterVocabulary(vocab)
```
