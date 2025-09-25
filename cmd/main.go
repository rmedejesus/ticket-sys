package main

import (
	"log"
	"net/http"
	"ticket-sys/internal/config"
	"ticket-sys/internal/database"
	"ticket-sys/internal/handlers"
	"ticket-sys/internal/middleware"

	_ "ticket-sys/cmd/docs"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title SJ Ticketing API
// @version 1.0
// @description Ticketing API v1.0
// @host 192.168.1.57:8080
// @BasePath /api/v1
// @schemes http
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize database
	db, err := database.NewDatabase(cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.DB.Close()

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router with middleware
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Initialize handlers with JWT configuration
	authHandler := handlers.NewAuthHandler(db, []byte(cfg.JWT.Secret))

	// Public routes access
	public := r.Group("/api/v1")
	{
		public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
		public.POST("/refresh-token", authHandler.RefreshToken)
	}

	// Protected routes with JWT middleware
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware([]byte(cfg.JWT.Secret)))
	{

		protected.POST("/logout", authHandler.Logout)
		protected.GET("/profile", getUserProfile)
		protected.POST("/tickets", authHandler.CreateTicket)
		protected.GET("/tickets", authHandler.GetTickets)
		protected.GET("/tickets/:id", authHandler.GetTicket)
		protected.PATCH("/tickets/:id", authHandler.UpdateTicket)
		protected.PATCH("/tickets/:id/pending", authHandler.UpdatePendingTicket)
		protected.PATCH("/tickets/:id/completed", authHandler.UpdateCompletedTicket)
		protected.DELETE("/tickets/:id", authHandler.DeleteTicket)
		protected.GET("/users", authHandler.GetUsers)
		protected.GET("/users/:id", authHandler.GetUser)
	}

	// Swagger UI route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Start server with configured host and port
	serverAddr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s", serverAddr)

	//certFile := "onrender-cert.pem"     // or "server.crt" if using OpenSSL
	//keyFile := "192.168.1.57+1-key.pem" // or "server.key" if using OpenSSL

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed to start:", err)
	}
}

func getUserProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	firstName, _ := c.Get("first_name")
	lastName, _ := c.Get("last_name")

	c.JSON(http.StatusOK, gin.H{
		"id":         userID,
		"first_name": firstName,
		"last_name":  lastName,
	})
}
