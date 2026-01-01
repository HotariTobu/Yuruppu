# Thread Safety

## Compiled Schemas are Thread-Safe

Once a `*Schema` is compiled, it is immutable and can be safely used across multiple goroutines:

```go
c := jsonschema.NewCompiler()
schema, _ := c.Compile("schema.json")

// Safe to use schema.Validate() concurrently
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        schema.Validate(someData)  // Thread-safe
    }()
}
wg.Wait()
```

## Compiler is NOT Thread-Safe

Do not share a `*Compiler` across goroutines during compilation. Create separate compilers or use synchronization:

```go
// BAD: Concurrent compilation with shared compiler
c := jsonschema.NewCompiler()
go c.Compile("schema1.json")  // NOT SAFE
go c.Compile("schema2.json")  // NOT SAFE

// GOOD: Separate compilers per goroutine
go func() {
    c := jsonschema.NewCompiler()
    c.Compile("schema1.json")
}()
go func() {
    c := jsonschema.NewCompiler()
    c.Compile("schema2.json")
}()
```
