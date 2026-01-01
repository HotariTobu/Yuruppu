# Format Assertions

Format validation is optional in Draft 2019-09+ by default. Enable with `AssertFormat()`:

```go
c := jsonschema.NewCompiler()
c.AssertFormat()  // Enable format assertions
sch, err := c.Compile("schema.json")
```

## Built-in Formats

`regex`, `uuid`, `ipv4`, `ipv6`, `hostname`, `email`, `date`, `time`, `date-time`, `duration`, `json-pointer`, `relative-json-pointer`, `uri`, `uri-reference`, `uri-template`, `iri`, `iri-reference`, `period`, `semver`

## Custom Format Validators

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
