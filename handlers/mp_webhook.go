package handlers

import (
	"fmt"
	"log"
	"mentorly-backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MPWebhookHandler struct {
	mercadoPagoService *services.MercadoPagoService
	suscripcionService *services.SuscripcionService
}

func NewMPWebhookHandler(mp *services.MercadoPagoService, ss *services.SuscripcionService) *MPWebhookHandler {
	return &MPWebhookHandler{
		mercadoPagoService: mp,
		suscripcionService: ss,
	}
}

func (h *MPWebhookHandler) HandleWebhook(c *gin.Context) {
	var body map[string]interface{}

	// Se intenta parsear el JSON, pero si falla igual seguimos porque MP manda datos por query
	if err := c.BindJSON(&body); err != nil {
		log.Println("Error parseando body de webhook MP:", err)
	} else {
		log.Printf("Webhook MP body: %#v\n", body)
	}

	// 1) Sacar el type: primero query, luego body
	t := c.Query("type")
	if t == "" {
		if tv, ok := body["type"].(string); ok {
			t = tv
		}
	}

	if t != "payment" && t != "payment.updated" && t != "payment.created" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// 2) Sacar el payment_id
	var paymentIDStr string

	// Prioridad: query param data.id (es lo que te está mandando MP en la simulación)
	if idFromQuery := c.Query("data.id"); idFromQuery != "" {
		paymentIDStr = idFromQuery
	} else {
		// Si no viene por query, intentamos sacarlo del body["data"]["id"]
		if dataRaw, ok := body["data"].(map[string]interface{}); ok {
			paymentIDStr = fmt.Sprint(dataRaw["id"]) // convierte lo que sea a string
		} else if idRaw, ok := body["id"]; ok {
			// Algunos ejemplos de MP usan "id" directo
			paymentIDStr = fmt.Sprint(idRaw)
		}
	}

	if paymentIDStr == "" {
		log.Println("Webhook MP sin payment id, body/query:", body, c.Request.URL.RawQuery)
		c.JSON(http.StatusOK, gin.H{"status": "missing payment id"})
		return
	}

	paymentID, err := strconv.Atoi(paymentIDStr)
	if err != nil {
		log.Println("Payment id inválido en webhook MP:", paymentIDStr, "error:", err)
		c.JSON(http.StatusOK, gin.H{"status": "invalid payment id"})
		return
	}

	// 3) Llamar a Mercado Pago para obtener el pago completo
	payment, err := h.mercadoPagoService.GetPaymentInfo(paymentID)
	if err != nil {
		log.Println("Error obteniendo info de pago en MP:", err)
		c.JSON(http.StatusOK, gin.H{"status": "error fetching payment"})
		return
	}

	if payment.Status != "approved" {
		log.Println("Pago no aprobado, status:", payment.Status)
		c.JSON(http.StatusOK, gin.H{"status": "payment not approved"})
		return
	}

	// 4) Leer metadata y activar suscripción
	userIDStr := fmt.Sprint(payment.Metadata["user_id"])
	planIDStr := fmt.Sprint(payment.Metadata["plan_id"])

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Println("user_id inválido en metadata MP:", userIDStr, err)
		c.JSON(http.StatusOK, gin.H{"status": "invalid user id"})
		return
	}

	planID, err := strconv.Atoi(planIDStr)
	if err != nil {
		log.Println("plan_id inválido en metadata MP:", planIDStr, err)
		c.JSON(http.StatusOK, gin.H{"status": "invalid plan id"})
		return
	}

	if err := h.suscripcionService.Activate(userID, planID); err != nil {
		log.Println("Error activando suscripción:", err)
		c.JSON(http.StatusOK, gin.H{"status": "error activating subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
