package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"mentorly-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	authService *services.AuthService
	userService *services.UserService
	roleService *services.RoleService
}

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

type ResponseData struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	IDPersona int    `json:"id_persona"`
	Email     string `json:"email"`
	Nombre    string `json:"nombre"`
}

type Claims struct {
	IDPersona int    `json:"id_persona"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{
		authService: services.NewAuthService(db),
		userService: services.NewUserService(db),
		roleService: services.NewRoleService(db),
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
		if err == services.ErrEmailAlreadyExists {
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

// AuthMiddleware - Middleware para proteger rutas
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Token requerido",
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Formato de token inválido",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ResponseData{
				Success: false,
				Message: "Token inválido o expirado",
			})
			c.Abort()
			return
		}

		c.Set("id_persona", claims.IDPersona)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// JWT Functions

func GenerateToken(idPersona int, email string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "tu_clave_super_secreta_cambiar_en_produccion"
	}

	expirationTime := time.Now().Add(7 * 24 * time.Hour)

	claims := &Claims{
		IDPersona: idPersona,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyToken(tokenString string) (*Claims, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "tu_clave_super_secreta_cambiar_en_produccion"
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token inválido")
	}

	return claims, nil
}
