package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	models "github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService *Service
}

func NewHandler(authService *Service) *Handler {
	return &Handler{authService: authService}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware(authService *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader != "" {
			token := authHeader
			lower := strings.ToLower(authHeader)
			if strings.HasPrefix(lower, "bearer ") {
				token = strings.TrimSpace(authHeader[len("Bearer "):])
			}
			if token != "" {
				authService.personManager.UpdateToken(token)
				authService.personManager.UpdateLoginTime()
				c.Next()
				return
			}
		}

		person := authService.personManager.GetPerson()
		// Token kontrolünü email kontrolünden ayırıyoruz
		if person.Token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// @Tags Auth
// @Summary Login and get token
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]string "token"
// @Failure 400 {object} map[string]string "validation error"
// @Failure 401 {object} map[string]string "login failed"
// @Failure 502 {object} map[string]string "kepler service unavailable"
// @Failure 500 {object} map[string]string "internal server error"
// @Router /auth/login [post]
func (h *Handler) LoginHandler(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear any existing person data before new login
	h.authService.personManager.UpdatePerson(&models.Person{})

	// Önce mevcut person verilerini temizle
	h.authService.personManager.ClearPerson()

	token, err := h.authService.LoginService(req.Email, req.Password)
	if err != nil {
		switch err.Error() {
		case "login failed":
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login failed"})

		case "bad status":
			c.JSON(http.StatusBadGateway, gin.H{"error": "kepler service unavailable"})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{"token": token})

	}

}

// @Tags Auth
// @Summary Get authenticated user profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.PersonDTO
// @Failure 401 {object} map[string]string "unauthenticated"
// @Failure 404 {object} map[string]string "profile not found"
// @Failure 500 {object} map[string]string "internal server error"
// @Router /auth/profile [get]
func (h *Handler) ProfileHandler(c *gin.Context) {
	person := h.authService.personManager.GetPerson()

	// Token kontrolü
	if person.Token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	// Token yenilemesi gerekiyorsa (kimlik bilgileri mevcutsa)
	if person.Email != "" && person.Password != "" && time.Since(person.LoginTime).Hours() >= 4 {
		token, err := h.authService.LoginService(person.Email, person.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}
		person.Token = token
	}

	dto, err := h.authService.ProfileService(person)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Profile service error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Boş profil kontrolü
	if dto.First_name == "" && dto.Last_name == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error":        "profile not found",
			"email":        person.Email,
			"token_exists": person.Token != "",
		})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// @Tags Auth
// @Summary Logout user
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message"
// @Router /auth/logout [post]
func (h *Handler) LogoutHandler(c *gin.Context) {
	h.authService.LogoutService()
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
