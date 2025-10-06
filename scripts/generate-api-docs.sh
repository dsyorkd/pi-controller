#!/bin/bash
set -e

echo "Generating OpenAPI documentation..."

# Create output directory
mkdir -p docs/api

# Generate swagger docs
swag init \
  --dir ./internal/api,./internal/api/handlers \
  --generalInfo server.go \
  --output ./docs/api \
  --outputTypes yaml,json \
  --parseInternal \
  --parseDependency

echo "✅ API documentation generated successfully"
echo "  - YAML: docs/api/openapi.yaml"
echo "  - JSON: docs/api/swagger.json"
