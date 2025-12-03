# Security Infrastructure Guide

> JWT authentication, token management, and session security for goimg-datalayer.

## Overview

This directory implements the authentication and authorization infrastructure for goimg, including:

- **JWT Service**: RS256-signed access and refresh tokens
- **Token Blacklist**: Redis-backed revocation for immediate logout
- **Session Store**: Redis-based session tracking with multi-device support
- **Refresh Token Rotation**: Automatic rotation with replay attack detection

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ├─ 1. Login (username + password)
       │
       v
┌─────────────────────────────────────────────────────┐
│  Authentication Handler                              │
│  - Validates credentials via domain layer            │
│  - Generates access + refresh tokens                 │
│  - Creates session in Redis                          │
└──────────────────────┬──────────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       │               │               │
       v               v               v
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ JWT Service │ │Session Store│ │Refresh Token│
│             │ │             │ │  Service    │
│ RS256 Sign  │ │ Redis Store │ │ Rotation +  │
│ Validation  │ │ Multi-dev   │ │ Replay Det. │
└─────────────┘ └─────────────┘ └─────────────┘
       │               │               │
       └───────────────┴───────────────┘
                       │
       2. API Request with Access Token
                       │
                       v
       ┌───────────────────────────────┐
       │  JWT Middleware               │
       │  - Validates token signature  │
       │  - Checks blacklist           │
       │  - Extracts user claims       │
       └───────────────┬───────────────┘
                       │
                       v
       ┌───────────────────────────────┐
       │  Protected Resource           │
       └───────────────────────────────┘

       3. Token Refresh (before expiry)
                       │
                       v
       ┌───────────────────────────────┐
       │  Refresh Handler              │
       │  - Validates refresh token    │
       │  - Checks for replay attack   │
       │  - Marks old token as used    │
       │  - Generates new token pair   │
       └───────────────────────────────┘
```

## JWT Implementation (RS256)

### Design Decisions

1. **Asymmetric Algorithm (RS256)**
   - Private key signs tokens (API server only)
   - Public key verifies tokens (can distribute to other services)
   - Enables microservice architecture without shared secrets
   - Key compromise only requires public key redistribution

2. **Key Size: 4096-bit RSA**
   - OWASP recommendation for 2024+
   - Provides ~140-bit security strength
   - Resistant to quantum attacks until Shor's algorithm scales
   - Enforced at service initialization (fails if key < 4096 bits)

3. **Token Types**
   - **Access Token**: Short-lived (15 minutes), used for API authentication
   - **Refresh Token**: Long-lived (7 days), used to obtain new access tokens

4. **Claims Structure**
   ```json
   {
     "user_id": "uuid",
     "email": "user@example.com",
     "role": "user|moderator|admin",
     "session_id": "uuid",
     "token_type": "access|refresh",
     "iss": "goimg-api",
     "sub": "user_id",
     "exp": 1234567890,
     "iat": 1234567890,
     "nbf": 1234567890,
     "jti": "unique-token-id"
   }
   ```

5. **No Sensitive Data in Payload**
   - JWT payloads are base64-encoded (NOT encrypted)
   - Never include passwords, API keys, or PII
   - Claims are readable by anyone with the token
   - Use minimal claims (user_id, role, session_id)

### Key Management

#### Generating RSA Keys

```bash
# Generate 4096-bit private key
openssl genrsa -out jwt_private.pem 4096

# Extract public key
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem

# Verify key size
openssl rsa -in jwt_private.pem -text -noout | grep "Private-Key"
# Should show: Private-Key: (4096 bit, 2 primes)
```

#### Key Storage

**Development**:
- Store keys in `secrets/` directory (gitignored)
- Load via environment variables:
  ```
  JWT_PRIVATE_KEY_PATH=/path/to/jwt_private.pem
  JWT_PUBLIC_KEY_PATH=/path/to/jwt_public.pem
  ```

**Production**:
- Use secrets management (AWS Secrets Manager, Vault, etc.)
- Mount as files in container (never env vars for private keys)
- Restrict file permissions: `chmod 600 jwt_private.pem`
- Rotate keys every 6-12 months

#### Key Rotation Procedure

1. Generate new key pair (keep old keys)
2. Sign new tokens with new private key
3. Validate with both old and new public keys (grace period)
4. After all tokens signed with old key expire, remove old keys
5. Update service configuration to use new keys only

**Implementation Note**: Current implementation uses single key pair. Multi-key validation requires extending `jwt_service.go` to accept multiple public keys and try each during validation.

### Token Expiration Strategy

| Token Type | TTL | Rationale |
|------------|-----|-----------|
| Access Token | 15 minutes | Limits exposure window if token is compromised |
| Refresh Token | 7 days | Balances security vs. user convenience |
| Session | 7 days | Aligned with refresh token lifetime |

**Refresh Flow**:
1. Client detects access token expiry (401 response or proactive check)
2. Sends refresh token to `/auth/refresh` endpoint
3. Server validates refresh token and checks blacklist
4. Generates new access + refresh token pair
5. Marks old refresh token as "used" for replay detection
6. Returns new tokens to client

**Security Properties**:
- Stolen access token has limited 15-minute window
- Refresh token stored securely (HttpOnly cookie or secure storage)
- Replay attack detection revokes entire token family

## Refresh Token Security

### Cryptographically Secure Generation

```go
// 32 bytes of random data = 256 bits of entropy
tokenBytes := make([]byte, 32)
crypto/rand.Read(tokenBytes)
token := base64.URLEncoding.EncodeToString(tokenBytes)
```

**Why not JWT for refresh tokens?**
- Refresh tokens don't need to be stateless
- Opaque tokens easier to revoke immediately
- SHA-256 hash prevents brute-force attacks
- Constant-time comparison prevents timing attacks

### Token Rotation with Family Tracking

Every refresh operation:
1. Validates current refresh token
2. Marks current token as "used"
3. Generates new refresh token with parent reference
4. Stores new token in same family

**Family Structure**:
```
Family ID: 123e4567-e89b-12d3-a456-426614174000
├─ Token 1 (hash: abc...) [used]
├─ Token 2 (hash: def..., parent: abc...) [used]
└─ Token 3 (hash: ghi..., parent: def...) [active]
```

**Replay Attack Detection**:
- If a "used" token is presented again → entire family revoked
- Protects against stolen refresh tokens
- User must re-authenticate after detection

### Anomaly Detection

Track metadata with each refresh token:
- IP address
- User agent string

On refresh, check for changes:
- Different IP → potential theft
- Different user agent → potential theft

**Response**: Revoke token family, require re-authentication

**Note**: Legitimate scenarios exist (VPN switching, browser updates). Consider implementing:
- IP range allowlists for mobile users
- User notification before revocation
- Step-up authentication instead of immediate revocation

## Token Blacklist

### Why Blacklist?

- Immediate logout across all devices
- Revoke tokens on security incident
- Account suspension/deletion
- Password change invalidates all tokens

### Redis Key Pattern

```
goimg:blacklist:{jti}
TTL: Remaining token lifetime
Value: Expiration timestamp (for debugging)
```

**TTL Strategy**: Store token in blacklist until natural expiration. Redis automatically removes expired entries.

### Performance Considerations

**Middleware Check**:
```go
// 1. Extract JTI from token (fast, no validation)
jti := extractTokenID(tokenString)

// 2. Check blacklist (Redis GET, ~1ms)
if blacklist.IsBlacklisted(ctx, jti) {
    return Unauthorized
}

// 3. Full token validation (crypto verification, ~5-10ms)
claims := jwt.ValidateToken(tokenString)
```

**Optimization**: Check blacklist before expensive signature verification.

### Blacklist Operations

| Operation | Use Case | Performance |
|-----------|----------|-------------|
| Add | Logout, revoke access | O(1) - Redis SET |
| Check | Every API request | O(1) - Redis GET |
| Remove | Admin un-revoke (rare) | O(1) - Redis DEL |
| Clear | Testing only | O(N) - Redis SCAN + DEL |

## Session Management

### Session Store (Redis)

**Purpose**: Track active sessions for multi-device support and security monitoring.

**Key Patterns**:
```
goimg:session:{session_id}       → Session metadata (JSON)
goimg:user:sessions:{user_id}    → Set of session IDs
```

**Session Data**:
```go
type Session struct {
    SessionID string
    UserID    string
    Email     string
    Role      string
    IP        string
    UserAgent string
    CreatedAt time.Time
    ExpiresAt time.Time
}
```

### Multi-Device Support

Users can:
1. View all active sessions (devices)
2. Revoke specific session
3. Logout all devices

**Implementation**:
```go
// Get all sessions for user
sessions := sessionStore.GetUserSessions(ctx, userID)

// Revoke specific session
sessionStore.Revoke(ctx, sessionID)

// Revoke all sessions (logout everywhere)
sessionStore.RevokeAll(ctx, userID)
```

### Session Security

1. **Session Fixation Prevention**: Generate new session ID on login
2. **Session Hijacking Mitigation**: Track IP/User-Agent, alert on changes
3. **Concurrent Session Limits**: Optional limit on active sessions per user
4. **Idle Timeout**: Redis TTL automatically expires inactive sessions

## Security Checklist

### Implementation Requirements

- [x] JWT private key >= 4096-bit RSA
- [x] Refresh tokens stored as SHA-256 hash
- [x] Token rotation on every refresh
- [x] Replay attack detection implemented
- [x] Constant-time token comparison
- [x] No sensitive data in JWT payload
- [x] Proper key loading (not hardcoded)
- [x] Token blacklist for revocation
- [x] Session expiry aligned with refresh token TTL
- [x] Family tracking for token chains
- [x] IP/User-Agent anomaly detection

### Deployment Security

- [ ] Private keys stored in secrets manager
- [ ] Key file permissions set to 600
- [ ] Redis requires authentication (requirepass)
- [ ] Redis over TLS in production
- [ ] Rate limiting on auth endpoints
- [ ] Audit logging for token operations
- [ ] Monitoring for replay attack attempts
- [ ] Alerting for anomaly detection triggers

## Testing

### Unit Tests

All components have comprehensive unit tests:
- `jwt_service_test.go`: Token generation/validation
- `token_blacklist_test.go`: Blacklist operations
- `refresh_token_service_test.go`: Rotation and replay detection
- `session_store_test.go`: Session CRUD operations

**Coverage Target**: 90% minimum (security-critical code)

### Integration Testing

Required tests:
1. End-to-end login flow
2. Token refresh with rotation
3. Replay attack detection
4. Multi-device session management
5. Logout (single and all devices)
6. Token expiration handling

### Security Testing

Use `gosec` for static analysis:
```bash
gosec ./internal/infrastructure/security/...
```

**Critical Checks**:
- No hardcoded secrets
- Secure random number generation
- Constant-time comparisons
- Proper error handling (no info leakage)

## Common Pitfalls

### 1. Token Storage in Frontend

**Wrong**:
```javascript
localStorage.setItem('access_token', token);
```

**Right**:
```javascript
// HttpOnly cookie (inaccessible to JavaScript)
// OR secure storage (mobile apps)
```

### 2. Missing Blacklist Check

**Wrong**:
```go
// Just validate signature
claims := jwt.ValidateToken(token)
```

**Right**:
```go
// Check blacklist first
if blacklist.IsBlacklisted(ctx, jti) {
    return ErrTokenRevoked
}
claims := jwt.ValidateToken(token)
```

### 3. No Token Rotation

**Wrong**: Reuse same refresh token indefinitely

**Right**: Generate new refresh token on every use

### 4. Ignoring Anomalies

**Wrong**: Accept any valid token regardless of IP change

**Right**: Detect and respond to suspicious behavior

## Monitoring and Metrics

### Key Metrics

1. **Authentication Rate**
   - Login attempts (success/failure)
   - Token refresh rate
   - Average session duration

2. **Security Events**
   - Replay attacks detected
   - Anomalies detected (IP/UA changes)
   - Token revocations
   - Failed validation attempts

3. **Performance**
   - Token generation latency (p50, p95, p99)
   - Token validation latency
   - Redis operation latency
   - Blacklist hit rate

### Alerting

**Critical Alerts**:
- Replay attack detection spike
- Redis unavailable (auth will fail)
- Private key load failure
- High rate of token validation failures

**Warning Alerts**:
- Anomaly detection rate increase
- Unusual session creation patterns
- Token expiration errors (clock skew)

## Future Enhancements

### Planned Improvements

1. **Multi-Key Support**: Validate tokens with multiple public keys (key rotation)
2. **Token Binding**: Bind tokens to TLS certificate or device fingerprint
3. **Geolocation Tracking**: Enhance anomaly detection with country-level IP tracking
4. **Machine Learning**: Detect abnormal usage patterns
5. **Hardware Security Modules**: Store private keys in HSM for production
6. **Certificate-Based Auth**: Support mTLS for service-to-service authentication

### OAuth2 Integration

For third-party authentication:
- Implement OAuth2 authorization server
- Support authorization code flow
- PKCE for mobile/SPA clients
- Scope-based access control

## References

- [RFC 7519: JWT](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 7515: JWS (JSON Web Signature)](https://datatracker.ietf.org/doc/html/rfc7515)
- [OWASP JWT Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- [NIST SP 800-57: Key Management](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)

## Support

For security vulnerabilities, contact: security@goimg.example.com

**DO NOT** open public GitHub issues for security bugs.
