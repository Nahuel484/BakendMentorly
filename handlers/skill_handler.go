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

type SkillHandler struct {
	skillService *services.SkillService
}

func NewSkillHandler(db *pgxpool.Pool) *SkillHandler {
	return &SkillHandler{
		skillService: services.NewSkillService(db),
	}
}

type CreateSkillRequest struct {
	NombreHabilidad string `json:"nombre_habilidad" binding:"required"`
	Descripcion     string `json:"descripcion"`
	Categoria       string `json:"categoria"`
}

type AddUserSkillRequest struct {
	IDHabilidad  int    `json:"id_habilidad" binding:"required"`
	NivelDominio string `json:"nivel_dominio" binding:"required,oneof=beginner intermediate advanced"`
}

type UpdateSkillLevelRequest struct {
	NivelDominio string `json:"nivel_dominio" binding:"required,oneof=beginner intermediate advanced"`
}

// CreateSkillHandler - Crear una nueva habilidad (admin)
func (h *SkillHandler) CreateSkillHandler(c *gin.Context) {
	var req CreateSkillRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	skill, err := h.skillService.CreateSkill(context.Background(), req.NombreHabilidad)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al crear habilidad",
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Habilidad creada exitosamente",
		Data:    skill,
	})
}

// GetAllSkillsHandler - Obtener todas las habilidades disponibles
func (h *SkillHandler) GetAllSkillsHandler(c *gin.Context) {
	skills, err := h.skillService.GetAllSkills(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener habilidades",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Habilidades obtenidas correctamente",
		Data:    skills,
	})
}

// GetSkillByIDHandler - Obtener una habilidad por ID
func (h *SkillHandler) GetSkillByIDHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID inválido",
		})
		return
	}

	skill, err := h.skillService.GetSkillByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ResponseData{
			Success: false,
			Message: "Habilidad no encontrada",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Habilidad obtenida correctamente",
		Data:    skill,
	})
}

// AddSkillToUserHandler - Agregar una habilidad al usuario actual
func (h *SkillHandler) AddSkillToUserHandler(c *gin.Context) {
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

	var req AddUserSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	userSkill, err := h.skillService.AddSkillToUser(context.Background(), idPersona, req.IDHabilidad, req.NivelDominio)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al agregar habilidad. Puede que ya exista.",
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Habilidad agregada al usuario",
		Data:    userSkill,
	})
}

// GetUserSkillsHandler - Obtener las habilidades del usuario actual
func (h *SkillHandler) GetUserSkillsHandler(c *gin.Context) {
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

	skills, err := h.skillService.GetUserSkills(context.Background(), idPersona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener habilidades",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Habilidades del usuario obtenidas",
		Data:    skills,
	})
}

// RemoveUserSkillHandler - Eliminar una habilidad del usuario
func (h *SkillHandler) RemoveUserSkillHandler(c *gin.Context) {
	idUsuarioHabilidad, err := strconv.Atoi(c.Param("skill_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID de habilidad inválido",
		})
		return
	}

	err = h.skillService.RemoveUserSkill(context.Background(), idUsuarioHabilidad)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{
				Success: false,
				Message: "Habilidad no encontrada",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ResponseData{
				Success: false,
				Message: "Error al eliminar habilidad",
			})
		}
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Habilidad eliminada correctamente",
	})
}

// UpdateUserSkillLevelHandler - Actualizar el nivel de dominio de una habilidad
func (h *SkillHandler) UpdateUserSkillLevelHandler(c *gin.Context) {
	idUsuarioHabilidad, err := strconv.Atoi(c.Param("skill_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "ID de habilidad inválido",
		})
		return
	}

	var req UpdateSkillLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	err = h.skillService.UpdateUserSkillLevel(context.Background(), idUsuarioHabilidad, req.NivelDominio)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al actualizar nivel de habilidad",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Nivel de habilidad actualizado",
	})
}
