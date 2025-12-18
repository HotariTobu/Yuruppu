# Debugging

## Overview

The `ko` tool supports debugging Go applications using [delve](https://github.com/go-delve/delve), enabling developers to iteratively explore and troubleshoot app behavior efficiently.

## Key Features

The `--debug` flag modifies the image build process in three ways:

1. **Installs delve** - The debugger is included alongside your application
2. **Sets ENTRYPOINT** - Configures the image to run via `delve exec` command, listening on port `40000`
3. **Preserves debug symbols** - Ensures compiled Go binaries include necessary debugging information

**Important**: This feature is designed for development only and should never be used in production environments.

## Usage Instructions

### Step 1: Build with Debug Flag

```bash
ko build --debug ./cmd/app
```

### Step 2: Run Container with Exposed Port

```bash
docker run -p 40000:40000 $(ko build --debug ./cmd/app)
```

This exposes the debug port `40000`, allowing debugger clients to establish connections to the running container.

### Step 3: Connect Debugger

Attach your debugger client to the container listening on the exposed port to begin interactive debugging.

For example, with Delve CLI:

```bash
dlv connect localhost:40000
```

Or configure your IDE (VS Code, GoLand, etc.) to connect to the remote debugger at `localhost:40000`.

## Related Options

Use `--disable-optimizations` flag to further improve debugging experience by preventing Go compiler optimizations that can make stepping through code difficult.

```bash
ko build --debug --disable-optimizations ./cmd/app
```
