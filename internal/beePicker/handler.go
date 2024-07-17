package beepicker

import (
	"github.com/gin-gonic/gin"
)

// @Tags BeePicker
// @Summary Hello World
// @Accept json
// @Produce json
// @Router /beePicker/courses [get]
func CourseHandler(c *gin.Context) {

	data, err := CourseService()

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	}

	c.JSON(200, data)

}
