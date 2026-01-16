# Agent Instructions

## Project

Nexo - A file-system based Go framework for APIs and websites, inspired by Next.js App Router.

**Repository:** https://github.com/abdul-hamid-achik/nexo

## Build Commands

```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run specific package tests
go test ./pkg/nexo/scanner/...
go test ./pkg/nexo/generator/...

# Build CLI
go build -o bin/nexo ./cmd/nexo

# Install locally
go install ./cmd/nexo

# Run linter
golangci-lint run --build-tags=nexo
```

## Development Workflow

1. Clone the repo if not already done
2. Create new packages in `pkg/nexo/scanner/` and `pkg/nexo/generator/`
3. Write tests alongside implementation
4. Run tests frequently: `go test ./...`
5. Update CLI commands in `cmd/nexo/`

## File Locations

- Scanner package: `pkg/nexo/scanner/`
- Generator package: `pkg/nexo/generator/`
- CLI commands: `cmd/nexo/`
- Templates: `templates/`

## Testing

Run tests with verbose output:
```bash
go test -v ./pkg/nexo/scanner/...
go test -v ./pkg/nexo/generator/...
```

Run integration tests:
```bash
go test -v ./pkg/nexo/...
```

## Code Style

- Use `gofmt` for formatting
- Follow Go conventions
- Add comments for exported functions
- Write table-driven tests

## Important Notes

- Route files in `app/` must have `//go:build nexo` tag
- Generated code goes in `.nexo/generated/`
- Never manually edit generated files
- Keep backward compatibility with underscore convention during transition