package handlers

import (
	"context"
	"errors"
	"mentorly-backend/models"
	"mentorly-backend/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	authService         *services.AuthService
	userService         *services.UserService
	roleService         *services.RoleService
	planService         *services.PlanService
	subscriptionService *services.SubscriptionService
}

// RegisterRequest - Estructura para registro con campos en minúsculas
type RegisterRequest struct {
	Nombre     string `json:"nombre" binding:"required,min=2"`
	Apellido   string `json:"apellido" binding:"required,min=2"`
	Email      string `json:"email" binding:"required,email"`
	Contrasena string `json:"contrasena" binding:"required,min=6"`
	Confirmar  string `json:"confirmar" binding:"required"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Contrasena string `json:"contrasena" binding:"required"`
}

type SelectRoleRequest struct {
	Rol string `json:"rol" binding:"required"`
}

type UpdateProfileRequest struct {
	// omitempty permite que los campos no se envíen si no se quieren modificar
	Nombre   string `json:"nombre" binding:"omitempty,min=2"`
	Apellido string `json:"apellido" binding:"omitempty,min=2"`
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{
		authService:         services.NewAuthService(db),
		userService:         services.NewUserService(db),
		roleService:         services.NewRoleService(db),
		planService:         services.NewPlanService(db),
		subscriptionService: services.NewSubscriptionService(db),
	}
}

// RegisterHandler - Registrar nuevo usuario
func (h *Handler) RegisterHandler(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	// Validar que las contraseñas coincidan
	if req.Contrasena != req.Confirmar {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Las contraseñas no coinciden",
		})
		return
	}

	// Hash de la contraseña
	hashedPassword, err := h.authService.HashPassword(req.Contrasena)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al procesar contraseña",
		})
		return
	}

	// Registrar usuario
	idPersona, err := h.authService.RegisterUser(context.Background(), req.Nombre, req.Apellido, req.Email, hashedPassword)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, ResponseData{
				Success: false,
				Message: "El email ya está registrado",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ResponseData{
				Success: false,
				Message: "Error al crear usuario",
			})
		}
		return
	}

	// Generar token
	token, err := GenerateToken(idPersona, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al generar token",
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Usuario registrado exitosamente",
		Data: TokenResponse{
			Token:     token,
			IDPersona: idPersona,
			Email:     req.Email,
			Nombre:    req.Nombre,
		},
	})
}

// LoginHandler - Iniciar sesión
func (h *Handler) LoginHandler(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos",
		})
		return
	}

	// Verificar credenciales
	idPersona, nombre, err := h.authService.LoginUser(context.Background(), req.Email, req.Contrasena)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Email o contraseña incorrectos",
		})
		return
	}

	// Generar token
	token, err := GenerateToken(idPersona, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al generar token",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Sesión iniciada correctamente",
		Data: TokenResponse{
			Token:     token,
			IDPersona: idPersona,
			Email:     req.Email,
			Nombre:    nombre,
		},
	})
}

// SelectRoleHandler - Seleccionar rol
func (h *Handler) SelectRoleHandler(c *gin.Context) {
	var req SelectRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Rol inválido",
		})
		return
	}

	// Obtener ID de persona del token
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

	// Obtener ID del rol
	idRol, err := h.userService.GetRoleIDByName(context.Background(), req.Rol)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Rol no válido",
		})
		return
	}

	// Actualizar rol
	err = h.userService.UpdateUserRole(context.Background(), idPersona, idRol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al actualizar rol",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Rol asignado correctamente",
		Data: gin.H{
			"rol": req.Rol,
		},
	})
}

// GetProfileHandler - Obtener perfil del usuario
func (h *Handler) GetProfileHandler(c *gin.Context) {
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

	// Obtener perfil del usuario
	profile, err := h.userService.GetUserProfile(context.Background(), idPersona)
	if err != nil {
		c.JSON(http.StatusNotFound, ResponseData{
			Success: false,
			Message: "Usuario no encontrado",
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
			"rol":        profile.Rol,
		},
	})
}

// UpdateProfileHandler - Actualizar el perfil del usuario
func (h *Handler) UpdateProfileHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{Success: false, Message: "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "Datos inválidos: " + err.Error()})
		return
	}

	// Si no se envía ningún dato, no hay nada que hacer.
	if req.Nombre == "" && req.Apellido == "" {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "No se proporcionaron datos para actualizar"})
		return
	}

	// Llamar al servicio para actualizar el perfil
	err := h.userService.UpdateUserProfile(context.Background(), idPersona, req.Nombre, req.Apellido)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al actualizar el perfil",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Perfil actualizado correctamente",
	})
}

// SubscribeToPlanHandler - Suscribe al usuario logueado a un plan
func (h *Handler) SubscribeToPlanHandler(c *gin.Context) {
	// 1. Obtener el ID del usuario desde el token
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{Success: false, Message: "No autenticado"})
		return
	}
	idPersona := idPersonaInterface.(int)

	// 2. Obtener el ID del plan desde la URL (parámetro)
	idPlan, err := strconv.Atoi(c.Param("plan_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "ID de plan inválido"})
		return
	}

	// 3. Llamar al servicio para crear la suscripción
	// (Aquí en el futuro podrías añadir lógica de pago con Stripe, etc.)
	subscription, err := h.subscriptionService.CreateSubscription(context.Background(), idPersona, idPlan)
	if err != nil {
		c.Error(err) // Loguear el error real
		c.JSON(http.StatusInternalServerError, ResponseData{Success: false, Message: "Error al procesar la suscripción"})
		return
	}

	// 4. Devolver una respuesta exitosa
	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Suscripción creada exitosamente",
		Data:    subscription,
	})
}

// Plan Handlers

// CreatePlanHandler - Crea un nuevo plan
func (h *Handler) CreatePlanHandler(c *gin.Context) {
	var plan models.Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "Datos de plan inválidos: " + err.Error()})
		return
	}

	createdPlan, err := h.planService.CreatePlan(context.Background(), plan)
	if err != nil {
		c.Error(err) // Añadimos el error al contexto de Gin para que se muestre en el log
		c.JSON(http.StatusInternalServerError, ResponseData{Success: false, Message: "Error al crear el plan"})
		return
	}

	c.JSON(http.StatusCreated, ResponseData{
		Success: true,
		Message: "Plan creado exitosamente",
		Data:    createdPlan,
	})
}

// GetAllPlansHandler - Obtiene todos los planes
func (h *Handler) GetAllPlansHandler(c *gin.Context) {
	plans, err := h.planService.GetAllPlans(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{Success: false, Message: "Error al obtener los planes"})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Planes obtenidos correctamente",
		Data:    plans,
	})
}

// GetPlanByIDHandler - Obtiene un plan por su ID
func (h *Handler) GetPlanByIDHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "ID de plan inválido"})
		return
	}

	plan, err := h.planService.GetPlanByID(context.Background(), id)
	if err != nil {
		// Aquí sería ideal chequear si el error es pgx.ErrNoRows y devolver 404
		c.JSON(http.StatusNotFound, ResponseData{Success: false, Message: "Plan no encontrado"})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Plan obtenido correctamente",
		Data:    plan,
	})
}

// UpdatePlanHandler - Actualiza un plan existente
func (h *Handler) UpdatePlanHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "ID de plan inválido"})
		return
	}

	var plan models.Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "Datos de plan inválidos: " + err.Error()})
		return
	}

	err = h.planService.UpdatePlan(context.Background(), id, plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{Success: false, Message: "Error al actualizar el plan"})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Plan actualizado correctamente",
	})
}

// DeletePlanHandler - Elimina un plan
func (h *Handler) DeletePlanHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{Success: false, Message: "ID de plan inválido"})
		return
	}

	err = h.planService.DeletePlan(context.Background(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{Success: false, Message: "Plan no encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, ResponseData{Success: false, Message: "Error al eliminar el plan"})
		}
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Plan eliminado correctamente",
	})
}

// AuthMiddleware - acepta Authorization: Bearer <token> O cookie "auth_token"
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""

		// 1) Intentar por header Authorization
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") && len(authHeader) > 7 {
			token = strings.TrimSpace(authHeader[7:])
		}

		// 2) Si no hay header, intentar por cookie (flujo OAuth)
		if token == "" {
			if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
				token = cookie
			}
		}

		// 3) Si sigue vacío, rechazar
		if token == "" {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Token requerido",
			})
			c.Abort()
			return
		}

		// 4) Validar JWT
		claims, err := VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Token inválido o expirado",
			})
			c.Abort()
			return
		}

		// 5) Dejar datos en contexto
		c.Set("id_persona", claims.IDPersona)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// AdminMiddleware - Middleware para proteger rutas de administrador
func (h *Handler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idPersonaInterface, exists := c.Get("id_persona")
		if !exists {
			c.JSON(http.StatusUnauthorized, ResponseData{Success: false, Message: "No autenticado"})
			c.Abort()
			return
		}
		idPersona := idPersonaInterface.(int)

		// Obtener el perfil del usuario para verificar su rol
		profile, err := h.userService.GetUserProfile(context.Background(), idPersona)
		if err != nil {
			c.JSON(http.StatusForbidden, ResponseData{Success: false, Message: "Usuario no válido"})
			c.Abort()
			return
		}

		// Permitir que los mentores sean administradores
		if profile.Rol != "mentor" {
			c.JSON(http.StatusForbidden, ResponseData{
				Success: false,
				Message: "Acceso denegado. Se requiere rol de mentor.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
