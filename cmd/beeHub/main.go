package main

import (
	"fmt"
	"net/http"
	"os"

	_ "github.com/ITU-BeeHub/BeeHub-backend/docs"
	auth "github.com/ITU-BeeHub/BeeHub-backend/internal/auth"
	beepicker "github.com/ITU-BeeHub/BeeHub-backend/internal/beepicker"
	utils "github.com/ITU-BeeHub/BeeHub-backend/pkg/utils"

	gin "github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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
	utils.LoadEnvVariables()

	r := gin.Default()

	// Swagger handler
	// if SWAGGER_ENABLED=true in .env, enable swagger
	if os.Getenv("SWAGGER_ENABLED") == "true" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		fmt.Println("Swagger is disabled")

	}

	// Example route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, MessageResponse{
			Message: "hello world",
		})
	})

	// Auth routes
	r.GET("/auth/login", auth.LoginHandler)

	// Course routes
	r.GET("/beePicker/courses", beepicker.CourseHandler)

	r.GET("/hello", hello)

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
