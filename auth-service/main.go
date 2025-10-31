package main

import (
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Claims struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	SessionID string `json:"session_id"` // For session tracking
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func main() {
	// Initialize database connection
	initDB()
	defer db.Close()

	// Setup Gin router
	router := gin.Default()

	// Load HTML templates
	router.LoadHTMLGlob("templates/*")

	// Setup session store
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "default-secret-key"
	}
	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24, // 24 hours
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("auth_session", store))

	// Routes
	router.GET("/", handleHome)
	router.GET("/login", handleLoginPage)
	router.POST("/login", handleLogin)
	router.GET("/register", handleRegisterPage)
	router.POST("/register", handleRegister)
	router.GET("/auth", handleAuth)
	router.GET("/logout", handleLogout)

	// Advanced features
	router.POST("/logout/all", handleLogoutAll)            // Global logout
	router.POST("/token/refresh", handleRefreshToken)      // Refresh token endpoint
	router.GET("/session/validate", handleValidateSession) // Session validation for SPs

	// Keycloak SSO routes
	router.GET("/auth/keycloak/login", handleKeycloakLogin)
	router.GET("/auth/keycloak/callback", handleKeycloakCallback)
	router.GET("/auth/keycloak/userinfo", handleKeycloakUserInfo)

	// Initialize Keycloak OIDC (if enabled)
	if err := initOIDC(); err != nil {
		log.Printf("⚠️  Warning: Failed to initialize OIDC: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("AuthService starting on port %s", port)
	router.Run(":" + port)
}

func initDB() {
	var err error
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Retry connection
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("Successfully connected to database")
				return
			}
		}
		log.Printf("Failed to connect to database, retrying... (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatal("Could not connect to database:", err)
}

func handleHome(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID != nil {
		var user User
		err := db.QueryRow("SELECT id, email, full_name FROM users WHERE id = $1", userID).
			Scan(&user.ID, &user.Email, &user.FullName)
		if err == nil {
			c.HTML(http.StatusOK, "home.html", gin.H{
				"Authenticated": true,
				"User":          user,
			})
			return
		}
	}

	c.HTML(http.StatusOK, "home.html", gin.H{
		"Authenticated": false,
	})
}

func handleLoginPage(c *gin.Context) {
	redirectURI := c.Query("redirect_uri")
	c.HTML(http.StatusOK, "login.html", gin.H{
		"RedirectURI": redirectURI,
		"Error":       "",
	})
}

func handleLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	redirectURI := c.PostForm("redirect_uri")

	if email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Error":       "Email and password are required",
			"RedirectURI": redirectURI,
		})
		return
	}

	var user User
	err := db.QueryRow("SELECT id, email, password_hash, full_name FROM users WHERE email = $1", email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName)

	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error":       "Invalid credentials",
			"RedirectURI": redirectURI,
		})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error":       "Invalid credentials",
			"RedirectURI": redirectURI,
		})
		return
	}

	// Generate session ID for tracking
	sessionID, err := generateRandomToken(32)
	if err != nil {
		log.Printf("Failed to generate session ID: %v", err)
		sessionID = fmt.Sprintf("session_%d_%d", user.ID, time.Now().Unix())
	}

	// Create active session record in database
	_, err = db.Exec(`
		INSERT INTO active_sessions (user_id, session_id, expires_at)
		VALUES ($1, $2, $3)`,
		user.ID, sessionID, time.Now().Add(24*time.Hour))
	if err != nil {
		log.Printf("Failed to create session record: %v", err)
	}

	// Create session
	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("email", user.Email)
	session.Set("full_name", user.FullName)
	session.Set("session_id", sessionID)
	session.Save()

	// If redirect_uri is provided, generate JWT and redirect
	if redirectURI != "" {
		token, refreshToken, err := generateTokenPair(user, sessionID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "login.html", gin.H{
				"Error":       "Failed to generate token",
				"RedirectURI": redirectURI,
			})
			return
		}
		// Include refresh token in redirect (SPs can store it for future use)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?token=%s&refresh_token=%s", redirectURI, token, refreshToken))
		return
	}

	// Otherwise redirect to home
	c.Redirect(http.StatusFound, "/")
}

func handleRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Error": "",
	})
}

func handleRegister(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	fullName := c.PostForm("full_name")

	if email == "" || password == "" || fullName == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": "All fields are required",
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Error": "Failed to process registration",
		})
		return
	}

	// Insert user
	_, err = db.Exec("INSERT INTO users (email, password_hash, full_name) VALUES ($1, $2, $3)",
		email, string(hashedPassword), fullName)

	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": "Email already exists",
		})
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func handleAuth(c *gin.Context) {
	redirectURI := c.Query("redirect_uri")

	if redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "redirect_uri is required"})
		return
	}

	// Check if user is already logged in
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID == nil {
		// Not logged in, redirect to login page
		c.Redirect(http.StatusFound, fmt.Sprintf("/login?redirect_uri=%s", redirectURI))
		return
	}

	// User is logged in, generate JWT and redirect back
	var user User
	sessionIDValue := session.Get("session_id")
	sessionIDStr := ""
	if sessionIDValue != nil {
		sessionIDStr = sessionIDValue.(string)
	}

	err := db.QueryRow("SELECT id, email, full_name FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Email, &user.FullName)

	if err != nil {
		// Session is invalid, clear it and redirect to login
		session.Clear()
		session.Save()
		c.Redirect(http.StatusFound, fmt.Sprintf("/login?redirect_uri=%s", redirectURI))
		return
	}

	// Check if session is still valid (not invalidated by logout/all)
	if sessionIDStr != "" {
		var invalidated bool
		err := db.QueryRow(`
			SELECT invalidated FROM active_sessions 
			WHERE session_id = $1`, sessionIDStr).Scan(&invalidated)

		if err != nil || invalidated {
			// Session has been invalidated, require re-login
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, fmt.Sprintf("/login?redirect_uri=%s", redirectURI))
			return
		}
	}

	token, refreshToken, err := generateTokenPair(user, sessionIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("%s?token=%s&refresh_token=%s", redirectURI, token, refreshToken))
}

func handleLogout(c *gin.Context) {
	session := sessions.Default(c)

	// Check if user logged in via Keycloak
	loginMethod := session.Get("login_method")
	isKeycloakLogin := loginMethod != nil && loginMethod.(string) == "keycloak"

	// Get ID token BEFORE clearing session (needed for Keycloak logout)
	var idTokenStr string
	if isKeycloakLogin {
		idToken := session.Get("id_token")
		if idToken != nil {
			idTokenStr = idToken.(string)
		}
	}

	// Get user ID to invalidate all their sessions
	userID := session.Get("user_id")
	if userID != nil {
		// Invalidate all active sessions for this user
		_, err := db.Exec(`
			UPDATE active_sessions 
			SET invalidated = true 
			WHERE user_id = $1`,
			userID)

		if err != nil {
			log.Printf("Error invalidating sessions for user %v: %v", userID, err)
		}

		// Revoke all refresh tokens for this user
		_, err = db.Exec(`
			UPDATE refresh_tokens 
			SET revoked = true 
			WHERE user_id = $1`,
			userID)

		if err != nil {
			log.Printf("Error revoking refresh tokens for user %v: %v", userID, err)
		}
	}

	// Clear local session
	session.Clear()
	session.Save()

	// If user logged in via Keycloak, redirect to Keycloak logout
	if isKeycloakLogin {
		keycloakBrowserURL := os.Getenv("KEYCLOAK_BROWSER_URL")
		if keycloakBrowserURL == "" {
			keycloakBrowserURL = "http://localhost:8090"
		}
		realm := os.Getenv("KEYCLOAK_REALM")
		if realm == "" {
			realm = "micro-dashboard-sso"
		}

		// Keycloak logout URL with post_logout_redirect_uri and id_token_hint (OIDC standard parameters)
		// Note: The redirect URI must be registered in Keycloak client's "Valid post logout redirect URIs"
		logoutURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout?post_logout_redirect_uri=%s&id_token_hint=%s",
			keycloakBrowserURL,
			realm,
			"http://localhost:8080/",
			idTokenStr)

		log.Printf("🔓 Redirecting to Keycloak logout with ID token")
		c.Redirect(http.StatusFound, logoutURL)
		return
	}

	// Regular logout - redirect to home
	c.Redirect(http.StatusFound, "/")
}

func generateJWT(user User) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret"
	}

	claims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ========== ADVANCED FEATURES ==========

// generateRandomToken generates a cryptographically secure random token
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateTokenPair creates both access token (short-lived) and refresh token (long-lived)
func generateTokenPair(user User, sessionID string) (accessToken string, refreshToken string, err error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret"
	}

	// Generate short-lived access token (15 minutes)
	accessClaims := Claims{
		UserID:    user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		SessionID: sessionID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	// Generate long-lived refresh token (7 days)
	refreshTokenString, err := generateRandomToken(32)
	if err != nil {
		return "", "", err
	}

	// Store refresh token in database
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)`,
		user.ID, refreshTokenString, expiresAt)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenString, nil
}

// handleLogoutAll invalidates all sessions for the current user
func handleLogoutAll(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Invalidate all active sessions for this user
	_, err := db.Exec(`
		UPDATE active_sessions 
		SET invalidated = TRUE 
		WHERE user_id = $1 AND invalidated = FALSE`,
		userID)
	if err != nil {
		log.Printf("Failed to invalidate sessions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate sessions"})
		return
	}

	// Revoke all refresh tokens for this user
	_, err = db.Exec(`
		UPDATE refresh_tokens 
		SET revoked = TRUE 
		WHERE user_id = $1 AND revoked = FALSE`,
		userID)
	if err != nil {
		log.Printf("Failed to revoke refresh tokens: %v", err)
	}

	// Clear current session
	session.Clear()
	session.Save()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out from all devices successfully",
	})
}

// handleRefreshToken exchanges a refresh token for a new access token
func handleRefreshToken(c *gin.Context) {
	refreshToken := c.PostForm("refresh_token")
	if refreshToken == "" {
		refreshToken = c.GetHeader("X-Refresh-Token")
	}

	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token is required"})
		return
	}

	// Verify refresh token in database
	var userID int
	var email, fullName string
	var expiresAt time.Time
	var revoked bool

	err := db.QueryRow(`
		SELECT rt.user_id, u.email, u.full_name, rt.expires_at, rt.revoked
		FROM refresh_tokens rt
		JOIN users u ON rt.user_id = u.id
		WHERE rt.token = $1`,
		refreshToken).Scan(&userID, &email, &fullName, &expiresAt, &revoked)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Check if token is revoked or expired
	if revoked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has been revoked"})
		return
	}

	if time.Now().After(expiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has expired"})
		return
	}

	// Generate new access token
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret"
	}

	accessClaims := Claims{
		UserID:    userID,
		Email:     email,
		FullName:  fullName,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
		TokenType:    "Bearer",
	})
}

// handleValidateSession validates if a session is still active
func handleValidateSession(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	var invalidated bool
	var expiresAt time.Time

	err := db.QueryRow(`
		SELECT invalidated, expires_at 
		FROM active_sessions 
		WHERE session_id = $1`,
		sessionID).Scan(&invalidated, &expiresAt)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "Session not found",
		})
		return
	}

	if invalidated || time.Now().After(expiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "Session has been invalidated or expired",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"expires_at": expiresAt,
	})
}
