package handlers

import (
	"context"
	"errors"
	"log"
	"mentorly-backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	db           *pgxpool.Pool
	authService  *services.AuthService
	userService  *services.UserService
	roleService  *services.RoleService
	sessionService *services.SessionService
}

func NewAuthHandler(db *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{
		db:             db,
		authService:    services.NewAuthService(db),
		userService:    services.NewUserService(db),
		roleService:    services.NewRoleService(db),
		sessionService: services.NewSessionService(db),
	}
}

// RegisterRequest estructura para registro
type RegisterRequest struct {
	Nombre      string `json:"nombre" binding:"required,min=2"`
	Apellido    string `json:"apellido" binding:"required,min=2"`
	Email       string `json:"email" binding:"required,email"`
	Contrasena  string `json:"contrasena" binding:"required,min=6"`
	Confirmar   string `json:"confirmar" binding:"required,eqfield=Contrasena"`
}

// LoginRequest estructura para login
type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Contrasena string `json:"contrasena" binding:"required"`
}

// SelectRoleRequest estructura para seleccionar rol
type SelectRoleRequest struct {
	Rol string `json:"rol" binding:"required,oneof=mentor emprendedor"`
}

// RegisterHandler registra un nuevo usuario
func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
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
			return
		}
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al registrar usuario",
		})
		return
	}

	// Generar token JWT
	token, err := GenerateToken(idPersona, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al generar token",
		})
		return
	}

	// Crear sesión
	_, err = h.sessionService.CreateSession(context.Background(), idPersona, token)
	if err != nil {
		log.Printf("Error al crear sesión: %v", err)
		// No es crítico, continuamos
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

// LoginHandler inicia sesión de usuario
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	// Verificar credenciales
	idPersona, nombre, err := h.authService.LoginUser(context.Background(), req.Email, req.Contrasena)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Email o contraseña incorrectos",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al verificar credenciales",
		})
		return
	}

	// Generar token JWT
	token, err := GenerateToken(idPersona, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al generar token",
		})
		return
	}

	// Crear sesión
	_, err = h.sessionService.CreateSession(context.Background(), idPersona, token)
	if err != nil {
		log.Printf("Error al crear sesión: %v", err)
		// No es crítico, continuamos
	}

	// Guardar token en cookie (para OAuth compatibility)
	c.SetCookie("auth_token", token, 7*24*3600, "/", "", false, true)

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Login exitoso",
		Data: TokenResponse{
			Token:     token,
			IDPersona: idPersona,
			Email:     req.Email,
			Nombre:    nombre,
		},
	})
}

// GetProfileHandler obtiene el perfil del usuario autenticado
func (h *AuthHandler) GetProfileHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}

	idPersona := idPersonaInterface.(int)

	profile, err := h.userService.GetUserProfile(context.Background(), idPersona)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{
				Success: false,
				Message: "Usuario no encontrado",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener perfil",
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

// UpdateProfileHandler actualiza el perfil del usuario
func (h *AuthHandler) UpdateProfileHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}

	idPersona := idPersonaInterface.(int)

	type UpdateRequest struct {
		Nombre   string  `json:"nombre" binding:"omitempty,min=2"`
		Apellido string  `json:"apellido" binding:"omitempty,min=2"`
		Telefono *string `json:"telefono" binding:"omitempty"`
		Bio      *string `json:"bio" binding:"omitempty,max=500"`
		Avatar   *string `json:"avatar" binding:"omitempty,url"`
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	// Actualizar perfil
	err := h.userService.UpdateUserProfile(context.Background(), idPersona, req.Nombre, req.Apellido, req.Telefono, req.Bio, req.Avatar)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{
				Success: false,
				Message: "Usuario no encontrado",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al actualizar perfil",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseData{
		Success: true,
		Message: "Perfil actualizado correctamente",
	})
}

// SelectRoleHandler permite que el usuario seleccione su rol
func (h *AuthHandler) SelectRoleHandler(c *gin.Context) {
	idPersonaInterface, exists := c.Get("id_persona")
	if !exists {
		c.JSON(http.StatusUnauthorized, ResponseData{
			Success: false,
			Message: "Usuario no autenticado",
		})
		return
	}

	idPersona := idPersonaInterface.(int)

	var req SelectRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseData{
			Success: false,
			Message: "Datos inválidos: " + err.Error(),
		})
		return
	}

	// Obtener ID del rol
	idRol, err := h.userService.GetRoleIDByName(context.Background(), req.Rol)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, ResponseData{
				Success: false,
				Message: "Rol no encontrado",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ResponseData{
			Success: false,
			Message: "Error al obtener rol",
		})
		return
	}

	// Actualizar rol del usuario
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
		Message: "Rol seleccionado correctamente",
	})
}

// SubscribeToPlanHandler permite que el usuario se suscriba a un plan
func (h *AuthHandler) SubscribeToPlanHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ResponseData{
		Success: false,
		Message: "No implementado",
	})
}

// AdminMiddleware verifica si el usuario tiene permisos de administrador
func (h *AuthHandler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Por ahora, dejar pasar a todos. TODO: Implementar verificación de rol
		c.Next()
	}
}

// CreatePlanHandler crea un nuevo plan
func (h *AuthHandler) CreatePlanHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ResponseData{
		Success: false,
		Message: "No implementado",
	})
}

// GetAllPlansHandler obtiene todos los planes
func (h *AuthHandler) GetAllPlansHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ResponseData{
		Success: false,
		Message: "No implementado",
	})
}

// GetPlanByIDHandler obtiene un plan por ID
func (h *AuthHandler) GetPlanByIDHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ResponseData{
		Success: false,
		Message: "No implementado",
	})
}

// UpdatePlanHandler actualiza un plan
func (h *AuthHandler) UpdatePlanHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ResponseData{
		Success: false,
		Message: "No implementado",
	})
}

// DeletePlanHandler elimina un plan
func (h *AuthHandler) DeletePlanHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ResponseData{
		Success: false,
		Message: "No implementado",
	})
}

// AuthMiddleware verifica que el usuario esté autenticado
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			// Intentar obtener del cookie
			tokenString, err := c.Cookie("auth_token")
			if err != nil {
				c.JSON(http.StatusUnauthorized, ResponseData{
					Success: false,
					Message: "Token no proporcionado",
				})
				c.Abort()
				return
			}
			tokenString = tokenString
		}

		// Remover "Bearer " si está presente
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims, err := VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Token inválido o expirado",
			})
			c.Abort()
			return
		}

		// Guardar id_persona en el contexto para que otros handlers puedan acceder
		c.Set("id_persona", claims.IDPersona)
		c.Set("email", claims.Email)

		c.Next()
	}
}
