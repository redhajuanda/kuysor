# Contributing to Kuysor

Thank you for your interest in contributing to Kuysor! This document provides guidelines and information for contributors.

## 📋 Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Reporting Issues](#reporting-issues)
- [Documentation](#documentation)
- [Release Process](#release-process)

## 🚀 Getting Started

### Prerequisites

- Go 1.21.5 or later
- Git
- A GitHub account

### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/kuysor.git
   cd kuysor
   ```

3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/redhajuanda/kuysor.git
   ```

4. **Install dependencies** (verify the module):
   ```bash
   go mod tidy
   go mod verify
   ```

5. **Run tests** to ensure everything is working:
   ```bash
   go test -v ./...
   ```

## 📝 Code Standards

### Go Style Guidelines

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code (run `gofmt -s -w .`)
- Use `go vet` to check for common errors
- Follow Go naming conventions (exported functions start with capital letters)

### Code Quality Tools

Before submitting, ensure your code passes these checks:

```bash
# Format code
gofmt -s -w .

# Check for issues
go vet ./...

# Run tests
go test -v ./...

# Check test coverage
go test -cover ./...
```

### Coding Practices

1. **Error Handling**: Use proper error handling instead of `panic()` calls
2. **Documentation**: Add doc comments for all exported functions, types, and constants
3. **Naming**: Use clear, descriptive names for variables and functions
4. **Simplicity**: Prefer simple, readable code over clever optimizations
5. **Zero Dependencies**: Maintain the zero-dependency policy

### Performance Optimization Guidelines

When optimizing performance, follow these principles:

1. **Measure First**: Always benchmark before optimizing
2. **Hot Path Focus**: Optimize the most frequently called code paths
3. **Memory Allocation**: Minimize allocations in hot paths
4. **String Operations**: Use `strings.Builder` for string concatenation
5. **Avoid Premature Optimization**: Profile to identify real bottlenecks

#### Key Performance Areas
- **Query Building**: Main user-facing operation (~64μs baseline)
- **Cursor Operations**: Critical for pagination (~131μs with cursor)
- **Sort Parsing**: Scales with column count (113ns-688ns)
- **Memory Usage**: Keep allocations reasonable (<500 per operation)

#### Optimization Targets
- Sub-100μs for basic operations
- Linear scaling with complexity
- Minimal memory allocations
- Thread-safe concurrent access

### Example of Good Documentation:

```go
// WithLimit sets the maximum number of records to return per page.
// The limit must be a positive integer. If not set, the default limit
// of 10 will be used. An additional record is fetched internally to
// determine if there are more pages available.
func (p *Kuysor) WithLimit(limit int) *Kuysor {
    // implementation...
}
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test -v ./modifier

# Run specific test
go test -v -run TestCursorFirstPageQuestion

# Run benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmarks
go test -bench=BenchmarkQueryBuild -benchmem ./...

# Generate benchmark comparison
go test -bench=. -count=5 -benchmem ./... > old.txt
# (make changes)
go test -bench=. -count=5 -benchmem ./... > new.txt
# Compare with benchcmp tool
```

### Writing Tests

1. **Test Structure**: Follow the table-driven test pattern used in the project
2. **Test Names**: Use descriptive test names that explain what is being tested
3. **Coverage**: Aim for high test coverage, especially for new functionality
4. **Edge Cases**: Include tests for edge cases and error conditions

### Performance Testing

When making performance-critical changes, always run benchmarks:

#### Current Performance Baselines
- **Basic Query Build**: ~64μs (target: <100μs)
- **Query with Cursor**: ~131μs (target: <200μs)
- **Cursor Parsing**: ~1.6μs (target: <5μs)
- **Sort Parsing**: 113ns-688ns (target: linear scaling)

#### Benchmark Guidelines
1. **Run Before Changes**: Establish baseline performance
2. **Run After Changes**: Measure impact of your modifications
3. **Performance Regression**: No operation should be >20% slower
4. **Memory Regression**: Memory allocations should not increase significantly
5. **Document Changes**: Update README if performance characteristics change

#### Writing Performance Tests
- Use `b.ResetTimer()` before measurement loops
- Include `b.ReportAllocs()` for memory analysis
- Test with realistic data sizes and complexity
- Use `b.RunParallel()` for concurrent operation testing

### Example Test Structure:

```go
func TestNewFeature(t *testing.T) {
    testCases := []struct {
        name      string
        input     string
        expected  string
        expectErr bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "expected result",
        },
        {
            name:      "invalid input",
            input:     "",
            expectErr: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := NewFeature(tc.input)
            
            if tc.expectErr {
                if err == nil {
                    t.Error("expected error but got none")
                }
                return
            }
            
            if err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            
            if result != tc.expected {
                t.Errorf("expected %q, got %q", tc.expected, result)
            }
        })
    }
}
```

## 📤 Submitting Changes

### Workflow

1. **Create a branch** for your feature or fix:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. **Make your changes** following the code standards above

3. **Add tests** for your changes

4. **Update documentation** if necessary

5. **Commit your changes** with clear, descriptive commit messages:
   ```bash
   git add .
   git commit -m "feat: add support for custom cursor encryption
   
   - Add WithCursorEncryption method
   - Update documentation with encryption examples
   - Add tests for encryption functionality
   
   Fixes #123"
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Create a Pull Request** on GitHub

### Commit Message Format

Use conventional commit format:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for adding tests
- `refactor:` for code refactoring
- `perf:` for performance improvements
- `chore:` for maintenance tasks

### Pull Request Guidelines

1. **Clear Title**: Use a descriptive title that summarizes the change
2. **Description**: Provide a detailed description of what was changed and why
3. **Tests**: Ensure all tests pass and add new tests for new functionality
4. **Documentation**: Update relevant documentation
5. **Breaking Changes**: Clearly mark any breaking changes

## 🐛 Reporting Issues

### Before Reporting

1. Search existing issues to avoid duplicates
2. Try to reproduce the issue with the latest version
3. Check the documentation to ensure it's not expected behavior

### Issue Template

When reporting bugs, please include:

```
**Description:**
A clear description of the issue

**Steps to Reproduce:**
1. Step 1
2. Step 2
3. Step 3

**Expected Behavior:**
What you expected to happen

**Actual Behavior:**
What actually happened

**Environment:**
- Go version: 
- Kuysor version:
- Database: 
- OS: 

**Code Sample:**
```go
// Minimal code sample that demonstrates the issue
```

**Additional Context:**
Any other context about the problem
```

### Feature Requests

For feature requests, please include:

- **Use Case**: Describe the problem you're trying to solve
- **Proposed Solution**: Your ideas for implementation
- **Alternatives**: Other solutions you've considered
- **Examples**: Code examples of how the feature would be used

## 📚 Documentation

### Types of Documentation

1. **Code Comments**: Document all exported functions and types
2. **README**: Keep the README.md up to date with examples
3. **CONTRIBUTING**: This file
4. **Examples**: Add examples in the `examples/` directory (if created)

### Documentation Standards

- Use clear, concise language
- Provide practical examples
- Keep documentation up to date with code changes
- Include common use cases and gotchas

## 🚢 Release Process

Releases are managed by the maintainers. The process includes:

1. Version bump following [Semantic Versioning](https://semver.org/)
2. Update CHANGELOG.md
3. Create and push git tag
4. Create GitHub release with release notes

### Version Guidelines

- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

## 🤝 Community Guidelines

### Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help newcomers get started
- Assume good intentions

### Getting Help

- Open an issue for bugs or feature requests
- Start discussions for questions or ideas
- Check existing issues and discussions first

## 📞 Contact

- **Issues**: Use GitHub Issues for bugs and feature requests
- **Discussions**: Use GitHub Discussions for questions and ideas
- **Security**: For security issues, please email the maintainers directly

---

Thank you for contributing to Kuysor! 🙏

Your contributions help make cursor pagination easier for the Go community. 