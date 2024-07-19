package auth

import (
	"log"
	"net/http"
	"time"

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
		person := authService.personManager.GetPerson()
		if person.Email == "" {
			// User is not authenticated, redirect to login page
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// @Tags Login
// @Summary Hello World
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login credentials"
// @Router /auth/login [post]
func (h *Handler) LoginHandler(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

// @Tags Profile
// @Summary Hello World
// @Accept json
// @Produce json
// @Router /auth/profile [get]
func (h *Handler) ProfileHandler(c *gin.Context) {
	person := h.authService.personManager.GetPerson()

	if time.Since(person.LoginTime).Hours() >= 4 {
		h.authService.LoginService(person.Email, person.Password)
	}
	dto, err := h.authService.ProfileService(person)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, dto)
}
