# Assert Package

> The assert package provides non-fatal assertion functions that return boolean success indicators. All assertions accept an optional message parameter and continue test execution on failure, allowing collection of multiple test failures.

**Import:** `github.com/stretchr/testify/assert`

## Basic Usage

### Direct Function Calls

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSomething(t *testing.T) {
    assert.Equal(t, 123, 123, "values should be equal")
    assert.NotNil(t, object, "object should not be nil")
}
```

### Using Assertions Type (Recommended)

For multiple assertions in a single test, create an Assertions instance:

```go
func TestSomething(t *testing.T) {
    assert := assert.New(t)

    assert.Equal(123, 123, "values should be equal")
    assert.NotNil(object, "object should not be nil")
}
```

## Equality Assertions

### Equal / NotEqual

```go
// Basic equality check using ==
assert.Equal(t, expected, actual)
assert.NotEqual(t, unexpected, actual)
```

### EqualValues / NotEqualValues

Type-convertible equality (e.g., int32 vs uint32):

```go
assert.EqualValues(t, uint32(123), int32(123))
assert.NotEqualValues(t, uint32(123), int32(456))
```

### Exactly

Strict type and value equality (no type conversion):

```go
assert.Exactly(t, int32(123), int32(123))
// Would fail: assert.Exactly(t, uint32(123), int32(123))
```

### EqualExportedValues

Compare only exported (public) fields of structs:

```go
type S struct {
    PublicField  int
    privateField int
}

assert.EqualExportedValues(t, S{1, 2}, S{1, 999}) // Pass - private fields ignored
```

## Boolean Assertions

```go
assert.True(t, condition, "should be true")
assert.False(t, condition, "should be false")
```

## Nil and Empty Assertions

### Nil Checks

```go
assert.Nil(t, object, "should be nil")
assert.NotNil(t, object, "should not be nil")
```

### Empty Checks

Works with strings, slices, maps, channels, and zero values:

```go
assert.Empty(t, "", "empty string")
assert.Empty(t, []int{}, "empty slice")
assert.Empty(t, map[string]int{}, "empty map")
assert.Empty(t, 0, "zero value")

assert.NotEmpty(t, "hello", "non-empty string")
assert.NotEmpty(t, []int{1}, "non-empty slice")
```

### Zero Checks

```go
assert.Zero(t, 0, "should be zero")
assert.NotZero(t, 42, "should not be zero")
```

## Comparison Assertions

### Greater / Less

```go
assert.Greater(t, 10, 5, "10 > 5")
assert.GreaterOrEqual(t, 10, 10, "10 >= 10")
assert.Less(t, 5, 10, "5 < 10")
assert.LessOrEqual(t, 5, 5, "5 <= 5")
```

### Positive / Negative

```go
assert.Positive(t, 42, "should be > 0")
assert.Negative(t, -42, "should be < 0")
```

## Numeric Precision Assertions

### InDelta - Absolute Difference

Check if numbers are within an absolute delta:

```go
// Pass if |expected - actual| <= delta
assert.InDelta(t, 1.0, 1.1, 0.2)        // Pass: |1.0 - 1.1| = 0.1 <= 0.2
assert.InDeltaSlice(t, []float64{1.0, 2.0}, []float64{1.1, 2.1}, 0.2)
assert.InDeltaMapValues(t, map[string]float64{"a": 1.0}, map[string]float64{"a": 1.1}, 0.2)
```

### InEpsilon - Relative Difference

Check if numbers are within a relative epsilon:

```go
// Pass if |expected - actual| / |expected| <= epsilon
assert.InEpsilon(t, 100.0, 101.0, 0.02)  // Pass: 1/100 = 0.01 <= 0.02
assert.InEpsilonSlice(t, []float64{100.0}, []float64{101.0}, 0.02)
```

## String and Container Assertions

### Contains / NotContains

Works with strings, slices, arrays, and maps:

```go
// String contains
assert.Contains(t, "Hello World", "World")
assert.NotContains(t, "Hello World", "xyz")

// Slice contains
assert.Contains(t, []string{"a", "b", "c"}, "b")

// Map contains key
assert.Contains(t, map[string]int{"key": 123}, "key")
```

### Length

```go
assert.Len(t, []int{1, 2, 3}, 3)
assert.Len(t, "hello", 5)
assert.Len(t, map[string]int{"a": 1}, 1)
```

## Error Assertions

### Basic Error Checks

```go
err := SomeFunction()

assert.Error(t, err, "should return error")
assert.NoError(t, err, "should not return error")
```

### Error Message Checks

```go
assert.EqualError(t, err, "expected error message")
assert.ErrorContains(t, err, "partial message")
```

### Advanced Error Checks (Go 1.13+)

```go
var targetErr *CustomError

// Check error chain with errors.Is
assert.ErrorIs(t, err, io.EOF)
assert.NotErrorIs(t, err, context.Canceled)

// Check error type with errors.As
assert.ErrorAs(t, err, &targetErr)
assert.NotErrorAs(t, err, &targetErr)
```

## Panic Assertions

### Basic Panic

```go
assert.Panics(t, func() {
    panic("something went wrong")
}, "should panic")

assert.NotPanics(t, func() {
    // normal code
}, "should not panic")
```

### Panic With Specific Value

```go
assert.PanicsWithValue(t, "expected panic value", func() {
    panic("expected panic value")
})
```

### Panic With Error

```go
assert.PanicsWithError(t, "error message", func() {
    panic(errors.New("error message"))
})
```

## Type Assertions

### IsType / IsNotType

```go
assert.IsType(t, "", "hello")           // Both are strings
assert.IsType(t, (*int)(nil), &value)   // Both are *int
assert.IsNotType(t, "", 123)            // String vs int
```

### Implements / NotImplements

```go
var writer io.Writer
assert.Implements(t, (*io.Writer)(nil), &bytes.Buffer{})
assert.NotImplements(t, (*io.Writer)(nil), "string")
```

## Sequence and Collection Assertions

### ElementsMatch

Elements match regardless of order (allows duplicates):

```go
assert.ElementsMatch(t, []int{1, 3, 2, 3}, []int{1, 3, 3, 2})  // Pass
assert.NotElementsMatch(t, []int{1, 2}, []int{1, 3})
```

### Subset

Check if all elements of subset exist in the list:

```go
assert.Subset(t, []int{1, 2, 3, 4}, []int{1, 2})  // Pass
assert.NotSubset(t, []int{1, 2}, []int{3})
```

### Same / NotSame

Pointer equality (same memory address):

```go
obj := &MyStruct{}
assert.Same(t, obj, obj)           // Same pointer
assert.NotSame(t, &MyStruct{}, &MyStruct{})  // Different pointers
```

## Order Assertions

```go
assert.IsIncreasing(t, []int{1, 2, 3})           // Strictly increasing
assert.IsNonIncreasing(t, []int{3, 2, 1})        // Decreasing or equal
assert.IsDecreasing(t, []int{3, 2, 1})           // Strictly decreasing
assert.IsNonDecreasing(t, []int{1, 2, 2, 3})     // Increasing or equal
```

## JSON and YAML Assertions

### JSONEq

Compare JSON strings (ignores formatting and field order):

```go
expected := `{"name":"John","age":30}`
actual := `{
    "age": 30,
    "name": "John"
}`
assert.JSONEq(t, expected, actual)  // Pass
```

### YAMLEq

Compare YAML strings (ignores formatting):

```go
expected := "name: John\nage: 30"
actual := "age: 30\nname: John"
assert.YAMLEq(t, expected, actual)  // Pass
```

## Regular Expression Assertions

```go
assert.Regexp(t, regexp.MustCompile("^[a-z]+$"), "hello")
assert.Regexp(t, "^[0-9]+$", "12345")  // Can use string pattern

assert.NotRegexp(t, regexp.MustCompile("^[0-9]+$"), "hello")
```

## File and Directory Assertions

```go
assert.FileExists(t, "/path/to/file.txt")
assert.NoFileExists(t, "/path/to/missing.txt")
assert.DirExists(t, "/path/to/directory")
assert.NoDirExists(t, "/path/to/missing")
```

## Time Assertions

### WithinDuration

Check if times are within a duration:

```go
now := time.Now()
later := now.Add(100 * time.Millisecond)
assert.WithinDuration(t, now, later, 200*time.Millisecond)  // Pass
```

### WithinRange

Check if time is within a range:

```go
start := time.Now()
end := start.Add(1 * time.Hour)
actual := start.Add(30 * time.Minute)
assert.WithinRange(t, actual, start, end)  // Pass
```

## Conditional and Async Assertions

### Condition

Custom boolean condition:

```go
assert.Condition(t, func() bool {
    return complexCheck()
}, "custom condition failed")
```

### Eventually

Poll a condition until it becomes true or timeout:

```go
assert.Eventually(t, func() bool {
    return resourceIsReady()
}, 5*time.Second, 100*time.Millisecond, "resource should become ready")
```

### Never

Ensure condition never becomes true within duration:

```go
assert.Never(t, func() bool {
    return shouldNeverBeTrue()
}, 2*time.Second, 100*time.Millisecond, "should never be true")
```

### EventuallyWithT

Eventually with access to testing.T in the condition:

```go
assert.EventuallyWithT(t, func(c *assert.CollectT) {
    result := fetchResult()
    assert.New(c).Equal(expected, result)
}, 5*time.Second, 100*time.Millisecond, "should eventually equal")
```

## HTTP Assertions

Test HTTP handlers without running a server:

### Status Code Checks

```go
handler := http.HandlerFunc(myHandler)

assert.HTTPSuccess(t, handler, "GET", "/path", nil)
assert.HTTPError(t, handler, "GET", "/badpath", nil)
assert.HTTPRedirect(t, handler, "GET", "/oldpath", nil)
assert.HTTPStatusCode(t, handler, "GET", "/path", nil, 200)
```

### Response Body Checks

```go
assert.HTTPBodyContains(t, handler, "GET", "/path", nil, "expected text")
assert.HTTPBodyNotContains(t, handler, "GET", "/path", nil, "unexpected text")
```

## Formatted Variants

Every assertion has a formatted variant ending in `f`:

```go
assert.Equalf(t, expected, actual, "comparing %s", "values")
assert.Truef(t, condition, "expected %s to be true", varName)
assert.NoErrorf(t, err, "operation %s failed", operation)
```

## Helper Types and Functions

### TestingT Interface

```go
type TestingT interface {
    Errorf(format string, args ...interface{})
    FailNow()
}
```

Both `*testing.T` and `*testing.B` implement this interface.

### CollectT

Collects errors before failing (used with `EventuallyWithT`):

```go
collect := new(assert.CollectT)
assert.New(collect).Equal(expected, actual)
// Can accumulate multiple assertions before FailNow
```

### ObjectsAreEqual

Compare objects without assertions:

```go
equal := assert.ObjectsAreEqual(expected, actual)  // Returns bool
```

### Manual Failure

```go
assert.Fail(t, "failure message")       // Log failure, continue test
assert.FailNow(t, "failure message")    // Log failure, stop test immediately
```

## Best Practices

1. **Use Assertions Object:** Create `assert := assert.New(t)` for multiple assertions
2. **Provide Messages:** Always include descriptive failure messages
3. **Choose Appropriate Equality:** Use `Equal` for general cases, `Exactly` for strict type checking
4. **Conditional Assertions:** Use return value to avoid nil pointer panics:
   ```go
   if assert.NotNil(t, obj) {
       assert.Equal(t, "value", obj.Field)
   }
   ```
5. **Async Testing:** Use `Eventually` or `EventuallyWithT` for time-dependent conditions
6. **Error Checking:** Use `ErrorIs` and `ErrorAs` for wrapped errors (Go 1.13+)

## Common Pitfalls

- **ElementsMatch vs Equal:** `ElementsMatch` ignores order, `Equal` requires exact order
- **Empty vs Nil:** `Empty` checks for zero values, `Nil` specifically checks for nil
- **Eventually Timeout:** First parameter is total timeout, second is polling interval
- **InDelta vs InEpsilon:** `InDelta` uses absolute difference, `InEpsilon` uses relative difference
