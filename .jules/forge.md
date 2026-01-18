## 2025-01-20 - Pinned GitHub Actions and Updated Dockerfiles
**Issue:** GitHub Actions workflows were using unpinned versions (tags) for `trivy-action` and `codeql-action/upload-sarif`, which poses a supply chain security risk and could break if tags are updated with breaking changes. Also, Dockerfiles were using an outdated Go version (1.22) while the project requires 1.25.
**Root Cause:** Initial setup likely used convenient tags and didn't update the Docker base image when the project upgraded Go versions.
**Fix:** Pinned all actions to specific commit SHAs (found via release tags) and updated Dockerfiles to use `golang:1.25-alpine`.

## 2025-05-15 - Docker Runtime ABI Mismatch
**Issue:** `Dockerfile.api` build was fragile or failing at runtime due to library incompatibilities between the build stage (Alpine/musl) and the runtime stage (Distroless Debian/glibc).
**Root Cause:** Using an Alpine-based builder (`golang:1.25-alpine`) with a Debian-based runtime (`distroless/base-debian12`) causes ABI incompatibility, especially for CGO-enabled applications like `bimg`/`libvips` which link against `musl` but try to run on `glibc`.
**Fix:** Switched the runtime image to `alpine:3.20` to match the builder's OS family (Alpine). Used `apk add` to install `vips` and dependencies, eliminating manual library copying and ensuring ABI compatibility.
