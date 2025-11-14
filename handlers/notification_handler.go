package handlers

import (
	"context"
	"mentorly-backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(db *pgxpool.Pool) *NotificationHandler {
	return &NotificationHandler{
		notificationService: services.NewNotificationService(db),
	}
}

// CreateNotificationHandler - Crear una nueva notificación
func (h *NotificationHandler) CreateNotificationHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req services.NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Datos inválidos: " + err.Error()})
		return
	}

	// Asignar el ID del usuario autenticado
	req.IDPersona = idPersona

	notification, err := h.notificationService.CreateNotification(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error al crear notificación"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Notificación creada",
		"data":    notification,
	})
}

// GetUserNotificationsHandler - Obtener notificaciones del usuario
func (h *NotificationHandler) GetUserNotificationsHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	soloNoLeidas := c.Query("unread") == "true"

	notifications, err := h.notificationService.GetUserNotifications(context.Background(), idPersona, soloNoLeidas)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error al obtener notificaciones"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// MarkAsReadHandler - Marcar notificación como leída
func (h *NotificationHandler) MarkAsReadHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	idNotificacion, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ID de notificación inválido"})
		return
	}

	err = h.notificationService.MarkAsRead(context.Background(), idNotificacion, idPersona)
	if err != nil {
		if err == services.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Notificación no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error al marcar como leída"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación marcada como leída",
	})
}

// MarkAllAsReadHandler - Marcar todas las notificaciones como leídas
func (h *NotificationHandler) MarkAllAsReadHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	err := h.notificationService.MarkAllAsRead(context.Background(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error al marcar todas como leídas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todas las notificaciones marcadas como leídas",
	})
}

// GetUnreadCountHandler - Obtener número de notificaciones no leídas
func (h *NotificationHandler) GetUnreadCountHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	count, err := h.notificationService.GetUnreadCount(context.Background(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error al obtener contador"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"unread_count": count,
		},
	})
}

// DeleteNotificationHandler - Eliminar notificación
func (h *NotificationHandler) DeleteNotificationHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	idNotificacion, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ID de notificación inválido"})
		return
	}

	err = h.notificationService.DeleteNotification(context.Background(), idNotificacion, idPersona)
	if err != nil {
		if err == services.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Notificación no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error al eliminar notificación"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notificación eliminada",
	})
}
