package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Tags Login
// @Summary Hello World
// @Accept json
// @Produce html
// @Router /auth/login [get]
func LoginHandler(c *gin.Context) {
	token, err := LoginService()
	if err != nil {
		switch err.Error() {
		case "login failed":
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login failed"})

		case "bad status":
			c.JSON(http.StatusBadGateway, gin.H{"error": "kepler service unavailable"})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
