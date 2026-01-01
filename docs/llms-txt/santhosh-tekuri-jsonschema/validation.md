# Validation

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

## Validate HTTP Request Body

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
