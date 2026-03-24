1. **Fix timing attack vulnerability in `InvitationToken.Equals`**
   - The method `Equals` in `internal/domain/community/invitation_token.go` claims to use constant-time comparison to prevent timing attacks but actually uses the standard `==` string comparison operator.
   - I will update it to use `crypto/subtle.ConstantTimeCompare` by converting the strings to byte slices first, which addresses the memory entry: "To prevent timing attacks, sensitive string comparisons (such as token validation in the domain layer, e.g., `InvitationToken.Equals`) must use `crypto/subtle.ConstantTimeCompare` after converting strings to byte slices, rather than using the standard `==` operator."

2. **Add security journal entry**
   - I will create or update `.jules/sentinel.md` to add a journal entry about fixing the timing attack vulnerability, specifically focusing on using constant time comparisons for secrets.

3. **Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done**
   - Run `make pre-commit`, `make test`, and domain test coverage checks.

4. **Submit the changes**
   - I'll submit a PR with the title `security: [HIGH] Fix timing attack vulnerability in InvitationToken.Equals`.
