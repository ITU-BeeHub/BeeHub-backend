package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/ITU-BeeHub/BeeHub-backend/docs"
	auth "github.com/ITU-BeeHub/BeeHub-backend/internal/auth"

	beepicker "github.com/ITU-BeeHub/BeeHub-backend/internal/beePicker"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg/config"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
	utils "github.com/ITU-BeeHub/BeeHub-backend/pkg/utils"

	cors "github.com/gin-contrib/cors"
	gin "github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// GitHub'dan versiyon bilgisi çekme URL'i
const (
	GitHubVersionURL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-backend/main/version.txt"
	MaxRetryAttempts = 5
	RetryDelay       = 5 * time.Second
)

// Fallback backend version - anlamlı değer, hata yönetimi için
var (
	BackendVersion     = config.DefaultVersion
	backendVersionLock sync.RWMutex
)

// MessageResponse represents a JSON response with a message
type MessageResponse struct {
	Message string `json:"message"`
}

// getBackendVersion thread-safe versiyon okuma
func getBackendVersion() string {
	backendVersionLock.RLock()
	defer backendVersionLock.RUnlock()
	return BackendVersion
}

// setBackendVersion thread-safe versiyon yazma
func setBackendVersion(version string) {
	backendVersionLock.Lock()
	defer backendVersionLock.Unlock()
	BackendVersion = version
}

// fetchBackendVersion GitHub'dan versiyon bilgisini çeker
// Non-blocking fetch with retry logic
func fetchBackendVersion() {
	go func() {
		retryCount := 0
		for retryCount < MaxRetryAttempts {
			fmt.Println("Attempting to fetch backend version from GitHub...")

			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			resp, err := client.Get(GitHubVersionURL)
			if err != nil {
				fmt.Printf("Failed to fetch backend version: %v\n", err)
				retryCount++
				time.Sleep(RetryDelay)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Received non-OK HTTP status: %d\n", resp.StatusCode)
				resp.Body.Close()
				retryCount++
				time.Sleep(RetryDelay)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				fmt.Printf("Failed to read response body: %v\n", err)
				retryCount++
				time.Sleep(RetryDelay)
				continue
			}

			version := strings.TrimSpace(string(body))
			if version == "" {
				fmt.Println("Received empty version string")
				retryCount++
				time.Sleep(RetryDelay)
				continue
			}

			// Successfully fetched the backend version
			setBackendVersion(version)
			fmt.Printf("Fetched backend version from GitHub: %s\n", version)
			return
		}

		fmt.Printf("Failed to fetch version after %d attempts, using fallback: %s\n",
			MaxRetryAttempts, config.DefaultVersion)
	}()
}

// @title BeeHub Ders Seçim Botu API
// @version 1.0
// @description BeeHub Ders Seçim Botu için API dokümantasyonu.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Authorization header with Bearer token. Example: "Bearer {token}"
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
		c.JSON(http.StatusOK, gin.H{"version": getBackendVersion()})
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
	r.POST("/auth/logout", authHandler.LogoutHandler)
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
