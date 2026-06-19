## 2024-06-20 - Trivy Action SHA Update
**Issue:** `aquasecurity/trivy-action` was pinned to a commit SHA `915b19bbe73b92a6cf82a1bc12b087c9a19a5fe2` but referenced version tag `0.28.0` which is outdated and causes issues as it references an invalid/deleted `setup-trivy` tag.
**Root Cause:** Using an older version of the `aquasecurity/trivy-action` action.
**Fix:** Updated the pinned action to the verified stable commit hash for `v0.34.0` (`c1824fd6edce30d7ab345a9989de00bbd46ef284`) to prevent 'Unable to resolve action aquasecurity/setup-trivy' or binary download errors in CI workflows.
