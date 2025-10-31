# Advanced Features Documentation

## 🚀 New Features Overview

This document covers three advanced features added to the Micro-Dashboard Hub:

1. **Global Logout** - Log out from all services simultaneously
2. **Refresh Tokens** - Short-lived access tokens with long-lived refresh tokens
3. **Keycloak Integration** - OIDC-compliant authentication with Keycloak

---

## 1. 🔐 Global Logout Feature

### Overview

The Global Logout feature allows users to invalidate all active sessions across all services with a single action. This is useful for:
- Security: If a user suspects their account has been compromised
- Privacy: Logging out from all devices when using shared/public computers
- Session management: Cleaning up old sessions

### How It Works

1. User triggers `/logout/all` endpoint on AuthService
2. AuthService marks all active sessions for that user as "invalidated" in the database
3. All refresh tokens for that user are revoked
4. Current session is cleared
5. When any SP tries to authenticate the user, it checks if the session is still valid
6. Invalidated sessions force re-authentication

### Database Schema

```sql
CREATE TABLE active_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    invalidated BOOLEAN DEFAULT FALSE
);
```

### API Endpoints

#### POST /logout/all

Logs out the user from all devices and services.

**Request:**
```bash
curl -X POST http://localhost:8080/logout/all \
  -H "Cookie: auth_session=<session_cookie>"
```

**Response:**
```json
{
  "success": true,
  "message": "Logged out from all devices successfully"
}
```

**Status Codes:**
- 200: Success
- 401: Not authenticated
- 500: Server error

#### GET /session/validate

Validates if a session ID is still active.

**Request:**
```bash
curl "http://localhost:8080/session/validate?session_id=<session_id>"
```

**Response (Valid):**
```json
{
  "valid": true,
  "expires_at": "2025-10-31T10:30:00Z"
}
```

**Response (Invalid):**
```json
{
  "valid": false,
  "error": "Session has been invalidated or expired"
}
```

### Usage Example

**Scenario: User wants to log out from all devices**

1. User goes to AuthService home page
2. Clicks "Logout from All Devices" button (or calls API)
3. All sessions invalidated
4. User is redirected to login page
5. Any other device trying to access services will be prompted to re-login

### Testing

```bash
# 1. Login from first browser/device
curl -X POST http://localhost:8080/login \
  -d "email=test@example.com&password=password123" \
  -c cookies1.txt

# 2. Access ProfileService (should work)
curl -b cookies1.txt http://localhost:8081/dashboard

# 3. Logout from all devices
curl -X POST http://localhost:8080/logout/all -b cookies1.txt

# 4. Try to access ProfileService again (should redirect to login)
curl -b cookies1.txt http://localhost:8081/dashboard
```

---

## 2. 🔄 Refresh Token System

### Overview

The Refresh Token system implements a more secure authentication flow:
- **Access Tokens**: Short-lived (15 minutes), used for API requests
- **Refresh Tokens**: Long-lived (7 days), used to obtain new access tokens

### Benefits

1. **Enhanced Security**: Access tokens expire quickly, limiting the damage if stolen
2. **Better UX**: Users don't need to re-login frequently
3. **Token Revocation**: Refresh tokens can be revoked for specific devices
4. **Compliance**: Follows OAuth 2.0 and OIDC best practices

### Token Lifecycle

```
Login → Access Token (15 min) + Refresh Token (7 days)
       ↓
Access Token Expires
       ↓
Use Refresh Token → New Access Token (15 min)
       ↓
Refresh Token Expires or Revoked
       ↓
Re-login Required
```

### Database Schema

```sql
CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN DEFAULT FALSE
);
```

### API Endpoints

#### POST /token/refresh

Exchanges a refresh token for a new access token.

**Request:**
```bash
curl -X POST http://localhost:8080/token/refresh \
  -d "refresh_token=<your_refresh_token>"
```

**Alternative (Header):**
```bash
curl -X POST http://localhost:8080/token/refresh \
  -H "X-Refresh-Token: <your_refresh_token>"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "same_refresh_token",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

**Status Codes:**
- 200: Success
- 400: Missing refresh token
- 401: Invalid, revoked, or expired refresh token
- 500: Server error

### Token Structure

**Access Token (JWT):**
```json
{
  "user_id": 1,
  "email": "test@example.com",
  "full_name": "Test User",
  "session_id": "abc123...",
  "token_type": "access",
  "exp": 1730333700,
  "iat": 1730332800,
  "sub": "1"
}
```

**Refresh Token:**
- Random 32-byte base64-encoded string
- Stored in database
- Not a JWT (cannot be decoded)

### Usage Flow

#### Initial Login

1. User logs in to AuthService
2. AuthService returns both tokens:
   - Access token in JWT format
   - Refresh token as opaque string
3. SP receives both tokens via redirect
4. SP stores refresh token in session/cookie

#### Token Refresh

1. Access token expires (after 15 minutes)
2. SP detects expired token
3. SP calls `/token/refresh` with refresh token
4. AuthService returns new access token
5. SP updates session with new access token

### Implementation in Service Providers

**Example: ProfileService with token refresh**

```go
func refreshAccessToken(refreshToken string) (string, error) {
    authServiceURL := os.Getenv("AUTH_SERVICE_URL")
    
    resp, err := http.PostForm(
        authServiceURL + "/token/refresh",
        url.Values{"refresh_token": {refreshToken}},
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return "", errors.New("refresh failed")
    }
    
    var result TokenResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return result.AccessToken, nil
}
```

### Testing

```bash
# 1. Login and capture tokens
curl -X POST http://localhost:8080/login \
  -d "email=test@example.com&password=password123&redirect_uri=http://localhost:8081/auth/callback" \
  -L -c cookies.txt -s | grep -o 'refresh_token=[^&"]*'

# Example output: refresh_token=abc123def456...

# 2. Wait 15 minutes (or modify code to expire in 1 minute)

# 3. Refresh the token
curl -X POST http://localhost:8080/token/refresh \
  -d "refresh_token=abc123def456..." \
  -H "Content-Type: application/x-www-form-urlencoded"

# 4. Response with new access token
```

---

## 3. 🔑 Keycloak Integration (OIDC)

### Overview

Keycloak is an open-source Identity and Access Management (IAM) solution that provides:
- OpenID Connect (OIDC) protocol support
- User Federation
- Social Login (Google, Facebook, etc.)
- Two-Factor Authentication
- Single Sign-On across multiple applications

### Architecture

```
User → Service Provider → AuthService → Keycloak
                              ↓
                        Database (fallback)
```

### Setup

#### 1. Start Keycloak

Keycloak is included in `docker-compose.yml` and starts automatically:

```bash
docker-compose up -d keycloak
```

**Access Keycloak Admin Console:**
- URL: http://localhost:8090
- Username: `admin`
- Password: `admin`

#### 2. Configure Keycloak Realm

1. Login to Keycloak Admin Console
2. Create a new Realm (e.g., "sso-demo")
3. Create a Client:
   - Client ID: `auth-service`
   - Client Protocol: `openid-connect`
   - Access Type: `confidential`
   - Valid Redirect URIs: `http://localhost:8080/*`
   - Web Origins: `http://localhost:8080`
4. Note the Client Secret from the Credentials tab

#### 3. Configure AuthService

Update environment variables:

```yaml
auth_service:
  environment:
    - KEYCLOAK_URL=http://keycloak:8090
    - KEYCLOAK_REALM=sso-demo
    - KEYCLOAK_CLIENT_ID=auth-service
    - KEYCLOAK_CLIENT_SECRET=<your_client_secret>
    - OIDC_ENABLED=true
```

### OIDC Flow

```
1. User clicks "Login with Keycloak"
   ↓
2. Redirect to Keycloak login page
   ↓
3. User enters credentials in Keycloak
   ↓
4. Keycloak authenticates user
   ↓
5. Redirect back to AuthService with authorization code
   ↓
6. AuthService exchanges code for tokens
   ↓
7. AuthService creates local session
   ↓
8. User redirected to requested service
```

### Integration with go-oidc

**Install dependency:**
```bash
go get github.com/coreos/go-oidc/v3/oidc
go get golang.org/x/oauth2
```

**Example implementation (add to auth-service/main.go):**

```go
import (
    "context"
    "github.com/coreos/go-oidc/v3/oidc"
    "golang.org/x/oauth2"
)

var (
    oidcProvider *oidc.Provider
    oauth2Config *oauth2.Config
)

func initOIDC() error {
    keycloakURL := os.Getenv("KEYCLOAK_URL")
    realm := os.Getenv("KEYCLOAK_REALM")
    
    if keycloakURL == "" || realm == "" {
        return nil // OIDC not configured
    }
    
    ctx := context.Background()
    issuerURL := fmt.Sprintf("%s/realms/%s", keycloakURL, realm)
    
    provider, err := oidc.NewProvider(ctx, issuerURL)
    if err != nil {
        return err
    }
    
    oidcProvider = provider
    oauth2Config = &oauth2.Config{
        ClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
        ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
        RedirectURL:  "http://localhost:8080/auth/callback/oidc",
        Endpoint:     provider.Endpoint(),
        Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
    }
    
    return nil
}

// Add these handlers
func handleOIDCLogin(c *gin.Context) {
    state := generateRandomString(16)
    c.SetCookie("oauth_state", state, 600, "/", "", false, true)
    
    authURL := oauth2Config.AuthCodeURL(state)
    c.Redirect(http.StatusFound, authURL)
}

func handleOIDCCallback(c *gin.Context) {
    state := c.Query("state")
    savedState, _ := c.Cookie("oauth_state")
    
    if state != savedState {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
        return
    }
    
    code := c.Query("code")
    ctx := context.Background()
    
    oauth2Token, err := oauth2Config.Exchange(ctx, code)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
        return
    }
    
    // Verify ID Token
    rawIDToken, ok := oauth2Token.Extra("id_token").(string)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "No id_token"})
        return
    }
    
    verifier := oidcProvider.Verifier(&oidc.Config{ClientID: oauth2Config.ClientID})
    idToken, err := verifier.Verify(ctx, rawIDToken)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
        return
    }
    
    // Extract claims
    var claims struct {
        Email string `json:"email"`
        Name  string `json:"name"`
        Sub   string `json:"sub"`
    }
    
    if err := idToken.Claims(&claims); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse claims"})
        return
    }
    
    // Create or update user in local database
    // ... create session, redirect to SP
}
```

### Testing Keycloak Integration

1. **Access Keycloak:**
   ```bash
   open http://localhost:8090
   ```

2. **Create Test User in Keycloak:**
   - Go to Users → Add User
   - Username: `keycloak_user`
   - Email: `keycloak@example.com`
   - Save
   - Go to Credentials tab
   - Set password: `password123`
   - Temporary: OFF

3. **Test OIDC Login:**
   ```bash
   # Access the OIDC login endpoint
   open "http://localhost:8080/auth/oidc/login"
   ```

4. **Verify:**
   - Should redirect to Keycloak login
   - Login with Keycloak credentials
   - Should redirect back to AuthService
   - Session should be created

### Keycloak Features

#### User Federation
- Connect to LDAP/Active Directory
- Import existing users
- Sync user data

#### Social Login
- Google
- Facebook
- GitHub
- Twitter
- And more...

#### Two-Factor Authentication
- TOTP (Time-based One-Time Password)
- SMS
- Email

#### Advanced Features
- User registration
- Password policies
- Account management
- Admin console
- Events and logging

### Best Practices

1. **Use HTTPS in Production**: Keycloak requires HTTPS for production
2. **Secure Client Secrets**: Use environment variables, never commit
3. **Token Validation**: Always verify tokens on the backend
4. **Refresh Tokens**: Implement token refresh for better UX
5. **Logout**: Implement both local and Keycloak logout
6. **Error Handling**: Handle all OAuth/OIDC errors gracefully

---

## 🔗 Integration Summary

### How Features Work Together

1. **Login with Refresh Tokens:**
   - User logs in → Gets access token (15 min) + refresh token (7 days)
   - Access token expires → Use refresh token to get new access token
   - No re-login needed for 7 days

2. **Global Logout:**
   - User triggers logout/all
   - All sessions invalidated
   - All refresh tokens revoked
   - User must re-login everywhere

3. **Keycloak Option:**
   - Can use traditional AuthService OR Keycloak
   - Keycloak provides enterprise features
   - Same SSO flow, different authentication backend

### Security Considerations

#### Access Tokens (JWT)
- ✅ Short-lived (15 minutes)
- ✅ Cannot be revoked (by design)
- ✅ Contain user information
- ⚠️ If stolen, only valid for 15 minutes

#### Refresh Tokens
- ✅ Long-lived (7 days)
- ✅ Can be revoked
- ✅ Stored in database
- ⚠️ More valuable if stolen - implement rotation

#### Session Tracking
- ✅ All sessions tracked in database
- ✅ Can be invalidated globally
- ✅ Expiration enforced
- ✅ Audit trail

### Performance Impact

- **Database Queries**: +2 queries per login (session + refresh token)
- **Token Validation**: Slightly slower (check session validity)
- **Refresh Token Flow**: +1 database query
- **Keycloak**: Additional network roundtrip

### Migration Path

#### From Current Implementation:

1. **Deploy with backward compatibility:**
   - Old flow still works
   - New features are additive

2. **Enable refresh tokens:**
   - Update SPs to store refresh tokens
   - Implement token refresh logic

3. **Enable global logout:**
   - Add UI button
   - Educate users

4. **Optional: Enable Keycloak:**
   - Deploy Keycloak
   - Configure realm and clients
   - Add OIDC login option
   - Keep traditional login as fallback

---

## 📝 Configuration Reference

### Environment Variables

```bash
# AuthService
JWT_SECRET=your-jwt-secret
SESSION_SECRET=your-session-secret
DB_HOST=postgres
DB_PORT=5432
DB_USER=sso_user
DB_PASSWORD=sso_password
DB_NAME=sso_db

# Keycloak (Optional)
KEYCLOAK_URL=http://localhost:8090
KEYCLOAK_REALM=sso-demo
KEYCLOAK_CLIENT_ID=auth-service
KEYCLOAK_CLIENT_SECRET=your-client-secret
OIDC_ENABLED=true
```

### Token Expiration

```go
// Configurable in code
const (
    ACCESS_TOKEN_DURATION  = 15 * time.Minute
    REFRESH_TOKEN_DURATION = 7 * 24 * time.Hour
    SESSION_DURATION       = 24 * time.Hour
)
```

---

## 🧪 Testing All Features

### Complete Test Script

```bash
#!/bin/bash

echo "=== Testing Advanced Features ==="

# 1. Test Login with Refresh Tokens
echo "\n1. Testing login with refresh tokens..."
RESPONSE=$(curl -s -X POST http://localhost:8080/login \
  -d "email=test@example.com&password=password123&redirect_uri=http://localhost:8081/auth/callback" \
  -c cookies.txt -L)
echo "Login successful"

# Extract refresh token
REFRESH_TOKEN=$(echo "$RESPONSE" | grep -o 'refresh_token=[^&"]*' | cut -d= -f2)
echo "Refresh token: ${REFRESH_TOKEN:0:20}..."

# 2. Test Token Refresh
echo "\n2. Testing token refresh..."
NEW_TOKEN=$(curl -s -X POST http://localhost:8080/token/refresh \
  -d "refresh_token=$REFRESH_TOKEN" | jq -r '.access_token')
echo "New access token obtained: ${NEW_TOKEN:0:20}..."

# 3. Test Session Validation
echo "\n3. Testing session validation..."
curl -s "http://localhost:8080/session/validate?session_id=test" | jq '.'

# 4. Test Global Logout
echo "\n4. Testing global logout..."
curl -s -X POST http://localhost:8080/logout/all -b cookies.txt | jq '.'

# 5. Verify Keycloak is running
echo "\n5. Checking Keycloak..."
curl -s http://localhost:8090 > /dev/null && echo "Keycloak is accessible" || echo "Keycloak not running"

echo "\n=== All tests complete ==="
```

---

## 📚 Additional Resources

- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [OpenID Connect Specification](https://openid.net/connect/)
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [go-oidc Library](https://github.com/coreos/go-oidc)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)

---

**Implementation Status:**
- ✅ Global Logout: Fully implemented
- ✅ Refresh Tokens: Fully implemented
- ✅ Keycloak Setup: Docker container ready
- ⚠️ OIDC Integration: Requires go-oidc library installation and code implementation

**Next Steps:**
1. Install go-oidc library in AuthService
2. Implement OIDC handlers
3. Add "Login with Keycloak" button to UI
4. Test complete OIDC flow
