package handlers

import (
	"net/http"
	"strconv"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService *services.MessageService
}

func NewMessageHandler(ms *services.MessageService) *MessageHandler {
	return &MessageHandler{messageService: ms}
}

// ==========================================================
// ENVIAR MENSAJE  (POST /api/messages)
// ==========================================================

type sendMessageRequest struct {
	IDConversacion int    `json:"id_conversacion" binding:"required"`
	Contenido      string `json:"contenido" binding:"required"`
}

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

	msg, err := h.messageService.SendMessage(
		c.Request.Context(),
		idPersona,
		services.SendMessageInput{
			IDConversacion: req.IDConversacion,
			Contenido:      req.Contenido,
		},
	)
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

// ==========================================================
// LISTAR CONVERSACIONES (GET /api/conversaciones/mias)
// ==========================================================

func (h *MessageHandler) ListMisConversaciones(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	convs, err := h.messageService.ListConversacionesByPersona(c.Request.Context(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudieron obtener las conversaciones",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Conversaciones obtenidas",
		Data:    convs,
	})
}

// ==========================================================
// LISTAR MENSAJES DE UNA CONVERSACIÓN (GET /api/conversaciones/:id/mensajes)
// ==========================================================

func (h *MessageHandler) ListMensajesConversacion(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	idStr := c.Param("id")
	idConversacion, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID de conversación inválido",
		})
		return
	}

	// 👇 aquí usamos el nombre EXACTO del método del service
	msgs, err := h.messageService.ListMessagesByConversacion(
		c.Request.Context(),
		idConversacion,
		idPersona,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, ResponseData{
			Success: false,
			Message: "No se pudieron obtener los mensajes",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Mensajes obtenidos",
		Data:    msgs,
	})
}

// ==========================================================
// CERRAR CONVERSACIÓN (POST /api/conversaciones/:id/cerrar)
// ==========================================================

func (h *MessageHandler) CerrarConversacion(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}
	idPersona := idPersonaInterface.(int)

	idStr := c.Param("id")
	idConversacion, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID de conversación inválido",
		})
		return
	}

	// el service espera (ctx, idConversacion int, idPersona int)
	if err := h.messageService.CerrarConversacion(
		c.Request.Context(),
		idConversacion,
		idPersona,
	); err != nil {
		c.JSON(http.StatusForbidden, ResponseData{
			Success: false,
			Message: "No se pudo cerrar la conversación",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Conversación cerrada correctamente",
	})
}
