package main

import (
	"fmt"
	"net/http"
	"os"

	_ "github.com/ITU-BeeHub/BeeHub-backend/docs"
	auth "github.com/ITU-BeeHub/BeeHub-backend/internal/auth"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
	gin "github.com/gin-gonic/gin"
	godotenv "github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Loads environment variables from a .env file.
func loadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

// MessageResponse represents a JSON response with a message
type MessageResponse struct {
	Message string `json:"message"`
}

// @title BeeHub Ders Seçim Botu API
// @version 1.0
// @description Bu, BeeHub Ders Seçim Botu için API dokümantasyonudur.

// @host localhost:8080
// @BasePath /
func main() {

	personManager := pkg.NewPersonManager()

	person := &models.Person{}
	personManager.UpdatePerson(person)
	loadEnvVariables()

	r := gin.Default()

	// Swagger handler
	// if SWAGGER_ENABLED=true in .env, enable swagger
	if os.Getenv("SWAGGER_ENABLED") == "true" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		fmt.Println("Swagger is disabled")

	}
	r.GET("/hello", hello)

	authService := auth.NewService(personManager)
	authHandler := auth.NewHandler(authService)

	// Auth routes
	r.POST("/auth/login", authHandler.LoginHandler)

	// Protected routes
	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware(authService))
	{
		protected.GET("/auth/profile", authHandler.ProfileHandler)
	}
	r.Run(":8080")
}

// @Tags Hello
// @Summary Hello World
// @Description Hello World
// @Accept json
// @Produce json
// @Success 200 {object} MessageResponse
// @Router /hello [get]
func hello(c *gin.Context) {
	c.JSON(http.StatusOK, MessageResponse{
		Message: "hello world",
	})

}
