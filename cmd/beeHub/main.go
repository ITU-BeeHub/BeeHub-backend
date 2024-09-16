package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	cors "github.com/gin-contrib/cors"
	gin "github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Struct to parse the response from the beehubapp.com/version endpoint
type VersionResponse struct {
	DownloadURL string `json:"download_url"`
	Version     string `json:"version"`
}

// Fallback backend version in case fetching from beehubapp.com fails
var BackendVersion = "Alpha"

// MessageResponse represents a JSON response with a message
type MessageResponse struct {
	Message string `json:"message"`
}

// Function to fetch the latest backend version from beehubapp.com
func fetchBackendVersion() {
	resp, err := http.Get("https://beehubapp.com/api/version")
	if err != nil {
		fmt.Printf("Failed to fetch backend version: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Received non-OK HTTP status: %d\n", resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return
	}

	var versionResp VersionResponse
	if err := json.Unmarshal(body, &versionResp); err != nil {
		fmt.Printf("Failed to parse version response: %v\n", err)
		return
	}

	// Set the backend version
	BackendVersion = versionResp.Version
	fmt.Printf("Fetched backend version: %s\n", BackendVersion)
}

func main() {
	// Fetch the backend version on startup
	fetchBackendVersion()

	personManager := pkg.NewPersonManager()
	person := &models.Person{}
	personManager.UpdatePerson(person)
	utils.LoadEnvVariables()

	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Adjust this to your frontend's URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours
	}))

	// Version check endpoint
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": BackendVersion})
	})

	// Swagger handler
	if os.Getenv("SWAGGER_ENABLED") == "true" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		fmt.Println("Swagger is disabled")
	}

	authService := auth.NewService(personManager)
	authHandler := auth.NewHandler(authService)

	r.POST("/auth/login", authHandler.LoginHandler)

	// beePicker routes
	beePickerService := beepicker.NewService(personManager)
	beePickerHandler := beepicker.NewHandler(beePickerService)

	r.GET("/beePicker/courses", beePickerHandler.CourseHandler)

	// Protected routes
	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware(authService))
	{
		protected.GET("/auth/profile", authHandler.ProfileHandler)
		protected.POST("/beePicker/pick", beePickerHandler.PickHandler)
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
		cmd := exec.Command("cmd", "/C", "startasadmin.bat")
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
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "stopasadmin.bat")
		err := cmd.Run()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error stopping service: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Service stopped and set to manual"})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unsupported OS"})
	}
}
