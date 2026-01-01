# Quick Start

## Installation

```go
import "github.com/santhosh-tekuri/jsonschema/v6"
```

## Basic Usage

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
