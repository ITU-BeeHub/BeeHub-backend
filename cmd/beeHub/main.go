package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	_ "github.com/ITU-BeeHub/BeeHub-backend/docs"
	auth "github.com/ITU-BeeHub/BeeHub-backend/internal/auth"

	beepicker "github.com/ITU-BeeHub/BeeHub-backend/internal/beePicker"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
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

	personManager := pkg.NewPersonManager()

	person := &models.Person{}
	personManager.UpdatePerson(person)
	utils.LoadEnvVariables()

	utils.LoadEnvVariables()

	r := gin.Default()

	// Swagger handler
	// if SWAGGER_ENABLED=true in .env, enable swagger
	if os.Getenv("SWAGGER_ENABLED") == "true" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		fmt.Println("Swagger is disabled")

	}

	authService := auth.NewService(personManager)
	authHandler := auth.NewHandler(authService)

	r.POST("/auth/login", authHandler.LoginHandler)

	// beePicker routes
	r.GET("/beePicker/courses", beepicker.CourseHandler)
	r.GET("/beePicker/schedule", beepicker.ScheduleHandler)
	r.POST("/beePicker/schedule", beepicker.ScheduleSaveHandler)

	// Protected routes
	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware(authService))
	{
		protected.GET("/auth/profile", authHandler.ProfileHandler)
	}

	r.GET("/start-service", startService)
	r.GET("/stop-service", stopService)
	r.Run(":8080")
}

// @Summary Start the BeeHubBot process
// @Description Starts the BeeHubBot process as a background process
// @Tags Service
// @Success 200 {object} map[string]string "Process started"
// @Failure 500 {object} map[string]string "Error starting process"
// @Router /start-service [get]
// startService sets the service startup type to automatic and starts the service.
func startService(c *gin.Context) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "runasadmin.bat")
		err := cmd.Run()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error starting service: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Service started and set to automatic"})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unsupported OS"})
	}
}

// @Summary Stop the BeeHubBot process
// @Description Stops the BeeHubBot process
// @Tags Service
// @Success 200 {object} map[string]string "Process stopped"
// @Failure 500 {object} map[string]string "Error stopping process"
// @Router /stop-service [get]
// stopService stops the service and sets the startup type to manual.
func stopService(c *gin.Context) {
	// Stop the service
	cmd := exec.Command("sc.exe", "stop", "BeeHubBotService")
	err := cmd.Run()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error stopping service: %v", err)})
		return
	}

	// Change startup type to manual
	cmd = exec.Command("sc.exe", "config", "BeeHubBotService", "start=", "demand")
	err = cmd.Run()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error setting service to manual: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service stopped and set to manual"})
}
