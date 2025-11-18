package services

import (
	"context"
	"log"
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

	log.Printf("🔐 [RegisterUser] Intentando registrar: %s %s (%s)", nombre, apellido, email)
	log.Printf("🔐 [RegisterUser] Hash length: %d", len(hashedPassword))

	// Verificar si el email ya existe
	err := s.db.QueryRow(ctx,
		"SELECT id_persona FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona)

	if err == nil {
		log.Printf("❌ [RegisterUser] Email ya existe: %s", email)
		return 0, ErrEmailAlreadyExists
	}

	if err != pgx.ErrNoRows {
		log.Printf("❌ [RegisterUser] Error en query: %v", err)
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
		log.Printf("❌ [RegisterUser] Error al insertar: %v", err)
		return 0, err
	}

	log.Printf("✅ [RegisterUser] Usuario creado con ID: %d", idPersona)
	return idPersona, nil
}

// LoginUser verifica las credenciales del usuario
func (s *AuthService) LoginUser(ctx context.Context, email string, password string) (int, string, error) {
	var idPersona int
	var nombre string
	var hashedPassword string

	log.Printf("🔐 [LoginUser] Intento de login para: %s", email)
	log.Printf("🔐 [LoginUser] Contraseña recibida (length): %d", len(password))
	log.Printf("🔐 [LoginUser] Contraseña recibida (primeros 10 chars): %s", truncateString(password, 10))

	// Obtener usuario de tb_persona
	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, contrasena FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona, &nombre, &hashedPassword)

	if err == pgx.ErrNoRows {
		log.Printf("❌ [LoginUser] Usuario no encontrado: %s", email)
		return 0, "", ErrInvalidCredentials
	}

	if err != nil {
		log.Printf("❌ [LoginUser] Error en query: %v", err)
		return 0, "", err
	}

	log.Printf("✅ [LoginUser] Usuario encontrado: %s (ID: %d)", nombre, idPersona)
	log.Printf("🔐 [LoginUser] Hash almacenado (length): %d", len(hashedPassword))
	log.Printf("🔐 [LoginUser] Hash almacenado (primeros 30 chars): %s", truncateString(hashedPassword, 30))

	// Si el password está vacío (login OAuth), y el hash también, es un login válido.
	if password == "" && hashedPassword == "" {
		log.Printf("✅ [LoginUser] Login OAuth válido")
		return idPersona, nombre, nil
	}

	// Si el hash está vacío pero se proporciona contraseña
	if hashedPassword == "" {
		log.Printf("❌ [LoginUser] Usuario OAuth intentando login con contraseña")
		return 0, "", ErrInvalidCredentials
	}

	// Verificar contraseña
	log.Printf("🔐 [LoginUser] Comparando contraseñas...")
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		log.Printf("❌ [LoginUser] Contraseña incorrecta: %v", err)
		log.Printf("❌ [LoginUser] Email: %s", email)
		log.Printf("❌ [LoginUser] Password length: %d", len(password))
		log.Printf("❌ [LoginUser] Hash length: %d", len(hashedPassword))

		// Debug adicional
		log.Printf("🔍 [DEBUG] Password bytes: %v", []byte(password)[:min(len(password), 20)])
		log.Printf("🔍 [DEBUG] Hash preview: %s", hashedPassword[:min(len(hashedPassword), 50)])

		return 0, "", ErrInvalidCredentials
	}

	log.Printf("✅ [LoginUser] Login exitoso para: %s", email)
	return idPersona, nombre, nil
}

// HashPassword genera el hash de una contraseña
func (s *AuthService) HashPassword(password string) (string, error) {
	log.Printf("🔐 [HashPassword] Generando hash para contraseña (length: %d)", len(password))

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ [HashPassword] Error: %v", err)
		return "", err
	}

	hashedString := string(hashedPassword)
	log.Printf("✅ [HashPassword] Hash generado (length: %d)", len(hashedString))
	log.Printf("🔐 [HashPassword] Hash preview: %s", truncateString(hashedString, 30))

	return hashedString, nil
}

// Helper function
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
