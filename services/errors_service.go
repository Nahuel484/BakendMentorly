package services

import "errors"

var (
	ErrEmailAlreadyExists   = errors.New("el email ya está registrado")
	ErrInvalidCredentials   = errors.New("email o contraseña incorrectos")
	ErrUserNotFound         = errors.New("usuario no encontrado")
	ErrRoleNotFound         = errors.New("rol no encontrado")
	ErrInvalidRole          = errors.New("rol inválido")
)