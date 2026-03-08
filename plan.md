Wait, why did Trivy use v0.56.1 when I explicitly removed `version: 'v0.55.2'`?
In `aquasecurity/trivy-action@0.28.0` (which is what `915b19bbe73b92a6cf82a1bc12b087c9a19a5fe2` resolves to), it internally downloads Trivy. Since I did not specify the version, it defaults to downloading the latest release of Trivy. But maybe `0.28.0` is very old, and it has some bug downloading Trivy? Or maybe `v0.56.1` is the latest and it has a bug in its installation script? Wait, `0.28.0` has `trivy-version` parameter, actually, wait.
Let's look at `dependabot` branch info from the `git fetch` output:
`dependabot/github_actions/aquasecurity/trivy-action-0.34.0`
So there's an open PR to upgrade `trivy-action` to `0.34.0`? No, wait. I can just update the action to the latest `master` or a newer version! The prompt says "Infrastructure GitHub Actions code must strictly use SHA-pinned action versions (e.g., uses: actions/checkout@<sha> # v4.1.1) instead of solely relying on tag versions".

Okay, if I specify a valid version, say `v0.48.3`? No, I tried and `0.48.3` download URL was a 404.
Wait, let's look at the Trivy releases on GitHub.
Let's write a python script to hit GitHub API and get the latest releases of Trivy.
