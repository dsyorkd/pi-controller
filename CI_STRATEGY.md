# CI Strategy Documentation

## Current CI Approach (Updated: 2025-01-13)

### ❌ Container Image Approach (Not Currently Used)
We initially planned to use pre-built container images (`ghcr.io/dsyorkd/ci-image/ci-go-npm:v2.0`) with all dependencies pre-installed. However, this approach is **NOT CURRENTLY IMPLEMENTED** because:

1. **Container images are not publicly accessible** - The registry requires authentication
2. **Self-hosted runners lack Docker** - Our k3s-01 runner doesn't have Docker available
3. **Disk space issues** - Self-hosted runner has insufficient storage

### ✅ Current Working Approach
All CI workflows now use:
- **GitHub-hosted runners** (`ubuntu-latest`)
- **Runtime dependency installation** during each job
- **Dependencies installed**: make, protobuf-compiler, Go 1.25, golangci-lint v2.4.0

### 🔄 Future Migration Path
When ready to use container images:
1. Make container images publicly accessible OR configure registry authentication
2. Update all workflows to use `container: ghcr.io/dsyorkd/ci-image/ci-go-npm:v2.0`
3. Remove runtime dependency installation steps
4. Test thoroughly before merging

### 📁 Affected Files
- `.github/workflows/ci.yml`
- `.github/workflows/security.yml`
- `.github/workflows/build-multiarch.yml`
- `.github/workflows/release.yml`

**IMPORTANT**: Do not attempt to use container images in workflows until this strategy is explicitly updated.