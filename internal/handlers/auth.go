package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"ticket-sys/internal/database"
	"ticket-sys/internal/models"
	"ticket-sys/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	db        *database.Database
	jwtSecret []byte
	// Add token expiration configuration
	tokenExpiration        time.Duration
	refreshTokenExpiration time.Duration
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *database.Database, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		db:                     db,
		jwtSecret:              jwtSecret,
		tokenExpiration:        15 * time.Minute, // Default 15 minutes
		refreshTokenExpiration: 24 * time.Hour,
	}
}

// Register handles user registration
// @Summary Create New User
// @Description Create a new user
// @ID create-user
// @Produce json
// @Success 200 "Successful response"
// @Failure 400 "Invalid input format"
// @Router /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var user models.UserRegister

	// Validate input JSON
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input format",
			"details": err.Error(),
		})
		return
	}

	// Additional validation
	if err := user.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var exists bool
	err := h.db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_user WHERE email = $1)",
		user.Email).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password processing failed"})
		return
	}

	// Insert user with transaction
	tx, err := h.db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
		return
	}

	var id int
	err = tx.QueryRow(`
        INSERT INTO staff_user 
        VALUES (DEFAULT, $1, $2, $3, $4) 
        RETURNING id`,
		user.FirstName, user.LastName, user.Email, hashedPassword,
	).Scan(&id)

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User creation failed"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user_id": id,
	})
}

// Get Users
// @Summary Get All Users
// @Description Get All Users
// @ID get-users
// @Produce json
// @Success 200 "Successful response"
// @Failure 400 "Database error"
// @Router /users [get]
func (h *AuthHandler) GetUsers(c *gin.Context) {
	var users []models.User

	rows, err := h.db.DB.Query("SELECT * FROM staff_user")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	for rows.Next() {
		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.Password,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Process failed"})
			return
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User iteration failed"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, users)
}

// Get User
// @Summary Get Single User
// @Description Get Single User
// @ID get-user
// @Produce json
// @Success 200 "Successful response"
// @Failure 400 "Database error"
// @Router /users [get]
func (h *AuthHandler) GetUser(c *gin.Context) {
	var user models.User

	idStr := c.Param("id")         // Retrieve the path parameter "id" as a string
	id, err := strconv.Atoi(idStr) // Convert it to an integer

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}

	dbErr := h.db.DB.QueryRow(`
        SELECT * 
        FROM staff_user 
        WHERE id = $1`,
		id,
	).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password)

	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, user)
}

// Login handles user authentication and JWT generation
// @Summary Login User
// @Description Log a user in
// @ID login-user
// @Produce json
// @Param user_login body models.UserLogin true "User Login"
// @Success 200 "Successful response"
// @Failure 400 "Invalid login data"
// @Failure 401 "Unauthorized user"
// @Router /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var login models.UserLogin
	if err := c.ShouldBindJSON(&login); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login data"})
		return
	}

	// Get user from database
	var user models.User
	err := h.db.DB.QueryRow(`
        SELECT id, first_name, last_name, password
        FROM staff_user 
        WHERE email = $1`,
		login.Email,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Password)

	if err == sql.ErrNoRows {
		// Don't specify whether email or password was wrong
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login process failed"})
		return
	}

	// Verify password
	if !utils.CheckPasswordHash(login.Password, user.Password) {
		// Use same message as above for security
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT with claims
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":    user.ID,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"iat":        now.Unix(),
		"exp":        now.Add(h.tokenExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	refreshClaims := jwt.MapClaims{
		"user_id":    user.ID,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"iat":        now.Unix(),
		"exp":        now.Add(h.refreshTokenExpiration).Unix(),
	}

	refreshTokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(h.jwtSecret)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Refresh Token generation failed"})
		return
	}

	// Return token with expiration
	c.JSON(http.StatusOK, gin.H{
		"token":                tokenString,
		"refresh_token":        refreshTokenString,
		"expires_in":           h.tokenExpiration.Seconds(),
		"refersh_token_expiry": h.refreshTokenExpiration.Seconds(),
		"token_type":           "Bearer",
	})
}

// RefreshToken generates a new token for valid users
// @Summary Refresh a Token
// @Description Refresh token for re-authentication
// @ID refresh-token
// @Produce json
// @Success 200 "Successful response"
// @Failure 400 "Token refresh failed"
// @Failure 401 "Unauthorized user"
// @Router /refresh-token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	// userID, exists := c.Get("user_id")
	// if !exists {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
	// 	return
	// }

	var refreshTokenString models.Token
	if err := c.ShouldBindJSON(&refreshTokenString); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Refresh Token"})
		return
	}

	refreshClaims := jwt.MapClaims{}
	refreshToken, err := jwt.ParseWithClaims(refreshTokenString.RefreshToken, refreshClaims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Claims"})
	}

	// Generate new token
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":    refreshToken.Claims.(jwt.MapClaims)["user_id"],
		"first_name": refreshToken.Claims.(jwt.MapClaims)["first_name"],
		"last_name":  refreshToken.Claims.(jwt.MapClaims)["last_name"],
		"iat":        now.Unix(),
		"exp":        now.Add(h.tokenExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token refresh failed"})
		return
	}

	newRefreshClaims := jwt.MapClaims{
		"user_id":    refreshToken.Claims.(jwt.MapClaims)["user_id"],
		"first_name": refreshToken.Claims.(jwt.MapClaims)["first_name"],
		"last_name":  refreshToken.Claims.(jwt.MapClaims)["last_name"],
		"iat":        now.Unix(),
		"exp":        now.Add(h.refreshTokenExpiration).Unix(),
	}

	newRefreshToken, tokenErr := jwt.NewWithClaims(jwt.SigningMethodHS256, newRefreshClaims).SignedString(h.jwtSecret)

	if tokenErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Refresh Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":                tokenString,
		"refresh_token":        newRefreshToken,
		"expires_in":           h.tokenExpiration.Seconds(),
		"refersh_token_expiry": h.refreshTokenExpiration.Seconds(),
		"token_type":           "Bearer",
	})
}

// Logout endpoint (optional - useful for client-side cleanup)
func (h *AuthHandler) Logout(c *gin.Context) {
	// Since JWT is stateless, server-side logout isn't needed
	// However, we can return instructions for the client
	c.JSON(http.StatusOK, gin.H{
		"message":      "Successfully logged out",
		"instructions": "Please remove the token from your client storage",
	})
}
