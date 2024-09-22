package beepicker

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CourseHandler handles the request for retrieving courses from the BeePicker.
// @Tags BeePicker
// @Summary Retrieves courses from the BeePicker.
// @Produce json
// @Router /beePicker/courses [get]
func (h *Handler) CourseHandler(c *gin.Context) {

	data, err := h.service.CourseService()

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

type CourseRequest struct {
	CRN     string          `json:"crn" binding:"required"`
	Reserves []CourseRequest `json:"reserves,omitempty"`
  }
  
  type pickRequest struct {
	Courses []CourseRequest `json:"courses" binding:"required,min=1"`
  }  


// PickHandler handles the request for picking a course from the BeePicker.
// @Tags BeePicker
// @Summary Picks a course from the kepler.
// @Accept json
// @Produce json
// @Param request body pickRequest true "Request body containing the course codes"
// @Success 200 {object} string "Picking successful"
// @Failure 400 {object} string "Bad request"
// @Failure 500 {object} string "Internal server error"
// @Router /beePicker/pick [post]
func (h *Handler) PickHandler(c *gin.Context) {
	var req pickRequest
  
	// Bind JSON and handle errors
	if err := c.ShouldBindJSON(&req); err != nil {
	  c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	  return
	}
  
	// Pass the courses to the service layer
	data, err := h.service.PickService(req.Courses)
	if err != nil {
	  c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	  return
	}
	c.JSON(http.StatusOK, data)
  }
  
