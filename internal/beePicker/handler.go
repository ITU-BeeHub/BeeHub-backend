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

type scheduleSaveRequest struct {
	ScheduleName string `json:"scheduleName"`
	ECRN         []int  `json:"ECRN"`
	SCRN         []int  `json:"SCRN"`
}

// SelectHandler handles the request for selecting a course from the BeePicker.
// @Tags BeePicker
// @Summary Selects a course from the BeePicker.
// @Accept json
// @Produce json
// @Param request body scheduleSaveRequest true "Request body containing the ECRN"
// @Success 200 {object} string "Selection successful"
// @Failure 400 {object} string "Bad request"
// @Failure 500 {object} string "Internal server error"
// @Router /beePicker/schedule [post]
func (h *Handler) ScheduleSaveHandler(c *gin.Context) {

	var req scheduleSaveRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save the schedule
	err := h.service.ScheduleSaveService(req.ScheduleName, req.ECRN, req.SCRN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule saved successfully"})
}

// SchedulesHandler handles the request for retrieving schedules from the BeePicker.
// @Tags BeePicker
// @Summary Retrieves schedules from the BeePicker.
// @Produce json
// @Router /beePicker/schedule [get]
func (h *Handler) ScheduleHandler(c *gin.Context) {

	// Get the schedules
	data, err := h.service.SchedulesService()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}

	c.JSON(http.StatusOK, data)

}

type pickRequest struct {
	CourseCodes []string `json:"courseCodes"`
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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}


	data, err := h.service.PickService(req.CourseCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

