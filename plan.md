1. **Remove Trivy version pinning:** As per the memory: "To prevent missing release asset errors with `aquasecurity/trivy-action` in CI workflows, remove the `version:` parameter entirely to allow it to default to its stable embedded binary instead of explicitly pinning unavailable versions like `v0.55.2`." I will update `.github/workflows/security.yml` to remove the `version: 'v0.55.2'` lines under the `aquasecurity/trivy-action` steps.
2. **Pre-commit checks:** Run `make pre-commit`.
3. **Commit and Submit:** Create PR and submit.
