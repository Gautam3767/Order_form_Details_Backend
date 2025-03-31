package main

import (
	"log"
	"net/http"
	"os" // Import os

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // Import godotenv

	// --- Use YOUR actual module paths here ---
	// Make sure these paths match your go.mod file and project structure
	"github.com/Gautam3767/Order_form_Details_Backend.git/database"
	"github.com/Gautam3767/Order_form_Details_Backend.git/handlers"
	// -----------------------------------------
	// Add swagger imports if using swaggo
	// _ "github.com/Gautam3767/Order_form_Details_Backend.git/docs" // Adjust if using swagger docs
	// ginSwagger "github.com/swaggo/gin-swagger"
	// swaggerFiles "github.com/swaggo/files"
)

// Optional: Add Swagger annotations if you plan to generate API docs
// @title Brand Information Service API (MongoDB)
// @version 1.0
// @description This service manages brand information for the order form, using MongoDB.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080 // Default host, adjust if needed
// @BasePath /api/v1
func main() {
	// Load .env file first.
	// It's safe to ignore the error if the file is optional (e.g., in production using real env vars)
	err := godotenv.Load()
	if err != nil {
		log.Printf("Info: No .env file found or error loading it: %v. Relying on system environment variables.", err)
	}

	// Connect to Database (MongoDB implementation in database package)
	database.Connect()

	// Optional: Setup graceful shutdown to disconnect DB if needed
	// (More complex setup involving signal handling)
	// defer database.Disconnect() // Simple defer might not always run on abrupt termination

	// Initialize Gin Router
	router := gin.Default() // Includes Logger and Recovery middleware

	// --- CORS Middleware ---
	// Configure allowed origins based on your frontend URLs
	// Include both your main order form app and the admin UI
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		"http://localhost:3000", // Default React dev port for order form?
		"http://localhost:3001", // Default React dev port for admin UI?
		// Add your production frontend URLs here
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"} // Added common headers
	corsConfig.AllowCredentials = true                                                                                            // If you need cookies/sessions

	router.Use(cors.New(corsConfig))

	// --- API Routes ---
	// Group API endpoints under a versioned path
	api := router.Group("/api/v1")
	{
		// Group routes related to brands
		brandRoutes := api.Group("/brands")
		{
			brandRoutes.GET("", handlers.ListBrands)                   // Get list of brand names
			brandRoutes.POST("", handlers.CreateBrandManual)           // Create brand via JSON
			brandRoutes.GET("/:brandName", handlers.GetBrandDetails)   // Get details for one brand
			brandRoutes.PUT("/:brandName", handlers.UpdateBrandManual) // Update brand details via JSON
			brandRoutes.POST("/upload", handlers.UploadBrandPDF)       // Create/Update brand via PDF upload
			brandRoutes.DELETE("/:brandName", handlers.DeleteBrand)    // Delete a brand
		}
		// Add other resource routes here if needed (e.g., /api/v1/users)
	}

	// --- Swagger Route (Optional) ---
	// Uncomment if you have set up swaggo (`swag init` in your project root)
	// swaggerURL := ginSwagger.URL("/swagger/doc.json") // Point to generated JSON
	// router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))
	// log.Println("Swagger UI available at /swagger/index.html")

	// --- Health Check Endpoint ---
	// Basic health check to see if the service is running
	router.GET("/health", func(c *gin.Context) {
		// Consider adding a DB ping here for a more comprehensive check
		// Example (requires adapting database package to expose client or ping method):
		// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		// defer cancel()
		// if err := database.PingDB(ctx); err != nil { // Assuming PingDB exists
		//     c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "details": "database unreachable"})
		//     return
		// }
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// --- Start Server ---
	// Get port from environment variable or use a default
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080" // Default port if not specified
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Server starting and listening on http://localhost:%s", port)
	// router.Run() blocks until the server is stopped or an error occurs
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err) // Use Fatalf to exit on server start error
	}
}
