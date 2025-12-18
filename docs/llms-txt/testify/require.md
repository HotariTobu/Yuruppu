# Require Package

> The require package provides fatal assertion functions that call `t.FailNow()` and stop test execution immediately on failure. It mirrors the assert package API but uses a fail-fast approach instead of returning boolean results.

**Import:** `github.com/stretchr/testify/require`

## Key Difference from Assert

| Feature | assert | require |
|---------|--------|---------|
| On failure | Returns `false`, test continues | Calls `t.FailNow()`, test stops |
| Use case | Collect all failures | Stop on first failure |
| Goroutine safety | Can be called from any goroutine | Must be called from test goroutine |

## When to Use Require

Use `require` for preconditions that must be true before continuing:

```go
func TestUserOperations(t *testing.T) {
    user, err := CreateUser("test@example.com")
    require.NoError(t, err)      // Stop if user creation fails
    require.NotNil(t, user)       // Stop if user is nil

    // Only execute if above conditions pass
    assert.Equal(t, "test@example.com", user.Email)
    assert.True(t, user.Active)
}
```

## Goroutine Safety Warning

**Important:** `require` functions must be called from the goroutine running the test function. Calling from other goroutines will cause race conditions.

```go
// WRONG - Will cause race condition
func TestConcurrent(t *testing.T) {
    go func() {
        require.Equal(t, expected, actual)  // UNSAFE
    }()
}

// CORRECT - Use assert in goroutines
func TestConcurrent(t *testing.T) {
    go func() {
        assert.Equal(t, expected, actual)  // SAFE
    }()
}

// ALSO CORRECT - Use CollectT for goroutines
func TestConcurrent(t *testing.T) {
    wg := sync.WaitGroup{}
    wg.Add(1)
    go func() {
        defer wg.Done()
        c := new(assert.CollectT)
        assert.Equal(c, expected, actual)
        if c.Failed() {
            t.FailNow()  // Called from main goroutine
        }
    }()
    wg.Wait()
}
```

## Basic Usage

### Direct Function Calls

```go
import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
    require.Equal(t, expected, actual, "values should be equal")
    require.NotNil(t, object, "object should not be nil")
}
```

### Using Assertions Type

```go
func TestSomething(t *testing.T) {
    require := require.New(t)

    require.Equal(expected, actual)
    require.NotNil(object)
}
```

## Common Functions

All functions from the assert package are available in require with identical signatures:

### Equality

```go
require.Equal(t, expected, actual)
require.NotEqual(t, unexpected, actual)
require.EqualValues(t, expected, actual)
require.Exactly(t, expected, actual)
require.EqualExportedValues(t, expected, actual)
```

### Nil and Empty

```go
require.Nil(t, object)
require.NotNil(t, object)
require.Empty(t, value)
require.NotEmpty(t, value)
require.Zero(t, value)
require.NotZero(t, value)
```

### Boolean

```go
require.True(t, condition)
require.False(t, condition)
```

### Comparison

```go
require.Greater(t, e1, e2)
require.GreaterOrEqual(t, e1, e2)
require.Less(t, e1, e2)
require.LessOrEqual(t, e1, e2)
require.Positive(t, value)
require.Negative(t, value)
```

### Errors

```go
require.Error(t, err)
require.NoError(t, err)
require.EqualError(t, err, "expected error message")
require.ErrorContains(t, err, "substring")
require.ErrorIs(t, err, target)
require.ErrorAs(t, err, &target)
```

### Collections

```go
require.Contains(t, collection, element)
require.NotContains(t, collection, element)
require.Len(t, collection, length)
require.ElementsMatch(t, expected, actual)
require.Subset(t, list, subset)
```

### Types

```go
require.IsType(t, expectedType, object)
require.Implements(t, (*Interface)(nil), object)
```

### Panics

```go
require.Panics(t, func() { panic("error") })
require.NotPanics(t, func() { /* safe code */ })
require.PanicsWithValue(t, "expected", func() { panic("expected") })
require.PanicsWithError(t, "message", func() { panic(errors.New("message")) })
```

### JSON/YAML

```go
require.JSONEq(t, expectedJSON, actualJSON)
require.YAMLEq(t, expectedYAML, actualYAML)
```

### Regular Expressions

```go
require.Regexp(t, pattern, value)
require.NotRegexp(t, pattern, value)
```

### Files

```go
require.FileExists(t, path)
require.NoFileExists(t, path)
require.DirExists(t, path)
require.NoDirExists(t, path)
```

### Time

```go
require.WithinDuration(t, expected, actual, delta)
require.WithinRange(t, actual, start, end)
```

### HTTP

```go
require.HTTPSuccess(t, handler, method, url, values)
require.HTTPError(t, handler, method, url, values)
require.HTTPStatusCode(t, handler, method, url, values, statusCode)
require.HTTPBodyContains(t, handler, method, url, values, substring)
```

### Async Conditions

```go
require.Eventually(t, condition, timeout, polling, "message")
require.Never(t, condition, duration, polling, "message")
require.EventuallyWithT(t, func(c *require.CollectT) {
    // assertions with c
}, timeout, polling)
```

## Formatted Variants

All functions have formatted variants ending in `f`:

```go
require.Equalf(t, expected, actual, "comparing %s: got %v", name, actual)
require.NoErrorf(t, err, "operation %s failed", operation)
require.NotNilf(t, obj, "expected %s to exist", objName)
```

## Usage Patterns

### Pattern 1: Precondition Checks

Use require for setup preconditions:

```go
func TestDatabaseOperations(t *testing.T) {
    db, err := OpenDatabase()
    require.NoError(t, err)           // Stop if DB fails to open
    require.NotNil(t, db)              // Stop if db is nil
    defer db.Close()

    // All subsequent code can safely use db
    result := db.Query("SELECT * FROM users")
    assert.NotEmpty(t, result)
}
```

### Pattern 2: Resource Initialization

```go
func TestFileProcessing(t *testing.T) {
    file, err := os.Open("test.txt")
    require.NoError(t, err)           // Stop if file doesn't exist
    defer file.Close()

    content, err := io.ReadAll(file)
    require.NoError(t, err)           // Stop if read fails
    require.NotEmpty(t, content)      // Stop if file is empty

    // Process content knowing it's valid
    assert.Contains(t, string(content), "expected text")
}
```

### Pattern 3: Sequential Dependencies

```go
func TestUserWorkflow(t *testing.T) {
    // Step 1: Create user (must succeed)
    user := CreateUser("test@example.com")
    require.NotNil(t, user)

    // Step 2: Activate user (must succeed)
    err := user.Activate()
    require.NoError(t, err)

    // Step 3: Verify state (can fail without stopping)
    assert.True(t, user.IsActive())
    assert.NotEmpty(t, user.Token)
}
```

### Pattern 4: Table-Driven Tests with Require

```go
func TestCalculations(t *testing.T) {
    tests := []struct {
        name     string
        input    int
        expected int
    }{
        {"positive", 5, 10},
        {"zero", 0, 0},
        {"negative", -5, -10},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Calculate(tt.input)
            require.NoError(t, err)           // Each subtest fails independently
            require.Equal(t, tt.expected, result)
        })
    }
}
```

### Pattern 5: Combining Assert and Require

```go
func TestComplexOperation(t *testing.T) {
    require := require.New(t)
    assert := assert.New(t)

    // Critical preconditions
    config, err := LoadConfig()
    require.NoError(err)
    require.NotNil(config.Database)

    // Non-critical checks
    assert.NotEmpty(config.AppName)
    assert.Greater(config.Port, 0)
}
```

## Best Practices

1. **Use for Preconditions:** Use `require` when subsequent code depends on the assertion
2. **Prevent Nil Panics:** Check for nil before dereferencing:
   ```go
   require.NotNil(t, obj)
   assert.Equal(t, "value", obj.Field)  // Safe
   ```
3. **Resource Validation:** Validate critical resources (DB, files, network) with require
4. **Goroutine Safety:** Never use require in goroutines other than the test goroutine
5. **Clear Failure Messages:** Provide descriptive messages for faster debugging
6. **Combine with Assert:** Use require for preconditions, assert for verification

## Common Patterns

### Safe Pointer Dereferencing

```go
user, err := GetUser(123)
require.NoError(t, err)
require.NotNil(t, user)        // Prevents panic on next line
assert.Equal(t, "John", user.Name)
```

### Array/Slice Indexing

```go
items := GetItems()
require.NotEmpty(t, items)     // Prevents panic on next line
assert.Equal(t, "first", items[0].Name)
```

### Map Access

```go
data := GetData()
require.Contains(t, data, "key")  // Ensures key exists
assert.Equal(t, "value", data["key"])
```

### Method Chaining

```go
builder, err := NewBuilder()
require.NoError(t, err)
require.NotNil(t, builder)

result := builder.WithOption1().WithOption2().Build()
assert.NotNil(t, result)
```

## Performance Consideration

`require` stops test execution immediately, which can save time by not running unnecessary assertions when a critical condition fails. However, it provides less information than running all assertions and collecting all failures.

**Strategy:** Use require for critical paths, assert for comprehensive validation.

## Package Information

- **Version:** v1.11.1
- **License:** MIT
- **Repository:** https://github.com/stretchr/testify
- **Documentation:** https://pkg.go.dev/github.com/stretchr/testify/require
- **Imported by:** 17,733+ packages
