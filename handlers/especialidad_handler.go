package handlers

import (
	"context"
	"errors"
	"mentorly-backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EspecialidadHandler struct {
	especialidadService *services.EspecialidadService
}

func NewEspecialidadHandler(db *pgxpool.Pool) *EspecialidadHandler {
	return &EspecialidadHandler{
		especialidadService: services.NewEspecialidadService(db),
	}
}

type CreateEspecialidadRequest struct {
	NombreEspecialidad string `json:"nombre_especialidad" binding:"required"`
}

type AddUserEspecialidadRequest struct {
	IDEspecialidad int `json:"id_especialidad" binding:"required"`
}

// GetAllEspecialidadesHandler - Obtener todas las especialidades disponibles
func (h *EspecialidadHandler) GetAllEspecialidadesHandler(c *gin.Context) {
	especialidades, err := h.especialidadService.GetAllEspecialidades(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener especialidades",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Especialidades obtenidas correctamente",
		Data:    especialidades,
	})
}

// GetEspecialidadByIDHandler - Obtener una especialidad por ID
func (h *EspecialidadHandler) GetEspecialidadByIDHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID inválido",
		})
		return
	}

	especialidad, err := h.especialidadService.GetEspecialidadByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ResponseData{
			Success: false,
			Message: "Especialidad no encontrada",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Especialidad obtenida correctamente",
		Data:    especialidad,
	})
}

// CreateEspecialidadHandler - Crear una nueva especialidad (admin)
func (h *EspecialidadHandler) CreateEspecialidadHandler(c *gin.Context) {
	var req CreateEspecialidadRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	especialidad, err := h.especialidadService.CreateEspecialidad(context.Background(), req.NombreEspecialidad)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al crear especialidad",
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Especialidad creada exitosamente",
		Data:    especialidad,
	})
}

// DeleteEspecialidadHandler - Eliminar una especialidad (admin)
func (h *EspecialidadHandler) DeleteEspecialidadHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID inválido",
		})
		return
	}

	err = h.especialidadService.DeleteEspecialidad(context.Background(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{
				Success: false,
				Message: "Especialidad no encontrada",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ResponseData{
				Success: false,
				Message: "Error al eliminar especialidad",
			})
		}
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Especialidad eliminada correctamente",
	})
}

// AddEspecialidadToUserHandler - Agregar una especialidad al usuario actual
func (h *EspecialidadHandler) AddEspecialidadToUserHandler(c *gin.Context) {
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

	var req AddUserEspecialidadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	userEsp, err := h.especialidadService.AddEspecialidadToUser(context.Background(), idPersona, req.IDEspecialidad)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al agregar especialidad. Puede que ya exista.",
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Especialidad agregada al usuario",
		Data:    userEsp,
	})
}

// GetUserEspecialidadesHandler - Obtener las especialidades del usuario actual
func (h *EspecialidadHandler) GetUserEspecialidadesHandler(c *gin.Context) {
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

	especialidades, err := h.especialidadService.GetUserEspecialidades(context.Background(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener especialidades",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Especialidades del usuario obtenidas",
		Data:    especialidades,
	})
}

// RemoveUserEspecialidadHandler - Eliminar una especialidad del usuario
func (h *EspecialidadHandler) RemoveUserEspecialidadHandler(c *gin.Context) {
	idUsuarioEspecialidad, err := strconv.Atoi(c.Param("especialidad_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID de especialidad inválido",
		})
		return
	}

	err = h.especialidadService.RemoveUserEspecialidad(context.Background(), idUsuarioEspecialidad)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{
				Success: false,
				Message: "Especialidad no encontrada",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ResponseData{
				Success: false,
				Message: "Error al eliminar especialidad",
			})
		}
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Especialidad eliminada correctamente",
	})
}