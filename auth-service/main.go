package main

import (
	"database/sql"
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
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	jwt.RegisteredClaims
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

	// Create session
	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("email", user.Email)
	session.Set("full_name", user.FullName)
	session.Save()

	// If redirect_uri is provided, generate JWT and redirect
	if redirectURI != "" {
		token, err := generateJWT(user)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "login.html", gin.H{
				"Error":       "Failed to generate token",
				"RedirectURI": redirectURI,
			})
			return
		}
		c.Redirect(http.StatusFound, fmt.Sprintf("%s?token=%s", redirectURI, token))
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
	err := db.QueryRow("SELECT id, email, full_name FROM users WHERE id = $1", userID).
		Scan(&user.ID, &user.Email, &user.FullName)

	if err != nil {
		// Session is invalid, clear it and redirect to login
		session.Clear()
		session.Save()
		c.Redirect(http.StatusFound, fmt.Sprintf("/login?redirect_uri=%s", redirectURI))
		return
	}

	token, err := generateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("%s?token=%s", redirectURI, token))
}

func handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
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
