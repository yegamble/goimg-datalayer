# Two-Factor Authentication (2FA) E2E Testing Guide

This guide explains how to run the Newman/Postman E2E tests for the Two-Factor Authentication endpoints.

## Test Coverage

The 2FA test suite includes **13 comprehensive test requests** covering:

### Happy Path Tests
1. **Setup 2FA - Success**: Initiates 2FA setup and receives TOTP secret + backup codes
2. **Get 2FA Status - Setup Pending**: Verifies status shows setup pending verification
3. **Verify 2FA - Success (Manual)**: Enables 2FA with valid TOTP code
4. **Get 2FA Status - Enabled**: Confirms 2FA is active
5. **Regenerate Backup Codes - Success**: Generates new backup codes
6. **Disable 2FA - Success**: Disables 2FA with correct password
7. **Get 2FA Status - Disabled**: Confirms 2FA is disabled

### Error Handling Tests
8. **Verify 2FA - Invalid Code**: Tests invalid TOTP code (400 Bad Request)
9. **Setup 2FA - No Token**: Tests unauthenticated access (401 Unauthorized)
10. **Verify 2FA - No Setup Initiated**: Tests verification without setup (404 Not Found)
11. **Disable 2FA - Not Enabled**: Tests disabling when not enabled (404/400)
12. **Disable 2FA - Wrong Password**: Tests password validation (401 Unauthorized)
13. **Regenerate Backup Codes - Wrong Password**: Tests password validation (401 Unauthorized)

## Test Organization

The 2FA tests are organized as a subfolder within the **Auth** folder:

```
GoImg API E2E Tests
└── Auth
    ├── Register - Success
    ├── Login - Success
    ├── ... (other auth tests)
    └── 2FA
        ├── Setup 2FA - Success
        ├── Get 2FA Status - Setup Pending
        ├── Verify 2FA - Invalid Code
        ├── Verify 2FA - Success (Manual)
        ├── Get 2FA Status - Enabled
        ├── Regenerate Backup Codes - Success
        ├── Regenerate Backup Codes - Wrong Password
        ├── Disable 2FA - Success
        ├── Get 2FA Status - Disabled
        ├── Setup 2FA - No Token
        ├── Verify 2FA - No Setup Initiated
        ├── Disable 2FA - Not Enabled
        └── Disable 2FA - Wrong Password
```

## Prerequisites

1. **API Server Running**: Ensure the goimg-datalayer API is running on `http://localhost:8080`
2. **Newman CLI Installed**:
   ```bash
   npm install -g newman
   ```
3. **Valid Access Token**: The tests require authentication, so you must:
   - Run the registration and login tests first, OR
   - Manually set the `accessToken` environment variable

## Running the Tests

### Run All E2E Tests (Including 2FA)

```bash
# From project root
make test-e2e

# Or manually
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json
```

### Run Only 2FA Tests

```bash
newman run tests/e2e/postman/goimg-api.postman_collection.json \
  -e tests/e2e/postman/ci.postman_environment.json \
  --folder "2FA"
```

### Run in Postman UI

1. Import `tests/e2e/postman/goimg-api.postman_collection.json` into Postman
2. Import `tests/e2e/postman/ci.postman_environment.json` as an environment
3. Select the "GoImg CI Environment" environment
4. Navigate to the **Auth > 2FA** folder
5. Run the folder or individual requests

## Important Notes

### TOTP Code Generation (Manual Test)

The **"Verify 2FA - Success (Manual)"** test requires a valid TOTP code, which cannot be fully automated without additional libraries. To run this test:

#### Option 1: Manual Testing (Recommended for Development)
1. Run the "Setup 2FA - Success" test
2. Copy the `secret` from the response
3. Add the secret to Google Authenticator, Authy, or similar app
4. Generate a 6-digit code
5. Update the request body in "Verify 2FA - Success (Manual)" with the generated code
6. Run the test

#### Option 2: Automated Testing (CI/CD)
For automated CI/CD pipelines, you can programmatically generate TOTP codes using a library:

**Node.js Example (Pre-request Script)**:
```javascript
// Install: npm install otpauth
const OTPAuth = require('otpauth');

const secret = pm.environment.get('totpSecret');
const totp = new OTPAuth.TOTP({
    secret: OTPAuth.Secret.fromBase32(secret)
});

const code = totp.generate();
pm.environment.set('currentTOTP', code);

// Update request body
const requestBody = JSON.parse(pm.request.body.raw);
requestBody.code = code;
pm.request.body.raw = JSON.stringify(requestBody);
```

**Python Example (Newman Pre-run Script)**:
```python
import pyotp
import os

secret = os.environ.get('TOTP_SECRET')
totp = pyotp.TOTP(secret)
current_code = totp.now()
print(f"Current TOTP: {current_code}")
```

### Environment Variables

The following environment variables are used by the 2FA tests:

| Variable | Type | Description | Set By |
|----------|------|-------------|--------|
| `baseUrl` | default | API base URL (e.g., `http://localhost:8080/api/v1`) | Pre-configured |
| `accessToken` | secret | JWT access token for authentication | Login test |
| `testPassword` | default | Test user password | Collection variable |
| `totpSecret` | secret | TOTP secret from setup response | Setup 2FA test |
| `backupCodes` | secret | Array of backup codes (JSON string) | Setup/Regenerate tests |

### Test Assertions

All tests include comprehensive assertions:

1. **Status Code Validation**: Ensures correct HTTP status codes (200, 400, 401, 404, 409)
2. **Response Structure**: Validates JSON schema matches OpenAPI spec
3. **RFC 7807 Problem Details**: Error responses follow the RFC 7807 standard
4. **Business Logic**: Validates 2FA state transitions and data integrity
5. **Security**: Validates authentication and authorization requirements

### Test Flow

The tests are designed to run sequentially in the following order:

1. **Setup** → Initiates 2FA and receives secret
2. **Status (Pending)** → Confirms setup is pending
3. **Verify (Invalid)** → Tests error handling
4. **Verify (Success)** → Enables 2FA
5. **Status (Enabled)** → Confirms 2FA is active
6. **Regenerate Codes** → Tests backup code regeneration
7. **Disable** → Disables 2FA
8. **Status (Disabled)** → Confirms 2FA is disabled
9. **Error Cases** → Tests various error scenarios

## OpenAPI Specification Compliance

All tests are designed to match the OpenAPI 3.0.3 specification defined in:
```
/home/user/goimg-datalayer/api/openapi/openapi.yaml
```

The following endpoints are tested:
- `POST /api/v1/auth/2fa/setup`
- `POST /api/v1/auth/2fa/verify`
- `POST /api/v1/auth/2fa/disable`
- `GET /api/v1/auth/2fa/status`
- `POST /api/v1/auth/2fa/backup-codes/regenerate`

## Troubleshooting

### Tests Fail with 401 Unauthorized

**Cause**: Missing or expired access token

**Solution**:
1. Run the registration and login tests first to obtain a valid token
2. Or manually set the `accessToken` environment variable

### "Verify 2FA - Success" Always Fails

**Cause**: The placeholder code `123456` is invalid

**Solution**: Replace with a valid TOTP code from your authenticator app (see "TOTP Code Generation" above)

### Tests Show "2FA Already Enabled"

**Cause**: Previous test run left 2FA enabled

**Solution**:
1. Manually disable 2FA via API or database
2. Or use a fresh test user account

### Backup Codes Not Matching

**Cause**: Backup codes were regenerated between tests

**Solution**: Run tests sequentially to maintain state consistency

## CI/CD Integration

For GitHub Actions or similar CI/CD pipelines:

```yaml
- name: Run 2FA E2E Tests
  run: |
    docker-compose -f docker/docker-compose.yml up -d
    make migrate-up
    make run &
    sleep 5  # Wait for API to start
    make test-e2e
```

## Security Considerations

1. **Secrets**: TOTP secrets and backup codes are marked as `secret` type in environment variables
2. **Never Commit**: Do not commit environment files with actual secrets
3. **Test Data Only**: Use dedicated test accounts, not production data
4. **Token Expiry**: Tests assume tokens are valid for the duration of the test run

## Contributing

When adding new 2FA tests:

1. Follow the existing naming convention: `<Action> - <Scenario>`
2. Include comprehensive test assertions (status code, schema, business logic)
3. Add test descriptions explaining the purpose
4. Ensure error tests validate RFC 7807 Problem Details format
5. Update this README with any new test scenarios

## See Also

- Project Testing Guide: `/home/user/goimg-datalayer/claude/test_strategy.md`
- API Security Guide: `/home/user/goimg-datalayer/claude/api_security.md`
- OpenAPI Specification: `/home/user/goimg-datalayer/api/openapi/openapi.yaml`
- Sprint 11 Implementation: `/home/user/goimg-datalayer/claude/sprint_plan.md`
