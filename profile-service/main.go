package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	SessionID string `json:"session_id"` // For session tracking
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func main() {
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
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("profile_session", store))

	// Public routes
	router.GET("/", handleHome)
	router.GET("/auth/callback", handleCallback)
	router.GET("/logout", handleLogout)

	// Protected routes
	protected := router.Group("/")
	protected.Use(authMiddleware())
	{
		protected.GET("/dashboard", handleDashboard)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("ProfileService starting on port %s", port)
	router.Run(":" + port)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		sessionID := session.Get("session_id")

		log.Printf("authMiddleware: userID=%v, sessionID=%v, path=%s", userID, sessionID, c.Request.URL.Path)

		// Get URLs (for browser redirects)
		authServiceURL := os.Getenv("AUTH_SERVICE_URL")
		if authServiceURL == "" {
			authServiceURL = "http://localhost:8080"
		}

		serviceURL := os.Getenv("SERVICE_URL")
		if serviceURL == "" {
			serviceURL = "http://localhost:8081"
		}

		if userID == nil {
			// Not authenticated, redirect to AuthService
			redirectURI := serviceURL + "/auth/callback"
			authURL := authServiceURL + "/auth?redirect_uri=" + url.QueryEscape(redirectURI)

			c.Redirect(http.StatusFound, authURL)
			c.Abort()
			return
		}

		// Validate session with AuthService
		if sessionID != nil {
			// Use internal URL for server-to-server communication
			authServiceInternalURL := os.Getenv("AUTH_SERVICE_INTERNAL_URL")
			if authServiceInternalURL == "" {
				authServiceInternalURL = authServiceURL // Fallback to public URL
			}

			// Check if session is still valid
			validateURL := authServiceInternalURL + "/session/validate?session_id=" + sessionID.(string)
			resp, err := http.Get(validateURL)
			if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
				if err != nil {
					log.Printf("Session validation error: %v (URL: %s)", err, validateURL)
				} else {
					log.Printf("Session validation failed: HTTP %d (session_id: %s)", resp.StatusCode, sessionID.(string))
				}

				// Session invalid, clear local session and redirect to login
				session.Clear()
				session.Save()

				redirectURI := serviceURL + "/auth/callback"
				authURL := authServiceURL + "/auth?redirect_uri=" + url.QueryEscape(redirectURI)

				c.Redirect(http.StatusFound, authURL)
				c.Abort()
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
		}

		c.Next()
	}
}

func handleHome(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID != nil {
		c.HTML(http.StatusOK, "home.html", gin.H{
			"Authenticated": true,
			"Email":         session.Get("email"),
			"FullName":      session.Get("full_name"),
		})
		return
	}

	c.HTML(http.StatusOK, "home.html", gin.H{
		"Authenticated": false,
	})
}

func handleCallback(c *gin.Context) {
	token := c.Query("token")
	refreshToken := c.Query("refresh_token")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	// Validate JWT
	claims, err := validateJWT(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "details": err.Error()})
		return
	}

	// Create local session
	session := sessions.Default(c)
	session.Set("user_id", claims.UserID)
	session.Set("email", claims.Email)
	session.Set("full_name", claims.FullName)
	session.Set("session_id", claims.SessionID)

	// Store refresh token if provided (for future token renewal)
	if refreshToken != "" {
		session.Set("refresh_token", refreshToken)
	}

	session.Save()

	// Redirect to dashboard
	c.Redirect(http.StatusFound, "/dashboard")
}

func handleDashboard(c *gin.Context) {
	session := sessions.Default(c)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Email":    session.Get("email"),
		"FullName": session.Get("full_name"),
		"UserID":   session.Get("user_id"),
	})
}

func handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func validateJWT(tokenString string) (*Claims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret"
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check expiration
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, jwt.ErrTokenExpired
		}
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}
