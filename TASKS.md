# Tasks

## build

Build the application binary.

```bash
go build -o reazy ./cmd/reazy
```

## run

Run the Reazy application.

```bash
go run ./cmd/reazy
```

## test

Run all unit tests.

```bash
go test ./...
```

## cover

Run tests with coverage and open HTML report.

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
rm coverage.out
```

## clean

Remove coverage artifacts.

```bash
go clean
rm -f coverage.out
```
