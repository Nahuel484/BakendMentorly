package handlers

import (
	"net/http"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type SolicitudHandler struct {
	solicitudService   *services.SolicitudService
	suscripcionService *services.SuscripcionService
}

func NewSolicitudHandler(s *services.SolicitudService, ss *services.SuscripcionService) *SolicitudHandler {
	return &SolicitudHandler{
		solicitudService:   s,
		suscripcionService: ss,
	}
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
	ctx := c.Request.Context()

	var req createSolicitudRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	// 🔹 1) Obtener suscripción activa (o asumir plan Gratis si no tiene)
	sub, err := h.suscripcionService.GetActiveByPersona(ctx, idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error consultando suscripción",
			Data:    err.Error(),
		})
		return
	}

	planID := services.DefaultFreePlanID
	if sub != nil {
		planID = sub.IDPlan
	}

	limits := services.GetPlanLimitsByID(planID)

	// 🔹 2) Contar solicitudes/anuncios activos del usuario
	count, err := h.solicitudService.CountSolicitudesActivas(ctx, idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error consultando tus anuncios activos",
			Data:    err.Error(),
		})
		return
	}

	if count >= limits.MaxAnunciosEmprendedor {
		c.JSON(http.StatusForbidden, ResponseData{
			Success: false,
			Message: "Has alcanzado el máximo de anuncios activos para tu plan. Mejora tu suscripción para crear más solicitudes.",
		})
		return
	}

	// 🔹 3) Crear la solicitud normalmente
	input := services.CreateSolicitudInput{
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
	}

	sol, err := h.solicitudService.CreateSolicitud(ctx, idPersona, input)
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
