package handlers

import (
	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	planService        *services.PlanService
	mercadoPagoService *services.MercadoPagoService
}

func NewPaymentHandler(ps *services.PlanService, mp *services.MercadoPagoService) *PaymentHandler {
	return &PaymentHandler{planService: ps, mercadoPagoService: mp}
}

type CreatePreferenceRequest struct {
	PlanID int `json:"plan_id"`
}

func (h *PaymentHandler) CreateMercadoPagoPreference(c *gin.Context) {
	idPersona, exists := c.Get("id_persona")
	if !exists {
		c.JSON(401, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}

	var req CreatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	plan, err := h.planService.GetPlanByID(c.Request.Context(), req.PlanID)
	if err != nil {
		c.JSON(404, ResponseData{
			Success: false,
			Message: "Plan no encontrado",
			Data:    err.Error(),
		})
		return
	}

	url, err := h.mercadoPagoService.CreatePreference(
		c.Request.Context(),
		"Plan "+plan.Nombre,
		plan.Precio,
		idPersona.(int),
		plan.ID,
	)

	if err != nil {
		c.JSON(500, ResponseData{
			Success: false,
			Message: "Error creando pago",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(200, ResponseData{
		Success: true,
		Message: "Preference creada",
		Data: gin.H{
			"init_point": url,
		},
	})
}
