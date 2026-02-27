# Contributing to Keel CLI

Thank you for your interest in contributing to Keel CLI! 🎉

## Getting Started

1. **Fork the repository**
2. **Clone your fork**
   ```bash
   git clone https://github.com/YOUR_USERNAME/ss-keel-cli.git
   cd ss-keel-cli
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Create a branch**
   ```bash
   git checkout -b feat/your-feature-name
   ```

## Development

### Running locally

```bash
# Run the CLI in development
go run . [command]

# Example
go run . new test-app
go run . generate module users
```

### Building

```bash
# Build the binary
make build

# Or use go directly
go build -o keel .
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific tests
go test -v ./internal/generator/...
```

## Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages:

### Format
```
<type>(optional-scope): short descriptive summary
```

### Examples
```bash
feat(config): add GetEnvUint helper
fix(parser): handle nil body in ParseBody
docs(readme): add validation examples
test(router): add table-driven tests for route builder
refactor(core): simplify openapi path conversion
chore(ci): update workflow permissions
```

### Allowed Types
- `feat` — new feature
- `fix` — bug fix
- `docs` — documentation changes
- `test` — test additions or modifications
- `refactor` — code restructuring without behavior change
- `chore` — maintenance, tooling, config
- `ci` — CI/CD updates
- `perf` — performance improvements

### Rules
- Use present tense
- Keep messages concise but descriptive
- Do not use vague messages like "update", "fix stuff", or "wip"
- Do not mix unrelated concerns in a single commit
- Separate features, fixes, refactors, and docs into different commits

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Keep functions focused and small
- Add comments for exported functions
- Write descriptive variable names

### Formatting

```bash
# Format code
make fmt

# Or use go directly
go fmt ./...
```

### Linting

```bash
# Run linter
make lint
```

## Pull Request Process

1. **Update documentation** if needed
2. **Add tests** for new features
3. **Ensure all tests pass**
   ```bash
   make test
   ```
4. **Update CHANGELOG.md** with your changes
5. **Create a Pull Request** with a clear description

### PR Title Format

Follow the same format as commit messages:
```
feat(generator): add support for custom templates
```

### PR Description Template

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How has this been tested?

## Checklist
- [ ] Code follows project style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] All tests pass
```

## Adding New Features

### Adding a New Command

1. Create a new file in `cmd/` (e.g., `cmd/mycommand.go`)
2. Follow the existing command structure
3. Register it in `cmd/root.go` or the relevant parent command
4. Add tests in `cmd/mycommand_test.go`
5. Update documentation

### Adding New Templates

1. Add template file in `internal/generator/templates/`
2. Update `generator.go` to handle the new template
3. Add tests
4. Update documentation with usage examples

## Questions?

Feel free to:
- Open an issue for discussion
- Ask in pull request comments
- Contact the maintainers

## Code of Conduct

Be respectful, inclusive, and constructive. We're here to build something great together! 🚀

---

Thank you for contributing! ⚓
