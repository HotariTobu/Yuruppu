# Mock Package

> The mock package provides a comprehensive framework for creating test doubles with expectation verification. It allows you to specify method call expectations, return values, call counts, and custom behaviors for mocked dependencies.

**Import:** `github.com/stretchr/testify/mock`

## Core Concept

The mock package provides the `Mock` type that tracks method calls and verifies expectations:

```go
type Mock struct {
    ExpectedCalls []*Call  // Expected method calls
    Calls         []Call   // Actual method calls made
}
```

## Basic Mock Creation

### Step 1: Define Mock Struct

Embed `mock.Mock` in your struct:

```go
import "github.com/stretchr/testify/mock"

type MockDatabase struct {
    mock.Mock
}
```

### Step 2: Implement Methods

Each method should call `Called()` and use the result:

```go
func (m *MockDatabase) GetUser(id int) (*User, error) {
    args := m.Called(id)
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockDatabase) SaveUser(user *User) error {
    args := m.Called(user)
    return args.Error(0)
}

func (m *MockDatabase) Count() int {
    args := m.Called()
    return args.Int(0)
}
```

### Step 3: Setup Expectations

```go
func TestUserService(t *testing.T) {
    mockDB := new(MockDatabase)

    // Setup expectations
    expectedUser := &User{ID: 1, Name: "John"}
    mockDB.On("GetUser", 1).Return(expectedUser, nil)

    // Use the mock
    service := NewUserService(mockDB)
    user, err := service.GetUserByID(1)

    // Verify expectations
    assert.NoError(t, err)
    assert.Equal(t, "John", user.Name)
    mockDB.AssertExpectations(t)
}
```

## Core Methods

### On - Define Expectation

```go
mock.On(methodName string, arguments ...interface{}) *Call
```

Begins defining an expectation for a method:

```go
mockObj.On("MethodName", arg1, arg2, arg3)
```

### Called - Record Call

```go
mock.Called(arguments ...interface{}) Arguments
```

Records that a method was called and returns configured return values:

```go
func (m *MockService) DoWork(id int) (string, error) {
    args := m.Called(id)
    return args.String(0), args.Error(1)
}
```

### Return - Specify Return Values

```go
call.Return(returnValues ...interface{}) *Call
```

Specifies what values the mocked method should return:

```go
mockObj.On("GetValue").Return(42, nil)
mockObj.On("GetData").Return([]byte{1, 2, 3}, nil)
```

### AssertExpectations - Verify Calls

```go
mock.AssertExpectations(t TestingT) bool
```

Verifies all expected calls were made:

```go
mockObj.AssertExpectations(t)
```

### AssertNumberOfCalls

```go
mock.AssertNumberOfCalls(t TestingT, methodName string, expectedCalls int) bool
```

Verify specific method was called exact number of times:

```go
mockObj.AssertNumberOfCalls(t, "GetUser", 3)
```

### AssertCalled / AssertNotCalled

```go
mock.AssertCalled(t TestingT, methodName string, arguments ...interface{}) bool
mock.AssertNotCalled(t TestingT, methodName string, arguments ...interface{}) bool
```

Verify method was/wasn't called with specific arguments:

```go
mockObj.AssertCalled(t, "SaveUser", user)
mockObj.AssertNotCalled(t, "DeleteUser", mock.Anything)
```

## Argument Matchers

### Anything - Match Any Argument

```go
const Anything = "mock.Anything"
```

Matches any value for that argument:

```go
mockObj.On("ProcessData", mock.Anything).Return(nil)
mockObj.On("Store", "key", mock.Anything, mock.Anything).Return(nil)
```

### AnythingOfType - Match by Type

```go
mock.AnythingOfType(typeName string)
```

Matches any argument of the specified type:

```go
mockObj.On("Handle", mock.AnythingOfType("string")).Return(nil)
mockObj.On("Process", mock.AnythingOfType("*http.Request")).Return(nil)
mockObj.On("Save", mock.AnythingOfType("int"), mock.AnythingOfType("*User")).Return(nil)
```

**Type name format:** Use Go type names including package paths for non-builtin types:
- Builtin types: `"string"`, `"int"`, `"bool"`, `"[]byte"`
- Pointers: `"*User"`, `"*http.Request"`
- Qualified types: `"*mypackage.CustomType"`

### IsType - Match by Zero Value

```go
mock.IsType(exampleType interface{})
```

Alternative to `AnythingOfType` using a zero-value example:

```go
mockObj.On("Process", mock.IsType("")).Return(nil)           // string
mockObj.On("Handle", mock.IsType(&http.Request{})).Return(nil)  // *http.Request
```

### MatchedBy - Custom Matcher

```go
mock.MatchedBy(fn interface{})
```

Custom matching logic using a predicate function:

```go
// Match requests to specific host
mockObj.On("HandleRequest", mock.MatchedBy(func(req *http.Request) bool {
    return req.Host == "example.com"
})).Return(nil)

// Match positive numbers
mockObj.On("Calculate", mock.MatchedBy(func(n int) bool {
    return n > 0
})).Return(n * 2, nil)

// Match non-empty strings
mockObj.On("Process", mock.MatchedBy(func(s string) bool {
    return len(s) > 0
})).Return(nil)
```

## Call Modifiers

### Call Count Expectations

```go
mockObj.On("Method").Return(value).Once()              // Called exactly once
mockObj.On("Method").Return(value).Twice()             // Called exactly twice
mockObj.On("Method").Return(value).Times(3)            // Called exactly 3 times
mockObj.On("Method").Return(value).Maybe()             // Optional call (0 or more)
```

### Run - Execute Custom Logic

```go
call.Run(fn func(args Arguments))
```

Execute custom code when the method is called:

```go
var capturedArg string
mockObj.On("Process", mock.Anything).Run(func(args mock.Arguments) {
    capturedArg = args.String(0)
    fmt.Printf("Process called with: %s\n", capturedArg)
}).Return(nil)
```

### Return Functions

```go
call.Return(fn func(args Arguments) []interface{})
```

Dynamic return values based on input:

```go
mockObj.On("Calculate", mock.AnythingOfType("int")).Return(func(args mock.Arguments) (int, error) {
    n := args.Int(0)
    if n < 0 {
        return 0, errors.New("negative input")
    }
    return n * 2, nil
})
```

### Panic - Panic Instead of Return

```go
call.Panic(message string)
```

Make the method panic:

```go
mockObj.On("DangerousOperation").Panic("operation not allowed")
```

### Timing Control

```go
call.After(duration time.Duration)
call.WaitUntil(channel <-chan time.Time)
```

Control when the mock returns:

```go
// Block for duration
mockObj.On("SlowOperation").After(100 * time.Millisecond).Return(nil)

// Wait for channel
readyCh := make(chan time.Time)
mockObj.On("Operation").WaitUntil(readyCh).Return(nil)
```

### NotBefore - Ordering Constraint

```go
call.NotBefore(calls ...*Call)
```

Ensure calls happen in specific order:

```go
call1 := mockObj.On("First").Return(nil)
call2 := mockObj.On("Second").NotBefore(call1).Return(nil)
```

## Arguments Type

```go
type Arguments []interface{}
```

Helper type for extracting return values from `Called()`:

### Type-Safe Getters

```go
args.Int(index int) int
args.Bool(index int) bool
args.String(index int) string
args.Error(index int) error
args.Get(index int) interface{}
```

Usage:

```go
func (m *MockService) GetData(id int) (string, int, error) {
    args := m.Called(id)
    return args.String(0), args.Int(1), args.Error(2)
}
```

### Nil Handling

```go
// Return nil pointer safely
func (m *MockDB) GetUser(id int) (*User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

// Setup returning nil
mockDB.On("GetUser", 999).Return(nil, errors.New("not found"))
```

### Assert and Diff

```go
args.Assert(t TestingT, objects ...interface{}) bool
args.Is(objects ...interface{}) bool
args.Diff(objects ...interface{}) string
```

Compare arguments:

```go
args.Assert(t, expectedArg1, expectedArg2)
if args.Is(expected1, expected2) {
    // arguments match
}
```

## Common Patterns

### Pattern 1: Simple Mock

```go
type MockService struct {
    mock.Mock
}

func (m *MockService) GetValue() (int, error) {
    args := m.Called()
    return args.Int(0), args.Error(1)
}

func TestSimple(t *testing.T) {
    mockSvc := new(MockService)
    mockSvc.On("GetValue").Return(42, nil)

    value, err := mockSvc.GetValue()
    assert.NoError(t, err)
    assert.Equal(t, 42, value)
    mockSvc.AssertExpectations(t)
}
```

### Pattern 2: Dependency Injection

```go
type UserService struct {
    db Database
}

func (s *UserService) CreateUser(email string) (*User, error) {
    user := &User{Email: email}
    err := s.db.SaveUser(user)
    return user, err
}

func TestUserService(t *testing.T) {
    mockDB := new(MockDatabase)
    mockDB.On("SaveUser", mock.AnythingOfType("*User")).Return(nil)

    service := &UserService{db: mockDB}
    user, err := service.CreateUser("test@example.com")

    assert.NoError(t, err)
    assert.Equal(t, "test@example.com", user.Email)
    mockDB.AssertExpectations(t)
}
```

### Pattern 3: Multiple Return Values

```go
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) FindUsers(query string) ([]*User, int, error) {
    args := m.Called(query)
    return args.Get(0).([]*User), args.Int(1), args.Error(2)
}

func TestFindUsers(t *testing.T) {
    mockRepo := new(MockRepository)
    users := []*User{{ID: 1}, {ID: 2}}
    mockRepo.On("FindUsers", "active").Return(users, 2, nil)

    results, count, err := mockRepo.FindUsers("active")
    assert.NoError(t, err)
    assert.Equal(t, 2, count)
    assert.Len(t, results, 2)
    mockRepo.AssertExpectations(t)
}
```

### Pattern 4: Different Behaviors for Different Arguments

```go
func TestDifferentBehaviors(t *testing.T) {
    mockDB := new(MockDatabase)

    // Success case
    mockDB.On("GetUser", 1).Return(&User{ID: 1, Name: "John"}, nil)

    // Not found case
    mockDB.On("GetUser", 999).Return(nil, errors.New("not found"))

    // Any other ID
    mockDB.On("GetUser", mock.Anything).Return(&User{ID: 0}, nil)

    user1, _ := mockDB.GetUser(1)
    assert.Equal(t, "John", user1.Name)

    user2, err := mockDB.GetUser(999)
    assert.Error(t, err)
    assert.Nil(t, user2)

    mockDB.AssertExpectations(t)
}
```

### Pattern 5: Capturing Arguments

```go
func TestCaptureArguments(t *testing.T) {
    mockDB := new(MockDatabase)
    var savedUser *User

    mockDB.On("SaveUser", mock.AnythingOfType("*User")).Run(func(args mock.Arguments) {
        savedUser = args.Get(0).(*User)
    }).Return(nil)

    service := &UserService{db: mockDB}
    service.CreateUser("test@example.com")

    assert.NotNil(t, savedUser)
    assert.Equal(t, "test@example.com", savedUser.Email)
    mockDB.AssertExpectations(t)
}
```

### Pattern 6: Dynamic Return Values

```go
func TestDynamicReturns(t *testing.T) {
    mockCalc := new(MockCalculator)

    mockCalc.On("Square", mock.AnythingOfType("int")).Return(
        func(args mock.Arguments) int {
            n := args.Int(0)
            return n * n
        },
    )

    assert.Equal(t, 4, mockCalc.Square(2))
    assert.Equal(t, 9, mockCalc.Square(3))
    assert.Equal(t, 16, mockCalc.Square(4))
    mockCalc.AssertExpectations(t)
}
```

### Pattern 7: Complex Matching

```go
func TestComplexMatching(t *testing.T) {
    mockAPI := new(MockAPI)

    // Match only valid requests
    mockAPI.On("SendRequest", mock.MatchedBy(func(req *Request) bool {
        return req != nil && req.Valid() && req.UserID > 0
    })).Return(&Response{Status: 200}, nil)

    // Match invalid requests
    mockAPI.On("SendRequest", mock.MatchedBy(func(req *Request) bool {
        return req == nil || !req.Valid()
    })).Return(nil, errors.New("invalid request"))

    resp, err := mockAPI.SendRequest(&Request{UserID: 123, Valid: true})
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.Status)

    mockAPI.AssertExpectations(t)
}
```

### Pattern 8: Testing Error Cases

```go
func TestErrorHandling(t *testing.T) {
    mockDB := new(MockDatabase)

    // Simulate database error
    mockDB.On("SaveUser", mock.Anything).Return(errors.New("connection failed"))

    service := &UserService{db: mockDB}
    user, err := service.CreateUser("test@example.com")

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "connection failed")
    assert.Nil(t, user)
    mockDB.AssertExpectations(t)
}
```

### Pattern 9: Call Count Verification

```go
func TestCallCounts(t *testing.T) {
    mockCache := new(MockCache)

    mockCache.On("Get", "key1").Return("value1").Once()
    mockCache.On("Set", "key1", mock.Anything).Return(nil).Once()
    mockCache.On("Get", "key2").Return("value2").Twice()

    // Perform operations
    mockCache.Get("key1")
    mockCache.Set("key1", "newvalue")
    mockCache.Get("key2")
    mockCache.Get("key2")

    mockCache.AssertExpectations(t)
    mockCache.AssertNumberOfCalls(t, "Get", 3)
}
```

### Pattern 10: Optional Calls

```go
func TestOptionalCalls(t *testing.T) {
    mockLogger := new(MockLogger)

    // Maybe() makes the call optional
    mockLogger.On("Debug", mock.Anything).Maybe()

    // Test doesn't fail if Debug is never called
    mockLogger.AssertExpectations(t)
}
```

## Best Practices

1. **Define Clear Interfaces:** Mock interfaces, not concrete types
2. **One Mock Per Test:** Create new mock instances for each test to avoid state leakage
3. **Verify Expectations:** Always call `AssertExpectations(t)` at the end of tests
4. **Use Appropriate Matchers:** Balance between strict and flexible matching
5. **Capture Critical Arguments:** Use `Run()` to verify important argument values
6. **Return Realistic Values:** Make mock return values match real implementation behavior
7. **Test Error Paths:** Mock both success and failure scenarios
8. **Document Complex Mocks:** Add comments explaining complex mock setups

## Common Pitfalls

- **Forgetting AssertExpectations:** Mocks won't verify if you don't call `AssertExpectations(t)`
- **Wrong Type Assertions:** Ensure return types match when using `Get()` with type assertions
- **Nil Return Values:** Use explicit nil checks when mocking methods that return pointers
- **Argument Order:** Arguments in `On()` must match the order in the actual method
- **Multiple Expectations:** Order matters - more specific expectations should come before general ones
- **Goroutine Safety:** Mock is not goroutine-safe by default - synchronize access if needed

## Helper Utilities

### AssertExpectationsForObjects

Verify multiple mocks at once:

```go
mock.AssertExpectationsForObjects(t, mockDB, mockCache, mockLogger)
```

### Test Anything

Use in assertions to skip argument checking:

```go
mockObj.AssertCalled(t, "Method", mock.Anything, 42, mock.Anything)
```

## Package Information

- **Version:** v1.11.1
- **License:** MIT
- **Repository:** https://github.com/stretchr/testify
- **Documentation:** https://pkg.go.dev/github.com/stretchr/testify/mock
- **Related Tool:** mockery - Autogenerate mocks from interfaces
