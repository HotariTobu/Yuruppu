# Custom Loaders and Regex Engines

## Custom URL Loader

```go
type CustomLoader struct{}

func (l *CustomLoader) Load(url string) (io.ReadCloser, error) {
    // Custom loading logic (e.g., from database)
    return io.NopCloser(strings.NewReader(schemaContent)), nil
}

c := jsonschema.NewCompiler()
c.UseLoader(&CustomLoader{})
```

## Custom Regex Engine

```go
type CustomRegexpEngine struct{}

func (e *CustomRegexpEngine) Compile(pattern string) (jsonschema.Regexp, error) {
    // Custom regex compilation
    return regexp.Compile(pattern)
}

c := jsonschema.NewCompiler()
c.UseRegexpEngine(&CustomRegexpEngine{})
```
