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
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
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

		if userID == nil {
			// Not authenticated, redirect to AuthService
			authServiceURL := os.Getenv("AUTH_SERVICE_URL")
			if authServiceURL == "" {
				authServiceURL = "http://localhost:8080"
			}

			serviceURL := os.Getenv("SERVICE_URL")
			if serviceURL == "" {
				serviceURL = "http://localhost:8081"
			}

			redirectURI := serviceURL + "/auth/callback"
			authURL := authServiceURL + "/auth?redirect_uri=" + url.QueryEscape(redirectURI)

			c.Redirect(http.StatusFound, authURL)
			c.Abort()
			return
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
