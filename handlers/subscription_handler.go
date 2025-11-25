package handlers

import (
	"net/http"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type SuscripcionHandler struct {
	suscripcionService *services.SuscripcionService
	planService        *services.PlanService
}

func NewSuscripcionHandler(ss *services.SuscripcionService, ps *services.PlanService) *SuscripcionHandler {
	return &SuscripcionHandler{
		suscripcionService: ss,
		planService:        ps,
	}
}

func (h *SuscripcionHandler) GetMySubscription(c *gin.Context) {
	// id_persona lo sacás igual que en PaymentHandler
	idPersonaAny, ok := c.Get("id_persona")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	idPersona := idPersonaAny.(int)

	ctx := c.Request.Context()

	sub, err := h.suscripcionService.GetActiveByPersona(ctx, idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error consultando suscripción"})
		return
	}

	if sub == nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	plan, err := h.planService.GetPlanByID(ctx, sub.IDPlan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error consultando plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active": true,
		"plan": gin.H{
			"id":     plan.ID,
			"nombre": plan.Nombre,
			"precio": plan.Precio,
		},
		"fecha_inicial":    sub.FechaInicial,
		"fecha_expiracion": sub.FechaExpiracion,
	})
}
