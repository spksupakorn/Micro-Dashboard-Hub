# 🔑 Keycloak SSO Setup Guide - Complete Tutorial

## 📖 What is Keycloak?

**Keycloak** is an open-source Identity and Access Management (IAM) solution that provides:
- **Single Sign-On (SSO)** - Login once, access multiple applications
- **Social Login** - Login with Google, Facebook, GitHub, etc.
- **User Management** - Centralized user database
- **Two-Factor Authentication (2FA)**
- **OIDC & SAML Support** - Industry-standard protocols

Think of Keycloak as a **professional replacement** for your custom AuthService that big companies use!

---

## 🚀 Step-by-Step Setup Guide

### Step 1: Access Keycloak Admin Console

Your Keycloak is already running! Access it:

```bash
# Open in your browser
http://localhost:8090
```

**Login Credentials:**
- Username: `admin`
- Password: `admin`

![Keycloak Login](https://www.keycloak.org/resources/images/screen-login.png)

---

### Step 2: Create a New Realm

A **Realm** is like a "workspace" that contains users, clients (applications), and settings.

1. **Click dropdown** at top-left (currently shows "master")
2. **Click "Create Realm"** button
3. **Enter Realm name**: `micro-dashboard-sso`
4. **Click "Create"**

✅ You now have a dedicated realm for your SSO system!

**Why separate realm?** 
- `master` realm = Keycloak admin management
- `micro-dashboard-sso` = Your application users

---

### Step 3: Create a Client (Application)

A **Client** represents your application (AuthService) that will use Keycloak for authentication.

1. **Click "Clients"** in left sidebar
2. **Click "Create client"** button
3. **Fill in the form:**

   ```
   Client type: OpenID Connect
   Client ID: auth-service
   Name: Auth Service
   Description: Main authentication service
   ```

4. **Click "Next"**

5. **Configure Capability:**
   ```
   Client authentication: ON
   Authorization: OFF
   Authentication flow:
     ☑ Standard flow
     ☑ Direct access grants
   ```

6. **Click "Next"**

7. **Configure Login Settings:**
   ```
   Root URL: http://localhost:8080
   Home URL: http://localhost:8080
   Valid redirect URIs: 
     http://localhost:8080/*
     http://localhost:8081/*
     http://localhost:8082/*
   Valid post logout redirect URIs: 
     http://localhost:8080/*
   Web origins: 
     http://localhost:8080
     http://localhost:8081
     http://localhost:8082
   ```

8. **Click "Save"**

✅ Your AuthService is now registered with Keycloak!

---

### Step 4: Get Client Secret

This secret key allows your AuthService to communicate with Keycloak.

1. **In Clients list**, click on `auth-service`
2. **Click "Credentials"** tab
3. **Copy the "Client secret"** value
   - Example: `a1b2c3d4-e5f6-7890-abcd-ef1234567890`

📝 **Keep this secret safe!** You'll need it in Step 7.

---

### Step 5: Create Test Users

Let's create users that will login through Keycloak.

#### User 1: John Doe (Regular User)

1. **Click "Users"** in left sidebar
2. **Click "Create new user"** button
3. **Fill in:**
   ```
   Username: john.doe
   Email: john@example.com
   Email verified: ON
   First name: John
   Last name: Doe
   ```
4. **Click "Create"**

5. **Set Password:**
   - Click "Credentials" tab
   - Click "Set password"
   - Password: `password123`
   - Temporary: OFF (so user doesn't need to change it)
   - Click "Save"

#### User 2: Jane Admin (Admin User)

Repeat the same steps:
```
Username: jane.admin
Email: jane@example.com
Email verified: ON
First name: Jane
Last name: Admin
Password: password123
```

✅ You now have test users!

---

### Step 6: Update Environment Configuration

Update your `.env` file with Keycloak settings:

```bash
# Open .env file
nano .env
```

**Add these lines** at the end:

```env
# Keycloak Configuration
KEYCLOAK_URL=http://keycloak:8090
KEYCLOAK_REALM=micro-dashboard-sso
KEYCLOAK_CLIENT_ID=auth-service
KEYCLOAK_CLIENT_SECRET=<your-client-secret-from-step-4>
OIDC_ENABLED=true
```

**Replace** `<your-client-secret-from-step-4>` with your actual client secret!

**Save and exit** (Ctrl+X, then Y, then Enter)

---

### Step 7: Install OIDC Library in AuthService

We need to add the OIDC library to enable Keycloak integration.

```bash
cd auth-service

# Install go-oidc library
go get github.com/coreos/go-oidc/v3/oidc
go get golang.org/x/oauth2

# Update dependencies
go mod tidy

cd ..
```

---

### Step 8: Add Keycloak Routes to AuthService

I'll create a new file with Keycloak integration code:

```bash
# Create keycloak integration file
touch auth-service/keycloak.go
```

Now let me add the code for you:

```go
// File: auth-service/keycloak.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/coreos/go-oidc/v3/oidc"
    "github.com/gin-contrib/sessions"
    "github.com/gin-gonic/gin"
    "golang.org/x/oauth2"
)

var (
    oidcProvider *oidc.Provider
    oauth2Config *oauth2.Config
    oidcVerifier *oidc.IDTokenVerifier
)

// initOIDC initializes Keycloak OIDC provider
func initOIDC() error {
    oidcEnabled := os.Getenv("OIDC_ENABLED")
    if oidcEnabled != "true" {
        log.Println("OIDC is disabled. Set OIDC_ENABLED=true to enable Keycloak integration.")
        return nil
    }

    keycloakURL := os.Getenv("KEYCLOAK_URL")
    realm := os.Getenv("KEYCLOAK_REALM")
    
    if keycloakURL == "" || realm == "" {
        log.Println("OIDC configuration missing. Skipping Keycloak integration.")
        return nil
    }

    ctx := context.Background()
    issuerURL := keycloakURL + "/realms/" + realm

    provider, err := oidc.NewProvider(ctx, issuerURL)
    if err != nil {
        log.Printf("Failed to create OIDC provider: %v", err)
        return err
    }

    oidcProvider = provider
    
    clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
    clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")
    
    oauth2Config = &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        RedirectURL:  "http://localhost:8080/auth/keycloak/callback",
        Endpoint:     provider.Endpoint(),
        Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
    }

    oidcVerifier = provider.Verifier(&oidc.Config{ClientID: clientID})

    log.Println("✅ Keycloak OIDC provider initialized successfully!")
    log.Printf("   Issuer: %s", issuerURL)
    log.Printf("   Client ID: %s", clientID)
    
    return nil
}

// handleKeycloakLogin redirects user to Keycloak login page
func handleKeycloakLogin(c *gin.Context) {
    if oauth2Config == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "error": "Keycloak is not configured",
        })
        return
    }

    // Generate random state for CSRF protection
    state := generateRandomToken()
    
    // Store state in session
    session := sessions.Default(c)
    session.Set("oauth_state", state)
    session.Set("oauth_timestamp", time.Now().Unix())
    session.Save()

    // Redirect to Keycloak
    authURL := oauth2Config.AuthCodeURL(state)
    c.Redirect(http.StatusFound, authURL)
}

// handleKeycloakCallback handles the callback from Keycloak
func handleKeycloakCallback(c *gin.Context) {
    session := sessions.Default(c)
    
    // Verify state (CSRF protection)
    state := c.Query("state")
    savedState := session.Get("oauth_state")
    
    if savedState == nil || state != savedState.(string) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
        return
    }

    // Check state timestamp (prevent replay attacks)
    timestamp := session.Get("oauth_timestamp")
    if timestamp != nil {
        if time.Now().Unix()-timestamp.(int64) > 600 { // 10 minutes
            c.JSON(http.StatusBadRequest, gin.H{"error": "State expired"})
            return
        }
    }

    // Exchange authorization code for tokens
    code := c.Query("code")
    ctx := context.Background()
    
    oauth2Token, err := oauth2Config.Exchange(ctx, code)
    if err != nil {
        log.Printf("Failed to exchange token: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
        return
    }

    // Extract ID Token
    rawIDToken, ok := oauth2Token.Extra("id_token").(string)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "No id_token in token response"})
        return
    }

    // Verify ID Token
    idToken, err := oidcVerifier.Verify(ctx, rawIDToken)
    if err != nil {
        log.Printf("Failed to verify ID token: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
        return
    }

    // Extract claims
    var claims struct {
        Email         string `json:"email"`
        EmailVerified bool   `json:"email_verified"`
        Name          string `json:"name"`
        GivenName     string `json:"given_name"`
        FamilyName    string `json:"family_name"`
        Sub           string `json:"sub"` // Keycloak user ID
    }

    if err := idToken.Claims(&claims); err != nil {
        log.Printf("Failed to parse claims: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse claims"})
        return
    }

    log.Printf("Keycloak user logged in: %s (%s)", claims.Name, claims.Email)

    // Check if user exists in local database
    var user User
    err = db.QueryRow(`
        SELECT id, email, full_name 
        FROM users 
        WHERE email = $1`,
        claims.Email,
    ).Scan(&user.ID, &user.Email, &user.FullName)

    // If user doesn't exist, create them
    if err != nil {
        log.Printf("Creating new user from Keycloak: %s", claims.Email)
        
        err = db.QueryRow(`
            INSERT INTO users (email, full_name, password_hash) 
            VALUES ($1, $2, $3) 
            RETURNING id, email, full_name`,
            claims.Email,
            claims.Name,
            "keycloak-sso", // Placeholder, they login via Keycloak
        ).Scan(&user.ID, &user.Email, &user.FullName)

        if err != nil {
            log.Printf("Failed to create user: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
            return
        }
    }

    // Create session in database
    sessionID := generateRandomToken()
    expiresAt := time.Now().Add(24 * time.Hour)

    _, err = db.Exec(`
        INSERT INTO active_sessions (user_id, session_id, expires_at) 
        VALUES ($1, $2, $3)`,
        user.ID, sessionID, expiresAt)

    if err != nil {
        log.Printf("Failed to create session: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
        return
    }

    // Create local session
    session.Clear()
    session.Set("user_id", user.ID)
    session.Set("email", user.Email)
    session.Set("full_name", user.FullName)
    session.Set("session_id", sessionID)
    session.Set("login_method", "keycloak")
    session.Save()

    log.Printf("✅ User %s logged in via Keycloak successfully!", user.Email)

    // Check if there's a redirect URI
    redirectURI := c.Query("redirect_uri")
    if redirectURI != "" {
        // Generate JWT for SSO
        token, err := generateJWT(user)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
            return
        }

        // Generate refresh token
        refreshToken := generateRandomToken()
        refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)

        _, err = db.Exec(`
            INSERT INTO refresh_tokens (user_id, token, expires_at) 
            VALUES ($1, $2, $3)`,
            user.ID, refreshToken, refreshExpiresAt)

        if err != nil {
            log.Printf("Failed to create refresh token: %v", err)
        }

        // Redirect to service with token
        c.Redirect(http.StatusFound, redirectURI+"?token="+token+"&refresh_token="+refreshToken)
        return
    }

    // Default: redirect to home
    c.Redirect(http.StatusFound, "/")
}

// handleKeycloakUserInfo returns current user info (for debugging)
func handleKeycloakUserInfo(c *gin.Context) {
    session := sessions.Default(c)
    
    userID := session.Get("user_id")
    if userID == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "user_id":      session.Get("user_id"),
        "email":        session.Get("email"),
        "full_name":    session.Get("full_name"),
        "session_id":   session.Get("session_id"),
        "login_method": session.Get("login_method"),
    })
}
```

---

### Step 9: Update AuthService main.go

We need to register the new Keycloak routes.

Open `auth-service/main.go` and find the `main()` function. Add the OIDC initialization and routes:

**Add after line where routes are defined:**

```go
// Initialize Keycloak OIDC (if enabled)
if err := initOIDC(); err != nil {
    log.Printf("Warning: Failed to initialize OIDC: %v", err)
}

// Keycloak SSO routes
router.GET("/auth/keycloak/login", handleKeycloakLogin)
router.GET("/auth/keycloak/callback", handleKeycloakCallback)
router.GET("/auth/keycloak/userinfo", handleKeycloakUserInfo)
```

---

### Step 10: Rebuild and Restart Services

```bash
# Rebuild auth service with Keycloak support
docker-compose up -d --build auth_service

# Check if it's running
docker-compose ps
```

You should see all services running, including `sso_keycloak`.

---

### Step 11: Test Keycloak SSO!

#### Option 1: Direct Keycloak Login

Open your browser:

```
http://localhost:8080/auth/keycloak/login
```

**What happens:**
1. Redirects to Keycloak login page
2. Login with: `john.doe` / `password123`
3. Keycloak authenticates user
4. Redirects back to your AuthService
5. Creates local session
6. ✅ You're logged in!

#### Option 2: SSO Flow with Services

1. **Open ProfileService**: http://localhost:8081/dashboard
2. **You'll be redirected to AuthService**: http://localhost:8080/auth?redirect_uri=...
3. **Modify URL to use Keycloak**:
   ```
   http://localhost:8080/auth/keycloak/login?redirect_uri=http://localhost:8081/auth/callback
   ```
4. **Login with Keycloak**: `john.doe` / `password123`
5. **You'll be redirected back to ProfileService** with SSO token
6. ✅ ProfileService dashboard shows your info!

---

### Step 12: Add Keycloak Login Button to UI

Update your login page template to show Keycloak option.

**Edit `auth-service/templates/login.html`:**

Find the login form and add this button **above or below** the existing login button:

```html
<!-- Existing login form -->
<form method="POST" action="/login">
    <!-- ... existing fields ... -->
    <button type="submit">Login with Email</button>
</form>

<!-- NEW: Keycloak SSO Button -->
<div style="margin-top: 20px; text-align: center;">
    <p style="color: #666;">- OR -</p>
    <a href="/auth/keycloak/login" 
       style="display: inline-block; padding: 12px 24px; background: #4285f4; 
              color: white; text-decoration: none; border-radius: 4px; 
              font-weight: bold;">
        🔐 Login with Keycloak SSO
    </a>
</div>
```

Rebuild:
```bash
docker-compose up -d --build auth_service
```

Now your login page has both options!

---

## 🎯 Testing Checklist

- [ ] Access Keycloak Admin Console: http://localhost:8090
- [ ] Created realm: `micro-dashboard-sso`
- [ ] Created client: `auth-service`
- [ ] Created test user: `john.doe`
- [ ] Updated `.env` with Keycloak settings
- [ ] Installed OIDC library: `go get github.com/coreos/go-oidc/v3/oidc`
- [ ] Created `keycloak.go` file
- [ ] Updated `main.go` with Keycloak routes
- [ ] Rebuilt AuthService: `docker-compose up -d --build auth_service`
- [ ] Test direct login: http://localhost:8080/auth/keycloak/login
- [ ] Test SSO flow with ProfileService
- [ ] Updated login template with Keycloak button

---

## 🔍 How It Works - Visual Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    KEYCLOAK SSO FLOW                        │
└─────────────────────────────────────────────────────────────┘

1. User clicks "Login with Keycloak"
   │
   ↓
2. AuthService → Redirect to Keycloak
   │   URL: http://keycloak:8090/realms/micro-dashboard-sso/protocol/openid-connect/auth
   │
   ↓
3. User enters credentials in Keycloak
   │   Username: john.doe
   │   Password: password123
   │
   ↓
4. Keycloak validates credentials
   │   ✓ User exists in Keycloak database
   │   ✓ Password correct
   │
   ↓
5. Keycloak → Redirect to AuthService with code
   │   URL: http://localhost:8080/auth/keycloak/callback?code=abc123&state=xyz
   │
   ↓
6. AuthService exchanges code for tokens
   │   POST to Keycloak: Exchange authorization code
   │   Response: { id_token, access_token, refresh_token }
   │
   ↓
7. AuthService verifies ID token
   │   Validates signature with Keycloak's public key
   │   Extracts user info: { email, name, sub }
   │
   ↓
8. AuthService creates/updates local user
   │   Check if user exists in local database
   │   If not, create new user with Keycloak info
   │
   ↓
9. AuthService creates session
   │   Store session in active_sessions table
   │   Set session cookie
   │
   ↓
10. User redirected to requested service
    │   ProfileService receives JWT token
    │   ProfileService validates token
    │   User sees dashboard ✅
```

---

## 💡 Benefits of Using Keycloak

### Before (Custom AuthService):
- ❌ Manual user management
- ❌ No social login
- ❌ No 2FA
- ❌ Manual password resets
- ❌ Limited audit logging

### After (With Keycloak):
- ✅ Professional user management UI
- ✅ Social login (Google, GitHub, Facebook, etc.)
- ✅ Two-Factor Authentication (TOTP, SMS)
- ✅ Self-service password reset
- ✅ Comprehensive audit logs
- ✅ User federation (LDAP, Active Directory)
- ✅ Role-based access control (RBAC)
- ✅ Industry-standard security

---

## 🔐 Advanced Features You Can Enable

### 1. Social Login (Google)

**In Keycloak Admin Console:**
1. Go to **Identity Providers**
2. Click "Add provider" → Select "Google"
3. Get Google OAuth credentials from: https://console.cloud.google.com
4. Enter Client ID and Client Secret
5. Save

Now users can login with "Login with Google"!

### 2. Two-Factor Authentication (2FA)

**In Keycloak Admin Console:**
1. Go to **Authentication**
2. Click "Required Actions"
3. Enable "Configure OTP"
4. Go to "Flows" → "Browser"
5. Add "OTP Form" to the flow

Users will be prompted to set up 2FA!

### 3. Custom Login Theme

**Make Keycloak match your brand:**
1. Create custom theme directory
2. Modify CSS, logos, colors
3. Deploy theme to Keycloak
4. Select theme in Realm Settings

### 4. User Federation (LDAP/AD)

**Connect to existing user directories:**
1. Go to **User Federation**
2. Click "Add provider" → "LDAP"
3. Enter LDAP server details
4. Map LDAP attributes to Keycloak attributes
5. Save and sync users

---

## 🐛 Troubleshooting

### Problem: "Connection refused" when accessing Keycloak

**Solution:**
```bash
# Check if Keycloak is running
docker-compose ps

# If not running, start it
docker-compose up -d keycloak

# Check logs
docker logs sso_keycloak
```

### Problem: "Client not found" error

**Solution:**
- Verify Client ID in Keycloak matches `.env` file
- Check realm name is correct
- Ensure redirect URIs are configured

### Problem: "Invalid redirect URI"

**Solution:**
- Add your redirect URI to Keycloak client settings
- Format: `http://localhost:8080/*`
- Must include `/*` wildcard

### Problem: OIDC provider initialization fails

**Solution:**
```bash
# Check if Keycloak is accessible from inside Docker
docker exec auth_service ping keycloak

# Check environment variables
docker exec auth_service env | grep KEYCLOAK
```

---

## 📚 Learn More

- **Keycloak Documentation**: https://www.keycloak.org/documentation
- **OpenID Connect Explained**: https://openid.net/connect/
- **OAuth 2.0 Guide**: https://oauth.net/2/

---

## 🎉 Congratulations!

You now have a **professional-grade SSO system** with Keycloak! Your users can:
- ✅ Login with username/password (Keycloak)
- ✅ Login with social accounts (if configured)
- ✅ Use 2FA for extra security
- ✅ Self-service password reset
- ✅ Single Sign-On across all your services

**Next steps:**
- Configure social login providers
- Enable 2FA
- Customize Keycloak theme
- Add more applications as clients
- Set up user roles and permissions

---

**Need help?** Check the logs:
```bash
# Keycloak logs
docker logs sso_keycloak -f

# AuthService logs  
docker logs auth_service -f
```

Happy coding! 🚀
