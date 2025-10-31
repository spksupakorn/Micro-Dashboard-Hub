package main

import (
	"context"
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
	oauth2Config *oauth2.Config
	oidcVerifier *oidc.IDTokenVerifier
)

// initOIDC initializes Keycloak OIDC provider
func initOIDC() error {
	oidcEnabled := os.Getenv("OIDC_ENABLED")
	if oidcEnabled != "true" {
		log.Println("ℹ️  OIDC is disabled. Set OIDC_ENABLED=true to enable Keycloak integration.")
		return nil
	}

	keycloakURL := os.Getenv("KEYCLOAK_URL")
	keycloakBrowserURL := os.Getenv("KEYCLOAK_BROWSER_URL")
	realm := os.Getenv("KEYCLOAK_REALM")

	if keycloakURL == "" || realm == "" {
		log.Println("⚠️  OIDC configuration missing. Skipping Keycloak integration.")
		return nil
	}

	// Use browser URL for fallback
	if keycloakBrowserURL == "" {
		keycloakBrowserURL = keycloakURL
	}

	ctx := context.Background()
	// Use internal URL for OIDC provider discovery (server-to-server)
	issuerURL := keycloakURL + "/realms/" + realm

	// Create a custom context that skips issuer validation
	// This is needed because Keycloak returns localhost:8090 as issuer
	// but we connect via keycloak:8090 from within Docker
	ctx = oidc.InsecureIssuerURLContext(ctx, issuerURL)

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		log.Printf("❌ Failed to create OIDC provider: %v", err)
		return err
	}

	clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")

	// Create OAuth2 config with browser-accessible endpoints
	oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8080/auth/keycloak/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  keycloakBrowserURL + "/realms/" + realm + "/protocol/openid-connect/auth",
			TokenURL: keycloakURL + "/realms/" + realm + "/protocol/openid-connect/token", // Use internal URL for server-to-server
		},
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// Create verifier with SkipIssuerCheck to handle issuer mismatch
	// (Docker internal URL vs browser-accessible URL)
	oidcVerifier = provider.Verifier(&oidc.Config{
		ClientID:          clientID,
		SkipIssuerCheck:   true, // Allow issuer mismatch between keycloak:8090 and localhost:8090
		SkipExpiryCheck:   false,
		SkipClientIDCheck: false,
	})

	log.Println("✅ Keycloak OIDC provider initialized successfully!")
	log.Printf("   Issuer: %s", issuerURL)
	log.Printf("   Client ID: %s", clientID)

	return nil
}

// handleKeycloakLogin redirects user to Keycloak login page
func handleKeycloakLogin(c *gin.Context) {
	if oauth2Config == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Keycloak is not configured. Please check OIDC_ENABLED and Keycloak settings in .env",
		})
		return
	}

	// Get redirect_uri from query params (for SSO flow)
	redirectURI := c.Query("redirect_uri")

	// Generate random state for CSRF protection
	state, err := generateRandomToken(32)
	if err != nil {
		log.Printf("❌ Failed to generate state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state and redirect_uri in session
	session := sessions.Default(c)
	session.Set("oauth_state", state)
	session.Set("oauth_timestamp", time.Now().Unix())
	if redirectURI != "" {
		session.Set("oauth_redirect_uri", redirectURI)
	}
	session.Save()

	// Redirect to Keycloak
	authURL := oauth2Config.AuthCodeURL(state)
	log.Printf("🔐 Redirecting to Keycloak login: %s", authURL)
	c.Redirect(http.StatusFound, authURL)
}

// handleKeycloakCallback handles the callback from Keycloak
func handleKeycloakCallback(c *gin.Context) {
	session := sessions.Default(c)

	// Verify state (CSRF protection)
	state := c.Query("state")
	savedState := session.Get("oauth_state")

	if savedState == nil || state != savedState.(string) {
		log.Printf("❌ Invalid state parameter: expected %v, got %s", savedState, state)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter (CSRF protection failed)"})
		return
	}

	// Check state timestamp (prevent replay attacks)
	timestamp := session.Get("oauth_timestamp")
	if timestamp != nil {
		if time.Now().Unix()-timestamp.(int64) > 600 { // 10 minutes
			log.Println("❌ State expired (older than 10 minutes)")
			c.JSON(http.StatusBadRequest, gin.H{"error": "State expired. Please try logging in again."})
			return
		}
	}

	// Exchange authorization code for tokens
	code := c.Query("code")
	ctx := context.Background()

	log.Println("🔄 Exchanging authorization code for tokens...")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		log.Printf("❌ Failed to exchange token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange authorization code"})
		return
	}

	// Extract ID Token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Println("❌ No id_token in token response")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No id_token in token response"})
		return
	}

	// Verify ID Token
	log.Println("🔍 Verifying ID token...")
	idToken, err := oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("❌ Failed to verify ID token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ID token"})
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
		log.Printf("❌ Failed to parse claims: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user claims"})
		return
	}

	log.Printf("✅ Keycloak user authenticated: %s (%s)", claims.Name, claims.Email)

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
		log.Printf("👤 Creating new user from Keycloak: %s", claims.Email)

		err = db.QueryRow(`
			INSERT INTO users (email, full_name, password_hash) 
			VALUES ($1, $2, $3) 
			RETURNING id, email, full_name`,
			claims.Email,
			claims.Name,
			"keycloak-sso", // Placeholder, they login via Keycloak
		).Scan(&user.ID, &user.Email, &user.FullName)

		if err != nil {
			log.Printf("❌ Failed to create user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user account"})
			return
		}
		log.Printf("✅ New user created: ID=%d, Email=%s", user.ID, user.Email)
	}

	// Create session in database
	sessionID, err := generateRandomToken(32)
	if err != nil {
		log.Printf("❌ Failed to generate session ID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session"})
		return
	}
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = db.Exec(`
		INSERT INTO active_sessions (user_id, session_id, expires_at) 
		VALUES ($1, $2, $3)`,
		user.ID, sessionID, expiresAt)

	if err != nil {
		log.Printf("❌ Failed to create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Create local session
	session.Set("user_id", user.ID)
	session.Set("email", user.Email)
	session.Set("full_name", user.FullName)
	session.Set("session_id", sessionID)
	session.Set("login_method", "keycloak")
	session.Set("id_token", rawIDToken) // Store ID token for logout

	// Clear OAuth state
	session.Delete("oauth_state")
	session.Delete("oauth_timestamp")

	session.Save()

	log.Printf("🎉 User %s logged in via Keycloak successfully! Session ID: %s", user.Email, sessionID[:20]+"...")

	// Check if there's a redirect URI (SSO flow)
	savedRedirectURI := session.Get("oauth_redirect_uri")
	if savedRedirectURI != nil {
		redirectURI := savedRedirectURI.(string)
		session.Delete("oauth_redirect_uri")
		session.Save()

		log.Printf("↪️  SSO flow: Redirecting to %s", redirectURI)

		// Generate JWT for SSO
		token, err := generateJWT(user)
		if err != nil {
			log.Printf("❌ Failed to generate JWT: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Generate refresh token
		refreshToken, err := generateRandomToken(32)
		if err != nil {
			log.Printf("⚠️  Failed to generate refresh token: %v", err)
		}
		refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)

		_, err = db.Exec(`
			INSERT INTO refresh_tokens (user_id, token, expires_at) 
			VALUES ($1, $2, $3)`,
			user.ID, refreshToken, refreshExpiresAt)

		if err != nil {
			log.Printf("⚠️  Failed to create refresh token: %v", err)
		}

		// Redirect to service with token
		c.Redirect(http.StatusFound, redirectURI+"?token="+token+"&refresh_token="+refreshToken)
		return
	}

	// Default: redirect to home
	log.Println("↪️  Redirecting to home page")
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
