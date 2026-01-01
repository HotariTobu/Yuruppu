# Important Notes

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
