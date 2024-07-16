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
	token, http_code, err := LoginService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token,
		"code": http_code})
}
