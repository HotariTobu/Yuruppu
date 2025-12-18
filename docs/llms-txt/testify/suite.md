# Suite Package

> The suite package provides an object-oriented framework for organizing tests with setup/teardown lifecycle hooks. Tests are defined as methods on a struct that embeds suite.Suite, enabling structured test organization similar to traditional xUnit-style frameworks.

**Import:** `github.com/stretchr/testify/suite`

## Core Concept

The suite package allows you to:
- Group related tests together in a struct
- Share common setup/teardown logic
- Access assertion methods directly on the suite
- Organize tests with lifecycle hooks at multiple levels

## Basic Suite Structure

```go
import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
    suite.Suite
    // Add shared state/resources here
}

// Test methods (must start with "Test")
func (s *MyTestSuite) TestSomething() {
    s.Equal(expected, actual)
}

// Entry point for 'go test'
func TestMyTestSuite(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}
```

## Suite Type

```go
type Suite struct {
    *assert.Assertions
    // contains filtered or unexported fields
}
```

### Core Methods

```go
s.T()                    // Get current *testing.T
s.SetT(t *testing.T)     // Set testing.T (usually not needed)
s.Assert()               // Get *assert.Assertions
s.Require()              // Get *require.Assertions
s.Run(name, func())      // Run subtest
```

### Built-in Assertions

Since Suite embeds `*assert.Assertions`, you can use all assert methods directly:

```go
func (s *MyTestSuite) TestExample() {
    s.Equal(123, 123)
    s.NotNil(object)
    s.NoError(err)
    s.True(condition)
}
```

To use require-style assertions:

```go
func (s *MyTestSuite) TestExample() {
    s.Require().NotNil(object)  // Stops test if object is nil
    s.Equal("value", object.Field)
}
```

## Lifecycle Hooks

### Suite-Level Hooks

Run once for the entire suite:

#### SetupAllSuite Interface

```go
type SetupAllSuite interface {
    SetupSuite()
}
```

Runs once before any tests:

```go
func (s *MyTestSuite) SetupSuite() {
    // One-time setup for entire suite
    s.db = OpenDatabaseConnection()
}
```

#### TearDownAllSuite Interface

```go
type TearDownAllSuite interface {
    TearDownSuite()
}
```

Runs once after all tests:

```go
func (s *MyTestSuite) TearDownSuite() {
    // Cleanup after all tests
    s.db.Close()
}
```

### Test-Level Hooks

Run before/after each test method:

#### SetupTestSuite Interface

```go
type SetupTestSuite interface {
    SetupTest()
}
```

Runs before each test:

```go
func (s *MyTestSuite) SetupTest() {
    // Reset state before each test
    s.counter = 0
    s.cache.Clear()
}
```

#### TearDownTestSuite Interface

```go
type TearDownTestSuite interface {
    TearDownTest()
}
```

Runs after each test:

```go
func (s *MyTestSuite) TearDownTest() {
    // Cleanup after each test
    s.tempFiles.RemoveAll()
}
```

### Subtest-Level Hooks (v1.8.2+)

Run before/after each subtest:

#### SetupSubTest Interface

```go
type SetupSubTest interface {
    SetupSubTest()
}
```

Runs before each subtest:

```go
func (s *MyTestSuite) SetupSubTest() {
    // Setup for each subtest
    s.subtestData = LoadData()
}
```

#### TearDownSubTest Interface

```go
type TearDownSubTest interface {
    TearDownSubTest()
}
```

Runs after each subtest:

```go
func (s *MyTestSuite) TearDownSubTest() {
    // Cleanup after each subtest
    s.subtestData.Cleanup()
}
```

### Test-Specific Hooks

#### BeforeTest Interface

```go
type BeforeTest interface {
    BeforeTest(suiteName, testName string)
}
```

Executes right before each test starts:

```go
func (s *MyTestSuite) BeforeTest(suiteName, testName string) {
    s.T().Logf("Starting test: %s.%s", suiteName, testName)
}
```

#### AfterTest Interface

```go
type AfterTest interface {
    AfterTest(suiteName, testName string)
}
```

Executes right after each test finishes:

```go
func (s *MyTestSuite) AfterTest(suiteName, testName string) {
    s.T().Logf("Finished test: %s.%s", suiteName, testName)
}
```

### Suite Statistics

#### WithStats Interface

```go
type WithStats interface {
    HandleStats(suiteName string, stats *SuiteInformation)
}
```

Called when suite finishes with execution statistics:

```go
func (s *MyTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
    s.T().Logf("Suite %s: Passed=%t, Tests=%d",
        suiteName, stats.Passed(), len(stats.TestStats))
}
```

## Execution Order

For a suite with all hooks implemented:

```
1. SetupSuite()               # Once for entire suite
2.   BeforeTest()             # Before each test
3.     SetupTest()            # Before each test
4.       TestMethod1()        # First test
5.     TearDownTest()         # After each test
6.   AfterTest()              # After each test
7.   BeforeTest()             # Before next test
8.     SetupTest()
9.       TestMethod2()        # Second test
10.    TearDownTest()
11.  AfterTest()
12. TearDownSuite()           # Once after all tests
13. HandleStats()             # Suite completion statistics
```

With subtests:

```
1. SetupTest()
2.   Run("subtest1")
3.     SetupSubTest()
4.       subtest logic
5.     TearDownSubTest()
6.   Run("subtest2")
7.     SetupSubTest()
8.       subtest logic
9.     TearDownSubTest()
10. TearDownTest()
```

## Basic Examples

### Simple Suite

```go
type CalculatorTestSuite struct {
    suite.Suite
    calculator *Calculator
}

func (s *CalculatorTestSuite) SetupTest() {
    s.calculator = NewCalculator()
}

func (s *CalculatorTestSuite) TestAdd() {
    result := s.calculator.Add(2, 3)
    s.Equal(5, result)
}

func (s *CalculatorTestSuite) TestSubtract() {
    result := s.calculator.Subtract(5, 3)
    s.Equal(2, result)
}

func TestCalculatorTestSuite(t *testing.T) {
    suite.Run(t, new(CalculatorTestSuite))
}
```

### Suite with Database

```go
type DatabaseTestSuite struct {
    suite.Suite
    db *sql.DB
    tx *sql.Tx
}

func (s *DatabaseTestSuite) SetupSuite() {
    var err error
    s.db, err = sql.Open("postgres", "connection_string")
    s.Require().NoError(err)
}

func (s *DatabaseTestSuite) TearDownSuite() {
    s.db.Close()
}

func (s *DatabaseTestSuite) SetupTest() {
    var err error
    s.tx, err = s.db.Begin()
    s.Require().NoError(err)
}

func (s *DatabaseTestSuite) TearDownTest() {
    s.tx.Rollback()  // Rollback each test
}

func (s *DatabaseTestSuite) TestInsertUser() {
    _, err := s.tx.Exec("INSERT INTO users (name) VALUES ($1)", "John")
    s.NoError(err)
}

func TestDatabaseTestSuite(t *testing.T) {
    suite.Run(t, new(DatabaseTestSuite))
}
```

### Suite with Subtests

```go
type UserTestSuite struct {
    suite.Suite
    service *UserService
}

func (s *UserTestSuite) SetupTest() {
    s.service = NewUserService()
}

func (s *UserTestSuite) TestUserCreation() {
    s.Run("valid email", func() {
        user, err := s.service.CreateUser("test@example.com")
        s.NoError(err)
        s.NotNil(user)
        s.Equal("test@example.com", user.Email)
    })

    s.Run("invalid email", func() {
        user, err := s.service.CreateUser("invalid")
        s.Error(err)
        s.Nil(user)
    })
}

func TestUserTestSuite(t *testing.T) {
    suite.Run(t, new(UserTestSuite))
}
```

## Common Patterns

### Pattern 1: Shared Resources

```go
type APITestSuite struct {
    suite.Suite
    server     *httptest.Server
    client     *http.Client
    apiKey     string
}

func (s *APITestSuite) SetupSuite() {
    s.server = httptest.NewServer(CreateHandler())
    s.client = &http.Client{Timeout: 5 * time.Second}
    s.apiKey = "test-api-key"
}

func (s *APITestSuite) TearDownSuite() {
    s.server.Close()
}

func (s *APITestSuite) TestGetEndpoint() {
    req, _ := http.NewRequest("GET", s.server.URL+"/api/data", nil)
    req.Header.Set("X-API-Key", s.apiKey)

    resp, err := s.client.Do(req)
    s.NoError(err)
    s.Equal(200, resp.StatusCode)
}

func TestAPITestSuite(t *testing.T) {
    suite.Run(t, new(APITestSuite))
}
```

### Pattern 2: Test Isolation with Transactions

```go
type RepositoryTestSuite struct {
    suite.Suite
    db   *sql.DB
    repo *UserRepository
}

func (s *RepositoryTestSuite) SetupSuite() {
    s.db = OpenTestDatabase()
}

func (s *RepositoryTestSuite) TearDownSuite() {
    s.db.Close()
}

func (s *RepositoryTestSuite) SetupTest() {
    tx, _ := s.db.Begin()
    s.repo = NewUserRepository(tx)
}

func (s *RepositoryTestSuite) TearDownTest() {
    // Each test transaction rolls back
    s.repo.tx.Rollback()
}

func (s *RepositoryTestSuite) TestCreateUser() {
    user := s.repo.Create("test@example.com")
    s.NotNil(user)
}

func TestRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(RepositoryTestSuite))
}
```

### Pattern 3: Temporary Files/Directories

```go
type FileProcessorTestSuite struct {
    suite.Suite
    tempDir   string
    processor *FileProcessor
}

func (s *FileProcessorTestSuite) SetupTest() {
    var err error
    s.tempDir, err = os.MkdirTemp("", "test-*")
    s.Require().NoError(err)
    s.processor = NewFileProcessor(s.tempDir)
}

func (s *FileProcessorTestSuite) TearDownTest() {
    os.RemoveAll(s.tempDir)
}

func (s *FileProcessorTestSuite) TestProcessFile() {
    testFile := filepath.Join(s.tempDir, "test.txt")
    os.WriteFile(testFile, []byte("content"), 0644)

    err := s.processor.Process(testFile)
    s.NoError(err)
}

func TestFileProcessorTestSuite(t *testing.T) {
    suite.Run(t, new(FileProcessorTestSuite))
}
```

### Pattern 4: Mocked Dependencies

```go
type ServiceTestSuite struct {
    suite.Suite
    mockDB    *MockDatabase
    mockCache *MockCache
    service   *UserService
}

func (s *ServiceTestSuite) SetupTest() {
    s.mockDB = new(MockDatabase)
    s.mockCache = new(MockCache)
    s.service = NewUserService(s.mockDB, s.mockCache)
}

func (s *ServiceTestSuite) TearDownTest() {
    s.mockDB.AssertExpectations(s.T())
    s.mockCache.AssertExpectations(s.T())
}

func (s *ServiceTestSuite) TestGetUser() {
    expectedUser := &User{ID: 1, Name: "John"}
    s.mockCache.On("Get", "user:1").Return(nil)
    s.mockDB.On("FindUser", 1).Return(expectedUser, nil)
    s.mockCache.On("Set", "user:1", expectedUser).Return(nil)

    user, err := s.service.GetUser(1)
    s.NoError(err)
    s.Equal("John", user.Name)
}

func TestServiceTestSuite(t *testing.T) {
    suite.Run(t, new(ServiceTestSuite))
}
```

### Pattern 5: Logging and Debugging

```go
type DebugTestSuite struct {
    suite.Suite
}

func (s *DebugTestSuite) BeforeTest(suiteName, testName string) {
    s.T().Logf("=== Starting: %s.%s ===", suiteName, testName)
}

func (s *DebugTestSuite) AfterTest(suiteName, testName string) {
    s.T().Logf("=== Finished: %s.%s ===", suiteName, testName)
}

func (s *DebugTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
    s.T().Logf("Suite %s completed: Passed=%t", suiteName, stats.Passed())
}

func (s *DebugTestSuite) TestOperation() {
    s.T().Log("Performing operation...")
    s.True(true)
}

func TestDebugTestSuite(t *testing.T) {
    suite.Run(t, new(DebugTestSuite))
}
```

### Pattern 6: Configuration-Based Tests

```go
type ConfigurableTestSuite struct {
    suite.Suite
    config *TestConfig
}

func (s *ConfigurableTestSuite) SetupSuite() {
    s.config = LoadTestConfig()
    s.Require().NotNil(s.config)
}

func (s *ConfigurableTestSuite) TestWithConfig() {
    if s.config.EnableFeatureX {
        s.Run("feature X enabled", func() {
            s.True(s.config.EnableFeatureX)
        })
    } else {
        s.T().Skip("Feature X disabled in config")
    }
}

func TestConfigurableTestSuite(t *testing.T) {
    suite.Run(t, new(ConfigurableTestSuite))
}
```

### Pattern 7: Table-Driven Tests in Suites

```go
type ValidationTestSuite struct {
    suite.Suite
    validator *EmailValidator
}

func (s *ValidationTestSuite) SetupSuite() {
    s.validator = NewEmailValidator()
}

func (s *ValidationTestSuite) TestEmailValidation() {
    tests := []struct {
        name  string
        email string
        valid bool
    }{
        {"valid email", "test@example.com", true},
        {"missing @", "testexample.com", false},
        {"missing domain", "test@", false},
    }

    for _, tt := range tests {
        s.Run(tt.name, func() {
            result := s.validator.Validate(tt.email)
            s.Equal(tt.valid, result)
        })
    }
}

func TestValidationTestSuite(t *testing.T) {
    suite.Run(t, new(ValidationTestSuite))
}
```

## Advanced Usage

### Nested Suites

```go
type BaseSuite struct {
    suite.Suite
    db *sql.DB
}

func (s *BaseSuite) SetupSuite() {
    s.db = OpenDatabase()
}

type UserTestSuite struct {
    BaseSuite  // Inherit from BaseSuite
    userRepo *UserRepository
}

func (s *UserTestSuite) SetupTest() {
    s.userRepo = NewUserRepository(s.db)
}

func TestUserTestSuite(t *testing.T) {
    suite.Run(t, new(UserTestSuite))
}
```

### Conditional Test Execution

```go
func (s *MyTestSuite) TestRequiresDatabase() {
    if testing.Short() {
        s.T().Skip("Skipping database test in short mode")
    }

    // Test that requires database
}
```

### Custom Assertions

```go
func (s *MyTestSuite) AssertValidUser(user *User) {
    s.NotNil(user)
    s.NotEmpty(user.Email)
    s.Greater(user.ID, 0)
}

func (s *MyTestSuite) TestUser() {
    user := CreateUser()
    s.AssertValidUser(user)
}
```

## Best Practices

1. **One Suite Per Feature:** Group related tests in a single suite
2. **Isolate Test State:** Use SetupTest/TearDownTest to reset state between tests
3. **Share Expensive Resources:** Use SetupSuite for expensive one-time setup
4. **Use Require for Setup:** Use `s.Require()` in setup methods to fail fast
5. **Clean Up Resources:** Always implement TearDown methods for resources
6. **Name Tests Clearly:** Test method names should describe what they test
7. **Use Subtests:** Organize related assertions with `s.Run()`
8. **Avoid Test Dependencies:** Each test should be independent

## Important Limitations

### No Parallel Execution

The suite package does NOT support parallel test execution:

```go
// This will NOT work in suites
func (s *MyTestSuite) TestParallel() {
    s.T().Parallel()  // Has no effect
}
```

**Workaround:** Use standard Go table-driven tests with `t.Parallel()` if parallelism is required.

### Test Method Naming

Only methods starting with "Test" are executed as tests:

```go
func (s *MyTestSuite) TestThis()     // Executed
func (s *MyTestSuite) TestThat()     // Executed
func (s *MyTestSuite) HelperMethod() // Not executed (used as helper)
```

## Package Information

- **Version:** v1.11.1
- **License:** MIT
- **Repository:** https://github.com/stretchr/testify
- **Documentation:** https://pkg.go.dev/github.com/stretchr/testify/suite
- **Limitation:** No parallel test support (issue #934)
