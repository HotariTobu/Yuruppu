# Common Patterns

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

## Collect All Validation Errors

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
