package handlers

import (
	"fmt"
	"mentorly-backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PlanHandler struct {
	planService *services.PlanService
}

func NewPlanHandler(ps *services.PlanService) *PlanHandler {
	return &PlanHandler{planService: ps}
}

// GET /api/planes (listado de planes activos para frontend)
func (h *PlanHandler) GetActivePlans(c *gin.Context) {
	planes, err := h.planService.GetActivePlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "No se pudieron obtener los planes",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Planes activos",
		Data:    planes,
	})
}

// GET /api/planes/all (usar solo como admin)
func (h *PlanHandler) GetAllPlans(c *gin.Context) {
	planes, err := h.planService.GetAllPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener todos los planes",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Todos los planes",
		Data:    planes,
	})
}

// GET /api/planes/:id
func (h *PlanHandler) GetPlanByID(c *gin.Context) {
	id := c.Param("id")

	plan, err := h.planService.GetPlanByID(c.Request.Context(), toInt(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ResponseData{
			Success: false,
			Message: "No se encontró el plan",
			Data:    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Plan encontrado",
		Data:    plan,
	})
}

func toInt(s string) int {
	var n int
	fmt.Sscan(s, &n)
	return n
}
