# OAuth HTTP Handler Implementation

## Summary

Completed the OAuth HTTP handler implementation for Sprint 12, providing full OAuth 2.0 authentication support for Google and GitHub providers.

## Files Created

### 1. Query Handler
**File**: `/home/user/goimg-datalayer/internal/application/identity/queries/list_oauth_accounts.go`

- **Purpose**: Query handler to list all OAuth accounts linked to a user
- **Responsibility**: Retrieves OAuth accounts from repository and converts to DTOs
- **Returns**: `OAuthAccountListDTO` with provider, email, display name, avatar, and linked date

### 2. HTTP Handler
**File**: `/home/user/goimg-datalayer/internal/interfaces/http/handlers/oauth_handler.go`

- **Purpose**: HTTP interface for OAuth 2.0 authentication flows
- **Dependencies**:
  - Application layer command/query handlers
  - OAuth provider factory
  - Redis client for CSRF state management
- **Endpoints**: 5 total (2 public, 3 protected)

## Endpoints Implemented

### Public Endpoints (No Authentication Required)

#### 1. `GET /auth/oauth/{provider}` - Initiate OAuth Flow
- **Purpose**: Redirects user to OAuth provider authorization page
- **Flow**:
  1. Validates provider (google, github)
  2. Generates cryptographically secure CSRF state (32 bytes random)
  3. Stores state in Redis with 10-minute TTL
  4. Creates OAuth provider instance
  5. Redirects to provider authorization URL with state parameter
- **Security**: CSRF protection via random state parameter stored in Redis

#### 2. `GET /auth/oauth/{provider}/callback` - Handle OAuth Callback
- **Purpose**: Processes OAuth callback and authenticates user
- **Flow**:
  1. Extracts code and state from query parameters
  2. Validates CSRF state from Redis (one-time use)
  3. Verifies provider matches stored state
  4. Delegates to `AuthenticateWithOAuthCommand` handler
  5. Returns JWT tokens and user data
- **Response**: `AuthResponseDTO` (user + token pair)
- **Security**: State validation prevents CSRF attacks

### Protected Endpoints (JWT Authentication Required)

#### 3. `POST /auth/oauth/link` - Link OAuth Account
- **Purpose**: Links OAuth provider to existing authenticated user
- **Flow**:
  1. Validates JWT and extracts user context
  2. Validates CSRF state (same as callback)
  3. Delegates to `LinkOAuthAccountCommand` handler
  4. Returns linked account details
- **Response**: `OAuthAccountDTO`
- **Security**: Prevents linking OAuth account to different user

#### 4. `DELETE /auth/oauth/{provider}` - Unlink OAuth Account
- **Purpose**: Removes OAuth provider link from user account
- **Flow**:
  1. Validates JWT and extracts user context
  2. Validates provider from path parameter
  3. Delegates to `UnlinkOAuthAccountCommand` handler
  4. Returns success message
- **Response**: `MessageDTO`
- **Security**: Cannot unlink last authentication method

#### 5. `GET /auth/oauth/accounts` - List Linked Accounts
- **Purpose**: Returns all OAuth accounts linked to current user
- **Flow**:
  1. Validates JWT and extracts user context
  2. Delegates to `ListOAuthAccountsQuery` handler
  3. Returns list of linked accounts
- **Response**: `OAuthAccountListDTO`

## Security Features

### 1. CSRF Protection
- **State Generation**: Cryptographically secure 32-byte random state
- **State Storage**: Redis with 10-minute TTL
- **State Validation**: One-time use, deleted after validation
- **State Binding**: State tied to specific OAuth provider

### 2. Error Mapping
Comprehensive error handling with RFC 7807 Problem Details:
- `ErrOAuthAccountNotFound` → 404
- `ErrOAuthAccountExists` → 409 (already linked to another user)
- `ErrOAuthProviderNotLinked` → 404
- `ErrAccountSuspended` → 403
- `ErrAccountDeleted` → 403
- `ErrInvalidCredentials` → 401
- Generic errors → 500 (no internal details exposed)

### 3. Authentication Metadata
Captures for security auditing:
- IP address (respects X-Forwarded-For)
- User agent string
- Logged with all OAuth operations

## Integration Requirements

### 1. OAuth Provider Factory Adapter

The application commands expect an `OAuthProviderFactory` interface:

```go
type OAuthProviderFactory interface {
    CreateProvider(providerType domainidentity.OAuthProvider) (OAuthProvider, error)
    Encryptor() TokenEncryptor
}
```

However, the infrastructure factory requires provider configuration. You need to create an **adapter** that:

1. Stores configurations for each provider (loaded from environment)
2. Implements the application-layer interface
3. Internally calls infrastructure factory with correct config

**Example Adapter** (create in `internal/application/identity/services/oauth_provider_adapter.go`):

```go
package services

import (
    "fmt"

    "github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
    domainidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
    "github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
)

// OAuthProviderFactoryAdapter adapts infrastructure OAuthProviderFactory
// to application layer interface by managing provider configurations.
type OAuthProviderFactoryAdapter struct {
    factory *security.OAuthProviderFactory
    configs map[domainidentity.OAuthProvider]security.OAuthProviderConfig
}

// NewOAuthProviderFactoryAdapter creates a new adapter.
func NewOAuthProviderFactoryAdapter(
    factory *security.OAuthProviderFactory,
    googleCfg, githubCfg security.OAuthProviderConfig,
) *OAuthProviderFactoryAdapter {
    return &OAuthProviderFactoryAdapter{
        factory: factory,
        configs: map[domainidentity.OAuthProvider]security.OAuthProviderConfig{
            domainidentity.OAuthProviderGoogle: googleCfg,
            domainidentity.OAuthProviderGitHub: githubCfg,
        },
    }
}

// CreateProvider implements the application layer interface.
func (a *OAuthProviderFactoryAdapter) CreateProvider(
    providerType domainidentity.OAuthProvider,
) (commands.OAuthProvider, error) {
    cfg, exists := a.configs[providerType]
    if !exists {
        return nil, fmt.Errorf("no configuration for provider: %s", providerType)
    }

    return a.factory.CreateProvider(providerType, cfg)
}

// Encryptor returns the encryptor from the infrastructure factory.
func (a *OAuthProviderFactoryAdapter) Encryptor() commands.TokenEncryptor {
    return a.factory.Encryptor()
}
```

### 2. Environment Configuration

Add to `.env` or environment variables:

```bash
# Google OAuth
OAUTH_GOOGLE_CLIENT_ID=your-google-client-id
OAUTH_GOOGLE_CLIENT_SECRET=your-google-client-secret
OAUTH_GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/google/callback

# GitHub OAuth
OAUTH_GITHUB_CLIENT_ID=your-github-client-id
OAUTH_GITHUB_CLIENT_SECRET=your-github-client-secret
OAUTH_GITHUB_REDIRECT_URL=http://localhost:8080/api/v1/auth/oauth/github/callback

# AES encryption key for OAuth tokens (32 bytes base64-encoded)
OAUTH_TOKEN_ENCRYPTION_KEY=generate-with-openssl-rand-base64-32
```

Generate encryption key:
```bash
openssl rand -base64 32
```

### 3. Router Integration

Add OAuth handler to HTTP server router (in `internal/interfaces/http/router.go`):

```go
// OAuth routes (public + protected)
r.Route("/api/v1/auth/oauth", func(r chi.Router) {
    r.Mount("/", oauthHandler.Routes(jwtAuthMiddleware))
})
```

### 4. Dependency Injection

In `cmd/api/main.go` or server initialization:

```go
// Load OAuth configs from environment
googleCfg := security.OAuthProviderConfig{
    ClientID:     os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
    RedirectURL:  os.Getenv("OAUTH_GOOGLE_REDIRECT_URL"),
}

githubCfg := security.OAuthProviderConfig{
    ClientID:     os.Getenv("OAUTH_GITHUB_CLIENT_ID"),
    ClientSecret: os.Getenv("OAUTH_GITHUB_CLIENT_SECRET"),
    RedirectURL:  os.Getenv("OAUTH_GITHUB_REDIRECT_URL"),
}

// Create infrastructure factory
encryptionKey := os.Getenv("OAUTH_TOKEN_ENCRYPTION_KEY")
encryptor := security.NewSecretEncryptor(encryptionKey)
infraFactory := security.NewOAuthProviderFactory(encryptor)

// Create adapter
providerFactory := services.NewOAuthProviderFactoryAdapter(
    infraFactory,
    googleCfg,
    githubCfg,
)

// Create command handlers
authenticateHandler := commands.NewAuthenticateWithOAuthHandler(
    userRepo,
    oauthAccountRepo,
    providerFactory,
    jwtService,
    refreshService,
    sessionStore,
    logger,
)

linkHandler := commands.NewLinkOAuthAccountHandler(
    userRepo,
    oauthAccountRepo,
    providerFactory,
    logger,
)

unlinkHandler := commands.NewUnlinkOAuthAccountHandler(
    userRepo,
    oauthAccountRepo,
    logger,
)

// Create query handler
listAccountsHandler := queries.NewListOAuthAccountsHandler(
    oauthAccountRepo,
    logger,
)

// Create OAuth HTTP handler
oauthHandler := handlers.NewOAuthHandler(
    authenticateHandler,
    linkHandler,
    unlinkHandler,
    listAccountsHandler,
    providerFactory,
    redisClient,
    *logger,
)
```

## Testing Requirements

### Unit Tests

Create the following test files:

1. **Query Handler Tests**: `internal/application/identity/queries/list_oauth_accounts_test.go`
   - Test successful listing
   - Test empty list
   - Test invalid user ID
   - Test repository errors

2. **Handler Tests**: `internal/interfaces/http/handlers/oauth_handler_test.go`
   - Test OAuth initiation (state generation, Redis storage, redirect)
   - Test callback flow (state validation, authentication)
   - Test link flow (state validation, linking)
   - Test unlink flow (provider validation, unlinking, last auth method check)
   - Test list accounts (query delegation)
   - Test error mapping
   - Test CSRF state expiration

### Integration Tests (E2E)

Add to Postman collection (`tests/e2e/postman/goimg-api.postman_collection.json`):

1. **OAuth Authentication Flow**
   - Initiate OAuth (mock redirect)
   - Mock callback with code
   - Verify JWT tokens returned
   - Verify user created/linked

2. **OAuth Account Linking**
   - Login with password
   - Link OAuth account
   - Verify account linked
   - Attempt duplicate link (should fail)

3. **OAuth Account Management**
   - List linked accounts
   - Unlink OAuth account
   - Verify cannot unlink last auth method

4. **CSRF Protection**
   - Test expired state (should fail)
   - Test invalid state (should fail)
   - Test state reuse (should fail)

## Security Checklist

- [x] CSRF state is cryptographically random (32 bytes)
- [x] CSRF state stored in Redis with short TTL (10 minutes)
- [x] CSRF state is one-time use (deleted after validation)
- [x] State tied to specific OAuth provider
- [x] OAuth tokens encrypted before storage (handled by command layer)
- [x] Cannot link OAuth account to different user
- [x] Cannot unlink last authentication method
- [x] IP and User-Agent captured for auditing
- [x] Error responses don't leak internal details
- [x] All paths validated and sanitized

## OpenAPI Spec Updates

Update `api/openapi/openapi.yaml` with OAuth endpoints:

```yaml
paths:
  /auth/oauth/{provider}:
    get:
      summary: Initiate OAuth flow
      parameters:
        - name: provider
          in: path
          required: true
          schema:
            type: string
            enum: [google, github]
      responses:
        '302':
          description: Redirect to OAuth provider
        '400':
          $ref: '#/components/responses/BadRequest'
        '500':
          $ref: '#/components/responses/InternalServerError'

  /auth/oauth/{provider}/callback:
    get:
      summary: Handle OAuth callback
      parameters:
        - name: provider
          in: path
          required: true
          schema:
            type: string
            enum: [google, github]
        - name: code
          in: query
          required: true
          schema:
            type: string
        - name: state
          in: query
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Authentication successful
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthResponse'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'

  /auth/oauth/link:
    post:
      summary: Link OAuth account
      security:
        - bearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [provider, code, state]
              properties:
                provider:
                  type: string
                  enum: [google, github]
                code:
                  type: string
                state:
                  type: string
      responses:
        '200':
          description: Account linked successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/OAuthAccount'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '409':
          $ref: '#/components/responses/Conflict'

  /auth/oauth/{provider}:
    delete:
      summary: Unlink OAuth account
      security:
        - bearerAuth: []
      parameters:
        - name: provider
          in: path
          required: true
          schema:
            type: string
            enum: [google, github]
      responses:
        '200':
          description: Account unlinked successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Message'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '404':
          $ref: '#/components/responses/NotFound'

  /auth/oauth/accounts:
    get:
      summary: List linked OAuth accounts
      security:
        - bearerAuth: []
      responses:
        '200':
          description: List of linked accounts
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/OAuthAccountList'
        '401':
          $ref: '#/components/responses/Unauthorized'
```

## Next Steps

1. **Create OAuth Provider Adapter** (highest priority)
   - Implement `services.OAuthProviderFactoryAdapter`
   - Load configs from environment
   - Wire up in dependency injection

2. **Update Server Initialization**
   - Add OAuth handler to router
   - Inject all dependencies
   - Configure Redis client access

3. **Add Unit Tests**
   - Test query handler
   - Test HTTP handler endpoints
   - Test error mapping

4. **Add E2E Tests**
   - Add Postman collection requests
   - Test complete OAuth flows
   - Test security boundaries

5. **Update OpenAPI Spec**
   - Add OAuth endpoint definitions
   - Add request/response schemas
   - Validate with `make validate-openapi`

6. **Frontend Integration**
   - Add OAuth buttons to login page
   - Handle OAuth redirect flow
   - Store tokens securely

## Files Reference

### Created Files
- `/home/user/goimg-datalayer/internal/application/identity/queries/list_oauth_accounts.go` (2.2K)
- `/home/user/goimg-datalayer/internal/interfaces/http/handlers/oauth_handler.go` (19K)

### Existing Dependencies
- Domain: `/home/user/goimg-datalayer/internal/domain/identity/oauth.go`
- Infrastructure: `/home/user/goimg-datalayer/internal/infrastructure/security/oauth_provider.go`
- Commands:
  - `/home/user/goimg-datalayer/internal/application/identity/commands/oauth_authenticate.go`
  - `/home/user/goimg-datalayer/internal/application/identity/commands/oauth_link.go`
  - `/home/user/goimg-datalayer/internal/application/identity/commands/oauth_unlink.go`
- DTOs: `/home/user/goimg-datalayer/internal/application/identity/dto/dto.go`

## Notes

- Handler follows existing patterns from `auth_handler.go` and `twofa_handler.go`
- CSRF protection uses Redis for state management (secure, distributed)
- Error mapping provides consistent RFC 7807 responses
- All operations are audited with IP and User-Agent
- Security-first design prevents common OAuth vulnerabilities
