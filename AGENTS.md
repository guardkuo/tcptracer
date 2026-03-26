# TCPTracer Agent Guidelines

## Build Commands

### Building the Application
- `make build` - Compiles the application for the current platform
- `make build_win` - Compiles the application specifically for Windows (amd64)
- `make clean` - Removes compiled objects and executables

### Formatting
- `make format-code` - Formats all Go source files using `go fmt`

### Manual Build Commands
- `go build -o bin/tcptracer.exe tcplog.go log_parse.go` - Standard build
- `GOOS=windows GOARCH=amd64 go build -o bin/tcptracer.exe tcplog.go log_parse.go` - Windows build

## Testing Guidelines

### Running Tests
Currently, there are no test files in this repository. When adding tests:
- Create test files with `_test.go` suffix
- Use the standard Go testing package
- Run tests with `go test ./...`
- Run a specific test with `go test -run TestFunctionName`

### Test Organization
- Table-driven tests are preferred for functions with multiple test cases
- Test files should be in the same package as the code being tested
- Use descriptive test function names that indicate what is being tested
- Include both positive and negative test cases

## Code Style Guidelines

### Imports
- Group imports: standard library first, then third-party, then local
- Use blank lines between import groups
- Align imports vertically when possible
- Avoid relative imports in favor of absolute imports
- Only import packages that are actually used

Example:
```go
import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)
```

### Formatting
- Follow Go's standard formatting (use `go fmt`)
- Line length should not exceed 120 characters
- Use tabs for indentation (Go standard)
- Opening braces go on the same line as the declaration
- Closing braces go on their own line
- Blank lines between logical sections of code
- No trailing whitespace

### Types and Variables
- Use mixedCaps for variable and function names
- Use PascalCase for exported names, camelCase for unexported
- Use meaningful names; avoid single-letter names except for loop counters
- Prefer structured types over parallel arrays
- Use interfaces to define behavior contracts
- Define custom error types when appropriate

### Constants
- Use ALL_CAPS with underscores for constant names
- Group related constants together
- Use `const` for truly constant values
- Use `var` for package-level variables that might change

### Functions
- Keep functions focused and small (generally under 50 lines)
- Return early to reduce nesting
- Handle errors explicitly; don't ignore them
- Use named return values only when they improve clarity
- Document exported functions with comments
- Place receiver methods near the type definition

### Error Handling
- Check errors immediately after function calls
- Return errors up the call stack when appropriate
- Use sentinel errors (like `ErrNotFound`) for common error conditions
- Wrap errors with context when propagating up
- Use `errors.New()` for simple error messages
- Use `fmt.Errorf()` for formatted error messages
- Don't suppress errors with blank identifiers unless intentionally ignoring

### Comments
- Write clear, concise comments explaining why, not what
- Use sentence case and end with a period
- Comment exported functions, types, and constants
- Avoid obvious comments that just restate the code
- Use TODO comments for future work with optional issue references

### Specific Conventions in This Codebase
- Use `uint32` for time and sequence numbers
- Use `int8` for flags and small integer values
- Use descriptive struct names like `TCPPacket` and `TCPConn`
- Use getter/setter patterns for struct manipulation
- Use pointer receivers for methods that modify structs
- Use value receivers for methods that only read structs
- Group related functionality in the same file
- Use early returns for error conditions
- Use bitwise operations for flag manipulation (seen in log parsing)

### Logging and Output
- Use `fmt.Fprintf()` for configurable output destinations
- Avoid direct `fmt.Print*` calls in library code
- Provide io.Writer parameters for flexible output
- Format output consistently for machine readability
- Include units in output when relevant (bytes, microseconds, etc.)

### Performance Considerations
- Pre-slice capacity when known (use `make([]Type, 0, expectedSize)`)
- Avoid unnecessary allocations in hot paths
- Use `strings.Builder` for string concatenation in loops
- Consider pooling for frequently allocated objects
- Profile before optimizing

## Directory Structure
- Root directory contains Go source files
- `bin/` directory contains compiled executables
- No separate `cmd/` or `pkg/` directories needed for this small project

## Version Control
- Write clear, descriptive commit messages
- Commit related changes together
- Don't commit generated files or binaries
- Use .gitignore to exclude build artifacts
- Keep commits small and focused

## Documentation
- Comment all exported functions and types
- Provide examples in comments for complex functions
- Document any non-obvious behavior or edge cases
- Keep documentation close to the code it describes