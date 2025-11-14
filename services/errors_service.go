package services

import "errors"

// Errores comunes del sistema
var (
	ErrNotFound           = errors.New("recurso no encontrado")
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrUnauthorized       = errors.New("no autorizado")
	ErrInvalidData        = errors.New("datos inválidos")
	ErrDuplicateKey       = errors.New("registro duplicado")
	ErrEmailAlreadyExists = errors.New("el email ya está registrado")
	ErrInvalidCredentials = errors.New("credenciales inválidas")
	ErrRoleNotFound       = errors.New("rol no encontrado")
	ErrInvalidToken       = errors.New("token inválido")
	ErrExpiredToken       = errors.New("token expirado")
	ErrInvalidPassword    = errors.New("contraseña inválida")
	ErrSessionNotFound    = errors.New("sesión no encontrada")
	ErrPermissionDenied   = errors.New("permiso denegado")
)
