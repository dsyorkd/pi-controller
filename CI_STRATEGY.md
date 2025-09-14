# CI Strategy Documentation

## Current CI Approach (Updated: 2025-01-13)

### ✅ Current Working Approach
All CI workflows now use:
- **GitHub-hosted runners** (`ubuntu-latest`)
- **Runtime dependency installation** during each job
- **Dependencies installed**: make, protobuf-compiler, Go 1.25, golangci-lint v2.4.0

This approach works because:
1. ✅ Compatible with GitHub-hosted infrastructure
2. ✅ All required tools are installed fresh with correct versions
3. ✅ golangci-lint v2.4.0 is compatible with Go 1.25
4. ✅ No registry authentication or permissions needed

### 🔧 Key Issue Resolved
**Local development incompatibility**: Local environments may have older golangci-lint versions (1.x) that are incompatible with Go 1.25. The CI workflows resolve this by installing the correct golangci-lint v2.4.0 during runtime.

Error example:
```
Error: can't load config: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.25.1)
```

### ❌ Container Image Approach (Future Enhancement)
Pre-built container images (`ghcr.io/dsyorkd/ci-image/ci-go-npm:v2.0`) are **NOT CURRENTLY USED** because:

1. **Registry permissions** - Requires proper authentication setup
2. **Development overhead** - Current approach works reliably without containers
3. **Complexity vs benefit** - Runtime installation is fast and reliable

### 🔄 Future Migration Path (Optional)
If container images become necessary:
1. Set up proper registry authentication
2. Build and push container images with all dependencies
3. Update workflows to use `container: ghcr.io/dsyorkd/ci-image/ci-go-npm:v2.0`
4. Remove runtime dependency installation steps
5. Test thoroughly before merging

### 📁 Affected Files
- `.github/workflows/ci.yml` ✅ Updated
- `.github/workflows/security.yml` ✅ Updated
- `.github/workflows/build-multiarch.yml` ✅ Updated
- `.github/workflows/release.yml` ✅ Updated

**Status**: Current approach is working and production-ready. Container images are an optional future enhancement.