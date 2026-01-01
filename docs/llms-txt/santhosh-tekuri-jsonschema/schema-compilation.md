# Schema Compilation

## Compile from Files

```go
c := jsonschema.NewCompiler()
sch, err := c.Compile("./schema.json")
if err != nil {
    log.Fatal(err)
}
```

## Compile from Strings/In-Memory Data

```go
schema, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
    "type": "object",
    "properties": {
        "name": { "type": "string" }
    }
}`))
if err != nil {
    log.Fatal(err)
}

c := jsonschema.NewCompiler()
if err := c.AddResource("schema.json", schema); err != nil {
    log.Fatal(err)
}
sch, err := c.Compile("schema.json")
```

## Compile from URLs

```go
c := jsonschema.NewCompiler()
sch, err := c.Compile("https://example.com/schema.json")
if err != nil {
    log.Fatal(err)
}
```

## Must Compile (Panics on Error)

```go
c := jsonschema.NewCompiler()
sch := c.MustCompile("./schema.json")
```

## Compile Multiple Schemas

```go
c := jsonschema.NewCompiler()

// Add all schemas first
c.AddResource("base.json", baseSchema)
c.AddResource("user.json", userSchema)
c.AddResource("order.json", orderSchema)

// Compile specific schema (can reference others)
sch, err := c.Compile("order.json")
```
