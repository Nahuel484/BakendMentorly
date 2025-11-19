package handlers

import (
	"net/http"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type PostulacionHandler struct {
	postulacionService *services.PostulacionService
}

func NewPostulacionHandler(s *services.PostulacionService) *PostulacionHandler {
	return &PostulacionHandler{postulacionService: s}
}

type createPostulacionRequest struct {
	IDSolicitud int `json:"id_solicitud" binding:"required"`
}

type rejectPostulacionRequest struct {
	IDPostulacion int `json:"id_postulacion" binding:"required"`
}

// POST /api/postulaciones
func (h *PostulacionHandler) CreatePostulacion(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req createPostulacionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	post, _, err := h.postulacionService.CreatePostulacion(c.Request.Context(), idPersona, req.IDSolicitud)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudo crear la postulación",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Postulación creada",
		Data:    post,
	})
}

// POST /api/postulaciones/rechazar
func (h *PostulacionHandler) RejectPostulacion(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req rejectPostulacionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	if err := h.postulacionService.RejectPostulacion(c.Request.Context(), req.IDPostulacion, idPersona); err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudo rechazar la postulación",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Postulación rechazada",
	})
}
