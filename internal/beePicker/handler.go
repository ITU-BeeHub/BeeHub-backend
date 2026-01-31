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
// @Success 200 {object} map[string]interface{} "course list"
// @Failure 502 {object} map[string]string "cannot retrieve course information"
// @Failure 500 {object} map[string]string "internal server error"
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
		return
	}

	c.JSON(http.StatusOK, data)
}

// pickRequest ders seçim/silme isteğini temsil eder
// Tek bir API isteğinde hem ekleme hem silme yapılabilir
type pickRequest struct {
	CourseCodes []string `json:"courseCodes,omitempty"`

	ECRN []string `json:"ECRN,omitempty"` // Eklenecek CRN'ler
	SCRN []string `json:"SCRN,omitempty"` // Silinecek CRN'ler
}

// PickHandler handles the request for picking/dropping courses from the BeePicker.
// @Tags BeePicker
// @Summary Picks or drops courses from Kepler.
// @Description Picks courses (ECRN) and/or drops courses (SCRN) based on the request.
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body pickRequest true "Request body containing courses to add and/or drop"
// @Success 200 {object} map[string]interface{} "Operation successful"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /beePicker/pick [post]
func (h *Handler) PickHandler(c *gin.Context) {
	var req pickRequest

	// JSON bind işlemi ve hata kontrolü
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Eklenecek dersler: ECRN veya eski format courseCodes
	var addCRNs []string
	if len(req.ECRN) > 0 {
		addCRNs = req.ECRN
	} else if len(req.CourseCodes) > 0 {
		// Eski format uyumluluğu
		addCRNs = req.CourseCodes
	}

	// Silinecek dersler: SCRN
	dropCRNs := req.SCRN

	// En az bir ders ekleme veya silme işlemi olmalı
	if len(addCRNs) == 0 && len(dropCRNs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one course to add or drop is required"})
		return
	}

	// CRN array'ini service katmanına iletme
	data, err := h.service.PickService(addCRNs, dropCRNs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
