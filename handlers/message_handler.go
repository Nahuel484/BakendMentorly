package handlers

import (
	"net/http"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService *services.MessageService
}

func NewMessageHandler(ms *services.MessageService) *MessageHandler {
	return &MessageHandler{messageService: ms}
}

type sendMessageRequest struct {
	IDConversacion int    `json:"id_conversacion" binding:"required"`
	Contenido      string `json:"contenido" binding:"required"`
}

// POST /api/messages
func (h *MessageHandler) SendMessage(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
			Data:    err.Error(),
		})
		return
	}

	input := services.SendMessageInput{
		IDConversacion: req.IDConversacion,
		Contenido:      req.Contenido,
	}

	msg, err := h.messageService.SendMessage(c.Request.Context(), idPersona, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudo enviar el mensaje",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Mensaje enviado",
		Data:    msg,
	})
}
