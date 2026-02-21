## 2026-02-21 - Testcontainers vs CI Services
**Issue:** `test-integration` CI job was defining Postgres/Redis services but tests were using `testcontainers-go` to spin up their own ephemeral containers.
**Root Cause:** Redundant configuration in `ci.yml`. The tests ignored the CI-provided services and spun up new ones, leading to wasted resources and potential confusion.
**Fix:** Removed `services` block and `env` variables from the `test-integration` job in `ci.yml`. Confirmed `go.mod` depends on `testcontainers-go`.
