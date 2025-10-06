#!/bin/bash
echo "=== Pi Controller Test Coverage Summary ==="
echo ""
echo "Running tests with coverage..."
go test ./internal/... -cover 2>&1 | grep -E "^(ok|FAIL|coverage:)" | grep -v "level=" | sort -k2
