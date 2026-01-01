# CLI Tool

The library includes a command-line tool for validation.

## Installation

```bash
go install github.com/santhosh-tekuri/jsonschema/cmd/jv@latest
```

## Usage

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

## Exit Codes

- `0` - Valid
- `1` - Validation error
- `2` - Usage error

## Output Formats

`simple` (default), `alt`, `flag`, `basic`, `detailed`
