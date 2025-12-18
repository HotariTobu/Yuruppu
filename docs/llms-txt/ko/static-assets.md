# Static Assets

## Overview

Ko provides the ability to bundle static assets directly into container images produced by the tool.

## Convention and Setup

By convention, Ko recognizes any directory named `<importpath>/kodata/` and automatically includes its contents in the image. The path where it's available in the image will be identified by the environment variable `KO_DATA_PATH`.

## Example Structure

A typical project layout for bundling static content:

```
cmd/
  app/
    main.go
    kodata/
      favicon.ico
      index.html
```

## Implementation Example

In your Go application, serve static files using the environment variable:

```go
package main

import (
    "log"
    "net/http"
    "os"
)

func main() {
    http.Handle("/", http.FileServer(http.Dir(os.Getenv("KO_DATA_PATH"))))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Local Development

To test the behavior outside a container, manually set the environment variable during development:

```bash
KO_DATA_PATH=cmd/app/kodata/ go run ./cmd/app
```

## Additional Features

### Symlinks

Symlinks within `kodata` are followed and included automatically, enabling use cases like embedding Git commit information.

### Timestamps

By default, `http.FileServer` does not embed timestamps, but you can enable this by setting `KO_DATA_DATE_EPOCH` during the build process.
