# Main Types

## Schema

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

## Compiler

Compiles schemas into executable validators:

```go
type Compiler struct {
    // contains filtered or unexported fields
}

func NewCompiler() *Compiler
```

### Key Methods

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

## ValidationError

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
