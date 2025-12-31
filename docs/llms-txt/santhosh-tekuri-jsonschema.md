# santhosh-tekuri/jsonschema

> Go library for validating JSON data against JSON Schema specifications. Supports Draft 4, 6, 7, 2019-09, and 2020-12. Passes JSON-Schema-Test-Suite with high compliance rates.

**Version:** v6.0.2
**Import:** `github.com/santhosh-tekuri/jsonschema/v6`

This library provides robust JSON Schema validation with detailed error reporting, custom format validators, and support for content assertions. Key features include cycle detection, vocabulary-based validation, and introspectable validation errors.

## Quick Start

### Installation

```go
import "github.com/santhosh-tekuri/jsonschema/v6"
```

### Basic Usage

```go
package main

import (
    "encoding/json"
    "log"

    "github.com/santhosh-tekuri/jsonschema/v6"
)

func main() {
    // Compile schema
    schemaData := []byte(`{
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer", "minimum": 0}
        },
        "required": ["name"]
    }`)

    schema, err := jsonschema.Compile(schemaData)
    if err != nil {
        log.Fatal(err)
    }

    // Validate instance
    instanceData := []byte(`{"name": "John", "age": 30}`)
    var instance interface{}
    json.Unmarshal(instanceData, &instance)

    if err := schema.Validate(instance); err != nil {
        log.Fatal(err)
    }
}
```

## Schema Compilation

### Compile from Files

```go
c := jsonschema.NewCompiler()
sch, err := c.Compile("./schema.json")
if err != nil {
    log.Fatal(err)
}
```

### Compile from Strings/In-Memory Data

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

### Compile from URLs

```go
c := jsonschema.NewCompiler()
sch, err := c.Compile("https://example.com/schema.json")
if err != nil {
    log.Fatal(err)
}
```

### Must Compile (Panics on Error)

```go
c := jsonschema.NewCompiler()
sch := c.MustCompile("./schema.json")
```

## Validating map[string]any (Go Native Types)

### Recommended: Use UnmarshalJSON

The library provides `jsonschema.UnmarshalJSON()` to unmarshal JSON data without losing number precision. This is critical for validating numeric constraints correctly.

```go
import (
    "strings"
    "github.com/santhosh-tekuri/jsonschema/v6"
)

// Unmarshal JSON maintaining number precision
inst, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
    "name": "John",
    "age": 30,
    "balance": 123.45
}`))
if err != nil {
    log.Fatal(err)
}

// inst is of type any (can be map[string]any, []any, etc.)
err = schema.Validate(inst)
if err != nil {
    fmt.Println("Validation failed:", err)
}
```

### Standard json.Unmarshal (May Lose Precision)

```go
instanceData := []byte(`{"name": "John", "age": 30}`)
var instance map[string]any
json.Unmarshal(instanceData, &instance)

err := schema.Validate(instance)
if err != nil {
    fmt.Println("Validation failed:", err)
}
```

**Warning:** Standard `json.Unmarshal` may lose precision for large numbers. Use `jsonschema.UnmarshalJSON()` for accurate numeric validation.

## Error Handling

### Basic Error Handling

```go
err := schema.Validate(instance)
if err != nil {
    fmt.Println("Validation failed:", err.Error())
}
```

### Introspectable Validation Errors

Validation errors provide detailed information with hierarchical error causes:

```go
err := schema.Validate(instance)
if err != nil {
    ve, ok := err.(*jsonschema.ValidationError)
    if !ok {
        // Not a validation error (compilation/schema error)
        log.Fatal(err)
    }

    // Access error details
    fmt.Println("Schema URL:", ve.SchemaURL)
    fmt.Println("Instance Location:", strings.Join(ve.InstanceLocation, "/"))
    fmt.Println("Error Kind:", ve.ErrorKind)
    fmt.Println("Message:", ve.Error())

    // Iterate through child errors (nested validation failures)
    for _, cause := range ve.Causes {
        fmt.Printf("  - %s: %s\n",
            strings.Join(cause.InstanceLocation, "/"),
            cause.Error())
    }
}
```

### Error Output Formats

The `ValidationError` type supports multiple output formats:

```go
ve := err.(*jsonschema.ValidationError)

// Basic output (simple format)
fmt.Println(ve.BasicOutput())

// Detailed output (verbose format)
fmt.Println(ve.DetailedOutput())

// Flag output (boolean flags)
fmt.Println(ve.FlagOutput())

// Localized errors (i18n support)
p := message.NewPrinter(language.English)
fmt.Println(ve.LocalizedError(p))
```

### Error Types

- `ValidationError` - Instance validation failure (introspectable)
- `SchemaValidationError` - Schema itself is invalid
- `CompilationError` - Schema compilation failure
- `AnchorNotFoundError` - Invalid anchor reference
- `InvalidRegexError` - Invalid pattern regex
- `LoadURLError` - Failed to load schema URL
- `UnsupportedDraftError` - Unsupported $schema version

## Draft Version Configuration

### Default Draft (2020-12)

By default, schemas without explicit `$schema` use Draft 2020-12:

```go
c := jsonschema.NewCompiler()
sch, err := c.Compile("schema.json")
```

### Specify Draft Version

```go
c := jsonschema.NewCompiler()
c.DefaultDraft(jsonschema.Draft7)  // For schemas without $schema
sch, err := c.Compile("schema.json")
```

### Available Draft Versions

```go
var (
    Draft4    *Draft  // JSON Schema Draft 4
    Draft6    *Draft  // JSON Schema Draft 6
    Draft7    *Draft  // JSON Schema Draft 7
    Draft2019 *Draft  // JSON Schema Draft 2019-09
    Draft2020 *Draft  // JSON Schema Draft 2020-12
)
```

### Per-Schema Draft (Using $schema)

```json
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object"
}
```

The library will automatically use the draft specified in `$schema`.

## Thread Safety

### Compiled Schemas are Thread-Safe

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

### Compiler is NOT Thread-Safe

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

## Advanced Features

### Format Assertions

Format validation is optional in Draft 2019-09+ by default. Enable with `AssertFormat()`:

```go
c := jsonschema.NewCompiler()
c.AssertFormat()  // Enable format assertions
sch, err := c.Compile("schema.json")
```

**Built-in Formats:**
`regex`, `uuid`, `ipv4`, `ipv6`, `hostname`, `email`, `date`, `time`, `date-time`, `duration`, `json-pointer`, `relative-json-pointer`, `uri`, `uri-reference`, `uri-template`, `iri`, `iri-reference`, `period`, `semver`

### Custom Format Validators

```go
validatePalindrome := func(v any) error {
    s, ok := v.(string)
    if !ok {
        return nil
    }
    // Check if string is a palindrome
    runes := []rune(s)
    for i := 0; i < len(runes)/2; i++ {
        if runes[i] != runes[len(runes)-1-i] {
            return fmt.Errorf("not a palindrome")
        }
    }
    return nil
}

c := jsonschema.NewCompiler()
c.RegisterFormat(&jsonschema.Format{
    Name:     "palindrome",
    Validate: validatePalindrome,
})
c.AssertFormat()
```

### Content Assertions (Draft 7+)

Validate `contentEncoding` and `contentMediaType`:

```go
c := jsonschema.NewCompiler()
c.AssertContent()  // Enable content assertions
sch, err := c.Compile("schema.json")
```

### Custom Content Encoding

```go
c := jsonschema.NewCompiler()

c.RegisterContentEncoding(&jsonschema.Decoder{
    Name:   "hex",
    Decode: hex.DecodeString,
})

c.AssertContent()
```

### Custom Content Media Type

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

### Custom Vocabularies (Draft 2019-09+)

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

### Custom URL Loader

```go
type CustomLoader struct{}

func (l *CustomLoader) Load(url string) (io.ReadCloser, error) {
    // Custom loading logic (e.g., from database)
    return io.NopCloser(strings.NewReader(schemaContent)), nil
}

c := jsonschema.NewCompiler()
c.UseLoader(&CustomLoader{})
```

### Custom Regex Engine

```go
type CustomRegexpEngine struct{}

func (e *CustomRegexpEngine) Compile(pattern string) (jsonschema.Regexp, error) {
    // Custom regex compilation
    return regexp.Compile(pattern)
}

c := jsonschema.NewCompiler()
c.UseRegexpEngine(&CustomRegexpEngine{})
```

## Main Types

### Schema

The compiled schema representation:

```go
type Schema struct {
    DraftVersion int
    Location     string
    Bool         *bool              // boolean schema (true/false)
    ID           string             // $id
    Ref          *Schema            // $ref target
    Types        *Types             // allowed types
    Enum         *Enum              // enum values
    Not          *Schema            // not constraint
    AllOf, AnyOf, OneOf []*Schema   // composition
    If, Then, Else      *Schema     // conditionals

    // Object validation
    Properties            map[string]*Schema
    PatternProperties     map[Regexp]*Schema
    AdditionalProperties  any
    Required              []string

    // Array validation
    Items           any              // nil, []*Schema, or *Schema
    PrefixItems     []*Schema
    UniqueItems     bool
    MinItems, MaxItems *int

    // String validation
    MinLength, MaxLength *int
    Pattern              Regexp
    ContentEncoding      *Decoder
    ContentMediaType     *MediaType

    // Number validation
    Maximum, Minimum *big.Rat
    ExclusiveMaximum, ExclusiveMinimum *big.Rat
    MultipleOf *big.Rat
}

func (s *Schema) Validate(inst any) error
```

### Compiler

Compiles schemas into executable validators:

```go
type Compiler struct {
    // contains filtered or unexported fields
}

func NewCompiler() *Compiler
```

**Key Methods:**

```go
// Compile schema from URL or file path
func (c *Compiler) Compile(loc string) (*Schema, error)

// Compile with panic on error
func (c *Compiler) MustCompile(loc string) *Schema

// Add in-memory schema
func (c *Compiler) AddResource(url string, doc any) error

// Configuration
func (c *Compiler) DefaultDraft(d *Draft)
func (c *Compiler) AssertFormat()
func (c *Compiler) AssertContent()
func (c *Compiler) AssertVocabs()

// Customization
func (c *Compiler) RegisterFormat(f *Format)
func (c *Compiler) RegisterContentEncoding(d *Decoder)
func (c *Compiler) RegisterContentMediaType(mt *MediaType)
func (c *Compiler) RegisterVocabulary(vocab *Vocabulary)
func (c *Compiler) UseLoader(loader URLLoader)
func (c *Compiler) UseRegexpEngine(engine RegexpEngine)
```

### ValidationError

Detailed error information with hierarchy:

```go
type ValidationError struct {
    SchemaURL         string
    InstanceLocation  []string
    ErrorKind         ErrorKind
    Causes            []*ValidationError  // Nested errors
}

func (e *ValidationError) Error() string
func (e *ValidationError) GoString() string
func (e *ValidationError) BasicOutput() *OutputUnit
func (e *ValidationError) DetailedOutput() *OutputUnit
func (e *ValidationError) FlagOutput() *FlagOutput
func (e *ValidationError) LocalizedError(p *message.Printer) string
```

## CLI Tool

The library includes a command-line tool for validation:

### Installation

```bash
go install github.com/santhosh-tekuri/jsonschema/cmd/jv@latest
```

### Usage

```bash
# Basic validation
jv schema.json instance.json

# Specify draft version
jv -d 7 schema.json instance.json

# Enable format assertions
jv -f schema.json instance.json

# Enable content assertions
jv -c schema.json instance.json

# Use stdin
cat instance.json | jv schema.json -

# Output formats
jv -o basic schema.json instance.json
jv -o detailed schema.json instance.json
```

**Exit Codes:**
- `0` - Valid
- `1` - Validation error
- `2` - Usage error

**Output Formats:**
`simple` (default), `alt`, `flag`, `basic`, `detailed`

## Common Patterns

### Validate HTTP Request Body

```go
func ValidateRequest(schema *jsonschema.Schema, r *http.Request) error {
    inst, err := jsonschema.UnmarshalJSON(r.Body)
    if err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }

    if err := schema.Validate(inst); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    return nil
}
```

### Compile Multiple Schemas

```go
c := jsonschema.NewCompiler()

// Add all schemas first
c.AddResource("base.json", baseSchema)
c.AddResource("user.json", userSchema)
c.AddResource("order.json", orderSchema)

// Compile specific schema (can reference others)
sch, err := c.Compile("order.json")
```

### Collect All Validation Errors

```go
func collectErrors(ve *jsonschema.ValidationError, errs *[]string) {
    *errs = append(*errs, fmt.Sprintf("%s: %s",
        strings.Join(ve.InstanceLocation, "/"),
        ve.Error()))

    for _, cause := range ve.Causes {
        collectErrors(cause, errs)
    }
}

var allErrors []string
if err := schema.Validate(inst); err != nil {
    if ve, ok := err.(*jsonschema.ValidationError); ok {
        collectErrors(ve, &allErrors)
    }
}
```

## Important Notes

- **Number Precision:** Use `jsonschema.UnmarshalJSON()` instead of `json.Unmarshal()` to preserve number precision
- **Thread Safety:** Compiled schemas are thread-safe; compilers are not
- **Cycle Detection:** Library detects and prevents infinite loops in both schema references and validation
- **Draft Support:** Schemas can specify their own draft via `$schema`; default is Draft 2020-12
- **Format Validators:** Built-in formats are available but must be enabled with `AssertFormat()` for Draft 2019-09+
- **File Formats:** Supports both JSON and YAML file formats
- **URL Schemes:** Supports file://, http://, and https:// URL schemes
- **Compliance:** Passes JSON-Schema-Test-Suite excluding optional tests

## References

- GitHub Repository: https://github.com/santhosh-tekuri/jsonschema
- Go Package Documentation: https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6
- JSON Schema Specifications: https://json-schema.org/specification.html
