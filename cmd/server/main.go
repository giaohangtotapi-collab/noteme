package main

import (
	"log"
	"noteme/internal/api"
	"noteme/internal/config"
	"noteme/internal/db"
	"noteme/internal/repository"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode (default to release mode)
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database if DATABASE_URL is provided
	if cfg.DatabaseURL != "" {
		log.Printf("Initializing database connection with DATABASE_URL...")
		if err := db.Init(); err != nil {
			log.Printf("Warning: Failed to initialize database: %v. Continuing without database.", err)
		} else {
			// Initialize repository
			log.Printf("Creating PostgreSQL repository...")
			repo := repository.NewPostgresRepository()
			if repo == nil {
				log.Printf("Error: Failed to create repository")
			} else {
				api.InitSTTRepository(repo)
				log.Println("Database and repository initialized successfully")
			}
		}
	} else {
		log.Println("DATABASE_URL not set, running without database (in-memory storage only)")
	}

	r := gin.Default()

	// Add CORS middleware for mobile app
	r.Use(corsMiddleware())

	// Register routes
	api.RegisterRoutes(r)

	log.Printf("NoteMe backend running on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// corsMiddleware adds CORS headers for mobile app
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
