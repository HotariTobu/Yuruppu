# Testify - Toolkit for Testifying That Code Behaves Correctly

> Testify is a comprehensive Go testing toolkit providing assertions, mocking, and test suite functionality. It offers expressive assertion methods, mock object creation with expectation verification, and structured test organization with setup/teardown support. Current stable version: v1.11.1, MIT licensed.

Testify is the most widely adopted testing toolkit in the Go ecosystem, with 25.5k+ stars and 17,000+ dependent packages. It extends Go's standard testing framework with readable assertions, powerful mocking capabilities, and object-oriented test organization.

**Installation:** `go get github.com/stretchr/testify`
**Update:** `go get -u github.com/stretchr/testify`
**Go Version Support:** Go 1.19 and later
**Repository:** https://github.com/stretchr/testify
**Package Documentation:** https://pkg.go.dev/github.com/stretchr/testify

## Core Packages

- [Assert Package](assert.md): Provides non-fatal assertion methods that return boolean success indicators
- [Require Package](require.md): Fatal assertions that terminate tests immediately on failure
- [Mock Package](mock.md): Comprehensive mocking framework for creating test doubles with expectation verification
- [Suite Package](suite.md): Object-oriented test organization with lifecycle hooks and setup/teardown methods

## Quick Start

### Basic Assertions

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
    assert.Equal(t, 123, 123, "values should be equal")
    assert.NotNil(t, object, "object should not be nil")
    assert.NoError(t, err, "should not return error")
}
```

### Fatal Assertions

```go
import "github.com/stretchr/testify/require"

func TestWithRequire(t *testing.T) {
    require.NotNil(t, object) // Test stops here if object is nil
    require.NoError(t, err)    // Test stops here if err is not nil

    // This code only executes if above assertions pass
    assert.Equal(t, "expected", object.Value)
}
```

### Mocking

```go
import "github.com/stretchr/testify/mock"

type MockService struct {
    mock.Mock
}

func (m *MockService) DoWork(id int) (string, error) {
    args := m.Called(id)
    return args.String(0), args.Error(1)
}

func TestWithMock(t *testing.T) {
    mockService := new(MockService)
    mockService.On("DoWork", 123).Return("result", nil).Once()

    result, err := mockService.DoWork(123)

    assert.NoError(t, err)
    assert.Equal(t, "result", result)
    mockService.AssertExpectations(t)
}
```

### Test Suites

```go
import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type AppTestSuite struct {
    suite.Suite
    db Database
}

func (s *AppTestSuite) SetupTest() {
    s.db = OpenTestDatabase()
}

func (s *AppTestSuite) TearDownTest() {
    s.db.Close()
}

func (s *AppTestSuite) TestUserCreation() {
    user := s.db.CreateUser("test@example.com")
    s.NotNil(user)
    s.Equal("test@example.com", user.Email)
}

func TestAppTestSuite(t *testing.T) {
    suite.Run(t, new(AppTestSuite))
}
```

## Common Usage Patterns

### Pattern 1: Using Assertions Object

When performing multiple assertions in a test, create an Assertions object to avoid repeating `t`:

```go
func TestMultipleAssertions(t *testing.T) {
    assert := assert.New(t)

    assert.Equal(expected, actual)
    assert.NotNil(object)
    assert.True(condition)
}
```

### Pattern 2: Conditional Assertions

Use assertion return values to perform conditional checks:

```go
if assert.NotNil(t, object) {
    assert.Equal(t, "expected", object.Value)
}
```

### Pattern 3: Eventually - Async Testing

Test conditions that become true over time:

```go
assert.Eventually(t, func() bool {
    return resourceIsReady()
}, 5*time.Second, 100*time.Millisecond, "resource should become ready")
```

### Pattern 4: Mock with Custom Matchers

Use custom matching functions for flexible mock expectations:

```go
mockObj.On("Process", mock.MatchedBy(func(req *Request) bool {
    return req.Valid() && req.UserID > 0
})).Return(nil)
```

### Pattern 5: Subtests with Suites

Combine suite functionality with Go's subtest feature:

```go
func (s *MyTestSuite) TestFeature() {
    s.Run("case1", func() {
        s.Equal(expected1, actual1)
    })

    s.Run("case2", func() {
        s.Equal(expected2, actual2)
    })
}
```

## Key Design Decisions

### Assert vs Require

- **assert**: Returns `bool`, allows test to continue, collects all failures
- **require**: Calls `t.FailNow()`, stops test immediately, prevents cascading failures

**Rule of thumb:** Use `require` for preconditions that must be true before continuing, use `assert` for independent checks.

### Goroutine Safety

**Important:** `require` functions must be called from the goroutine running the test function. For concurrent testing, use `assert` or pass `*assert.CollectT` to goroutines.

### Suite Limitations

The suite package does NOT support parallel test execution. Tests within a suite run sequentially. Use standard Go table-driven tests with `t.Parallel()` if parallelism is required.

## Additional Tools

- **testifylint:** A golangci-lint compatible linter that detects common testify mistakes and anti-patterns
- **mockery:** Code generator that automatically creates mock implementations from Go interfaces

## Optional

- [Error Assertions](assert.md#error-assertions): Comprehensive error checking including ErrorIs and ErrorAs
- [JSON/YAML Assertions](assert.md#json-yaml-assertions): Structured data comparison
- [HTTP Assertions](assert.md#http-assertions): Testing HTTP handlers without servers
- [Mock Argument Matchers](mock.md#argument-matchers): Advanced matching strategies for mock expectations
- [Suite Lifecycle Hooks](suite.md#lifecycle-interfaces): Setup/teardown at multiple levels
