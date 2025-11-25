package handlers

import (
	"net/http"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type PostulacionHandler struct {
	postulacionService *services.PostulacionService
	suscripcionService *services.SuscripcionService
}

func NewPostulacionHandler(
	s *services.PostulacionService,
	ss *services.SuscripcionService,
) *PostulacionHandler {
	return &PostulacionHandler{
		postulacionService: s,
		suscripcionService: ss,
	}
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
	ctx := c.Request.Context()

	var req createPostulacionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	// 🔹 1) Obtener suscripción activa (o asumir Gratis si no tiene)
	sub, err := h.suscripcionService.GetActiveByPersona(ctx, idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error consultando suscripción",
		})
		return
	}

	planID := services.DefaultFreePlanID
	if sub != nil {
		planID = sub.IDPlan
	}
	limits := services.GetPlanLimitsByID(planID)

	// 🔹 2) Contar postulaciones del mes actual
	count, err := h.postulacionService.CountPostulacionesMes(ctx, idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error consultando postulaciones existentes",
			Data:    err.Error(),
		})
		return
	}

	if count >= limits.MaxPostulacionesMentor {
		c.JSON(http.StatusForbidden, ResponseData{
			Success: false,
			Message: "Has alcanzado el máximo de postulaciones para tu plan. Mejora tu suscripción para seguir postulando.",
		})
		return
	}

	// 🔹 3) Crear la postulación normalmente
	post, _, err := h.postulacionService.CreatePostulacion(ctx, idPersona, req.IDSolicitud)
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
