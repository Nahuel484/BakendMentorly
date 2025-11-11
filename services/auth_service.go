package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// AuthService maneja la autenticación
type AuthService struct {
	db *pgxpool.Pool
}

// NewAuthService crea una nueva instancia del servicio de autenticación
func NewAuthService(db *pgxpool.Pool) *AuthService {
	return &AuthService{db: db}
}

// RegisterUser registra un nuevo usuario
func (s *AuthService) RegisterUser(ctx context.Context, nombre string, apellido string, email string, hashedPassword string) (int, error) {
	var idPersona int

	// Verificar si el email ya existe
	err := s.db.QueryRow(ctx,
		"SELECT id_persona FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona)

	if err == nil {
		return 0, ErrEmailAlreadyExists
	}

	if err != pgx.ErrNoRows {
		return 0, err
	}

	// Insertar nuevo usuario en tb_persona
	err = s.db.QueryRow(ctx,
		`INSERT INTO tb_persona (nombre, apellido, email, contrasena, fecha_registro)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id_persona`,
		nombre, apellido, email, hashedPassword, time.Now(),
	).Scan(&idPersona)

	if err != nil {
		return 0, err
	}

	return idPersona, nil
}

// LoginUser verifica las credenciales del usuario
func (s *AuthService) LoginUser(ctx context.Context, email string, password string) (int, string, error) {
	var idPersona int
	var nombre string
	var hashedPassword string

	// Obtener usuario de tb_persona
	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, contrasena FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona, &nombre, &hashedPassword)

	if err == pgx.ErrNoRows {
		return 0, "", ErrInvalidCredentials
	}

	if err != nil {
		return 0, "", err
	}

	// Si el password está vacío (login OAuth), y el hash también, es un login válido.
	if password == "" && hashedPassword == "" {
		return idPersona, nombre, nil
	}

	// Verificar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return 0, "", ErrInvalidCredentials
	}

	return idPersona, nombre, nil
}

// HashPassword genera el hash de una contraseña
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
