package handlers

import (
	"net/http"

	"mentorly-backend/services" // 🔁 cambia por tu módulo

	"github.com/gin-gonic/gin"
)

type SolicitudHandler struct {
	solicitudService *services.SolicitudService
}

func NewSolicitudHandler(s *services.SolicitudService) *SolicitudHandler {
	return &SolicitudHandler{solicitudService: s}
}

type createSolicitudRequest struct {
	Titulo      string `json:"titulo" binding:"required"`
	Descripcion string `json:"descripcion" binding:"required"`
}

// POST /api/solicitudes
func (h *SolicitudHandler) CreateSolicitud(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req createSolicitudRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	input := services.CreateSolicitudInput{
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
	}

	sol, err := h.solicitudService.CreateSolicitud(c.Request.Context(), idPersona, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudo crear la solicitud",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Solicitud creada",
		Data:    sol,
	})
}

// GET /api/solicitudes/explore
func (h *SolicitudHandler) ListSolicitudesAbiertas(c *gin.Context) {
	solicitudes, err := h.solicitudService.ListSolicitudesAbiertas(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudieron obtener las solicitudes",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Solicitudes abiertas",
		Data:    solicitudes,
	})
}

// GET /api/solicitudes/mias
func (h *SolicitudHandler) ListMisSolicitudes(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	solicitudes, err := h.solicitudService.ListSolicitudesByContratante(c.Request.Context(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudieron obtener tus solicitudes",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Solicitudes del usuario",
		Data:    solicitudes,
	})
}
