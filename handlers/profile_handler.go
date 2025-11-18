package handlers

import (
	"context"
	"errors"
	"mentorly-backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdateProfileRequest estructura para actualizar perfil
type UpdateProfileRequest struct {
	Nombre   *string `json:"nombre" binding:"omitempty,min=2"`
	Apellido *string `json:"apellido" binding:"omitempty,min=2"`
	Telefono *string `json:"telefono" binding:"omitempty"`
	Bio      *string `json:"bio" binding:"omitempty,max=500"`
	Avatar   *string `json:"avatar" binding:"omitempty,url"` // URL de la imagen
}

// ProfileHandler maneja las operaciones de perfil
type ProfileHandler struct {
	userService *services.UserService
}

// NewProfileHandler crea un nuevo handler de perfil
func NewProfileHandler(db *pgxpool.Pool) *ProfileHandler {
	return &ProfileHandler{
		userService: services.NewUserService(db),
	}
}

// UpdateProfileHandler - Actualizar el perfil del usuario
func (h *ProfileHandler) UpdateProfileHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{Success: false, Message: "Usuario no autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "Datos inválidos: " + err.Error()})
		return
	}

	// Si no se envió ningún dato, no hay nada que hacer
	if req.Nombre == nil && req.Apellido == nil && req.Telefono == nil && req.Bio == nil && req.Avatar == nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "No se proporcionaron datos para actualizar"})
		return
	}

	// Preparar valores para enviar al servicio
	var nombre, apellido string
	if req.Nombre != nil {
		nombre = *req.Nombre
	}
	if req.Apellido != nil {
		apellido = *req.Apellido
	}

	// Llamar al servicio para actualizar el perfil
	err := h.userService.UpdateUserProfile(context.Background(), idPersona, nombre, apellido, req.Telefono, req.Bio, req.Avatar)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{Success: false, Message: "Usuario no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al actualizar el perfil: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Perfil actualizado correctamente",
	})
}

// GetProfileHandler - Obtener perfil del usuario
func (h *ProfileHandler) GetProfileHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
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

	// Obtener perfil del usuario
	profile, err := h.userService.GetUserProfile(context.Background(), idPersona)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{Success: false, Message: "Usuario no encontrado"})
			return
		}
		c.JSON(http.StatusNotFound, ResponseData{
			Success: false, Message: "Error al obtener el perfil: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Perfil obtenido",
		Data: gin.H{
			"id_persona": profile.IDPersona,
			"nombre":     profile.Nombre,
			"apellido":   profile.Apellido,
			"email":      profile.Email,
			"telefono":   profile.Telefono,
			"bio":        profile.Bio,
			"avatar":     profile.Avatar,
			"rol":        profile.Rol,
		},
	})
}
