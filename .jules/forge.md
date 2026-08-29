## 2024-06-20 - Trivy Action SHA Update
**Issue:** `aquasecurity/trivy-action` was pinned to a commit SHA `915b19bbe73b92a6cf82a1bc12b087c9a19a5fe2` but referenced version tag `0.28.0` which is outdated and causes issues as it references an invalid/deleted `setup-trivy` tag.
**Root Cause:** Using an older version of the `aquasecurity/trivy-action` action.
**Fix:** Updated the pinned action to the verified stable commit hash for `v0.34.0` (`c1824fd6edce30d7ab345a9989de00bbd46ef284`) to prevent 'Unable to resolve action aquasecurity/setup-trivy' or binary download errors in CI workflows.

## 2024-06-20 - Node.js 20 Deprecation in GitHub Actions
**Issue:** Actions checkout, cache, setup-go, and upload-artifact triggered deprecation warnings because they were relying on Node.js 20.
**Root Cause:** Older versions of official actions (e.g. actions/checkout@v4.1.1) use Node.js 20 which is deprecated by GitHub.
**Fix:** Updated the action SHAs to the latest versions that use Node.js 24: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 (v4.2.2), actions/setup-go@3041bf56c941b39c61721a86cd11f3bb1338122a (v5.2.0), actions/upload-artifact@65c4c4a1ddee5b72f698fdd19549f0f0fb45cf08 (v4.6.0), and actions/cache@d4323d4df104b026a6aa633fdb11d772146be0bf (v4.2.2).

## 2024-06-20 - Missing Arguments in ReconstructUser
**Issue:** Integration tests failed with `not enough arguments in call to identity.ReconstructUser`.
**Root Cause:** `ReconstructUser` signature in `internal/domain/identity/user.go` was updated to include `emailVerified` and `emailVerifiedAt` but tests were not updated.
**Fix:** Updated `identity.ReconstructUser` calls in `tests/integration/user_repository_test.go` to include `false` and `nil` for the new arguments.
