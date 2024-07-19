package beepicker

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CourseHandler handles the request for retrieving courses from the BeePicker.
// @Tags BeePicker
// @Summary Retrieves courses from the BeePicker.
// @Produce json
// @Router /beePicker/courses [get]
func CourseHandler(c *gin.Context) {

	data, err := CourseService()

	if err != nil {
		switch err.Error() {
		case "error getting newest folder", "error getting course codes", "error getting course data":
			c.JSON(http.StatusBadGateway, gin.H{"error": "cannot retrieve course information"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
	}

	c.JSON(http.StatusOK, data)

}
