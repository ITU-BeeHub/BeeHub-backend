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
	loginURL, err := LoginService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"login_url": loginURL})
}
