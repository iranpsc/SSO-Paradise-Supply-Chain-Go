# SSO Paradise Supply Chain — Go

Minimal Go starter project, with no external dependencies.

## Requirements

- Go 1.24.3 or later

## Run

```sh
go run .
```

## Build

```sh
go build -o bin/sso-paradise-supply-chain .
```

On Windows, append `.exe` to the output filename.

## Check

```sh
go test ./...
go vet ./...
```

The project currently contains only a starter entry point in `main.go`.
