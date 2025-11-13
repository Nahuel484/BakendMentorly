package handlers

import (
	"context"
	"mentorly-backend/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionHandler struct {
	sessionService *services.SessionService
}

func NewSessionHandler(db *pgxpool.Pool) *SessionHandler {
	return &SessionHandler{
		sessionService: services.NewSessionService(db),
	}
}

// LogoutHandler - Cierra la sesión actual del usuario
func (h *SessionHandler) LogoutHandler(c *gin.Context) {
	// Obtener el token del header Authorization
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Token requerido",
		})
		return
	}

	// Extraer el token (formato: "Bearer TOKEN")
	var tokenString string
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		tokenString = parts[1]
	} else {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Formato de token inválido",
		})
		return
	}

	// Cerrar la sesión
	err := h.sessionService.LogoutSession(context.Background(), tokenString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al cerrar sesión",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Sesión cerrada correctamente",
	})
}

// LogoutAllHandler - Cierra todas las sesiones del usuario
func (h *SessionHandler) LogoutAllHandler(c *gin.Context) {
	// Obtener el ID de persona del token
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "No autenticado",
		})
		return
	}

	idPersona, ok := idPersonaInterface.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "ID de usuario inválido",
		})
		return
	}

	// Cerrar todas las sesiones
	err := h.sessionService.LogoutAllSessions(context.Background(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al cerrar sesiones",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Todas las sesiones han sido cerradas",
	})
}