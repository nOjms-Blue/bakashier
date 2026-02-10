#!/usr/bin/env bash

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed or not in PATH. Please install Go and try again." >&2
  exit 1
fi

go build -o bakashier main.go
