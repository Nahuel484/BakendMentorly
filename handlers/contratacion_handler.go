package handlers

import (
	"net/http"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type ContratacionHandler struct {
	contratacionService *services.ContratacionService
}

func NewContratacionHandler(s *services.ContratacionService) *ContratacionHandler {
	return &ContratacionHandler{contratacionService: s}
}

type createContratacionRequest struct {
	IDPostulacion int `json:"id_postulacion" binding:"required"`
}

// POST /api/contrataciones
func (h *ContratacionHandler) CreateContratacion(c *gin.Context) {
	// Podrías validar que quien contrata sea el dueño de la solicitud, pero por ahora lo dejamos simple.
	var req createContratacionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	result, err := h.contratacionService.CreateContratacion(c.Request.Context(), req.IDPostulacion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudo crear la contratación",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Contratación creada y conversación abierta",
		Data:    result,
	})
}
