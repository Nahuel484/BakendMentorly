package handlers

import (
	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	planService        *services.PlanService
	mercadoPagoService *services.MercadoPagoService
	suscripcionService *services.SuscripcionService
}

func NewPaymentHandler(ps *services.PlanService, mp *services.MercadoPagoService, ss *services.SuscripcionService) *PaymentHandler {
	return &PaymentHandler{planService: ps, mercadoPagoService: mp, suscripcionService: ss}
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

	ctx := c.Request.Context()

	// 🔹 1) Obtener plan solicitado
	plan, err := h.planService.GetPlanByID(ctx, req.PlanID)
	if err != nil {
		c.JSON(404, ResponseData{
			Success: false,
			Message: "Plan no encontrado",
			Data:    err.Error(),
		})
		return
	}

	// 🔹 2) Obtener suscripción activa SI EXISTE
	activeSub, err := h.suscripcionService.GetActiveByPersona(ctx, idPersona.(int))
	if err != nil {
		c.JSON(500, ResponseData{
			Success: false,
			Message: "Error consultando suscripción",
			Data:    err.Error(),
		})
		return
	}

	// 🔹 3) Si ya tiene suscripción → comparar niveles usando id_plan
	if activeSub != nil {
		currentPlan, err := h.planService.GetPlanByID(ctx, activeSub.IDPlan)
		if err != nil {
			c.JSON(500, ResponseData{
				Success: false,
				Message: "Error consultando plan actual",
			})
			return
		}

		// ✔️ USAMOS id_plan COMO NIVEL (0=gratis, 1=pro, 2/3=premium)
		if plan.ID <= currentPlan.ID {
			c.JSON(400, ResponseData{
				Success: false,
				Message: "Ya tenés el plan " + currentPlan.Nombre + ". Solo podés actualizar a uno superior.",
			})
			return
		}
	}

	// 🔹 4) Crear preferencia de Mercado Pago
	url, err := h.mercadoPagoService.CreatePreference(
		ctx,
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

	// 🔹 5) Respuesta OK
	c.JSON(200, ResponseData{
		Success: true,
		Message: "Preference creada",
		Data: gin.H{
			"init_point": url,
		},
	})
}
