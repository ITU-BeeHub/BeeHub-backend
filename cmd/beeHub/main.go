package main

import (
	"net/http"

	_ "github.com/ITU-BeeHub/BeeHub-backend/docs"
	auth "github.com/ITU-BeeHub/BeeHub-backend/internal/auth"
	"github.com/gin-gonic/gin"
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
	r := gin.Default()

	// Swagger handler
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Example route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, MessageResponse{
			Message: "hello world",
		})
	})

	// @Tags Auth
	// @Summary Login
	// @Description Login
	// @Accept json
	// @Produce json
	// @Success 200 {object} MessageResponse
	// @Router /login [post]
	r.GET("/login", auth.Login)

	// @Tags Auth
	// @Summary Register
	// @Description Register
	// @Accept json
	// @Produce json
	// @Success 200 {object} MessageResponse
	// @Router /register [post]
	r.GET("/register", auth.Register)

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
