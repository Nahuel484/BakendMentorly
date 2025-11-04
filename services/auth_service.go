package services

import (
	"context" // ¡Importante!
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// (Asumiendo que estos errores están definidos en errors_service.go)
// var (
// 	ErrEmailAlreadyExists = errors.New("el email ya existe")
// 	ErrInvalidCredentials = errors.New("credenciales inválidas")
// 	ErrUserNotFound       = errors.New("usuario no encontrado")
// )

// AuthService maneja la autenticación
type AuthService struct {
	db *pgxpool.Pool
}

// User estructura para autenticación tradicional y OAuth
type User struct {
	ID        int64
	Email     string
	Name      string
	Avatar    string
	Provider  string
	OAuthID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OAuthUserInfo ya debería estar definida en otro archivo

// NewAuthService crea una nueva instancia del servicio de autenticación
func NewAuthService(db *pgxpool.Pool) *AuthService {
	return &AuthService{db: db}
}

// RegisterUser registra un nuevo usuario (¡CORREGIDO!)
func (s *AuthService) RegisterUser(ctx context.Context, nombre, apellido, email string, hashedPassword string) (int, error) {
	var idPersona int

	// Verificar si el email ya existe
	err := s.db.QueryRow(ctx,
		"SELECT id_persona FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona)

	if err == nil {
		return 0, ErrEmailAlreadyExists // Asumiendo que ErrEmailAlreadyExists está en errors_service.go
	}

	if err != pgx.ErrNoRows {
		// Un error inesperado de la base de datos
		return 0, err
	}

	// ============================================
	// ¡¡AQUÍ ESTÁ EL CAMBIO!!
	// Añadimos "id_rol" a la consulta y "0" a los valores
	// ============================================
	err = s.db.QueryRow(ctx,
		`INSERT INTO tb_persona (nombre, apellido, email, contrasena, fecha_registro, id_rol)
		  VALUES ($1, $2, $3, $4, $5, $6)
		  RETURNING id_persona`,
		nombre, apellido, email, hashedPassword, time.Now(), 0, // <-- Se añade el 0 para el id_rol
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
		return 0, "", ErrInvalidCredentials // Asumiendo que está en errors_service.go
	}

	if err != nil {
		return 0, "", err
	}

	// Verificar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return 0, "", ErrInvalidCredentials // Asumiendo que está en errors_service.go
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

// ============================================
// MÉTODOS PARA OAUTH (Unificados con tb_persona)
// ============================================

// FindOrCreateOAuthUser busca un usuario existente o crea uno nuevo con OAuth
func (s *AuthService) FindOrCreateOAuthUser(ctx context.Context, oauthUser *OAuthUserInfo) (*User, error) {
	// Primero intentar buscar por email en tabla tb_persona
	user, err := s.GetUserByEmail(ctx, oauthUser.Email)
	if err == nil {
		// Usuario existe, actualizar proveedor si es necesario
		err = s.updateOAuthProvider(ctx, user.ID, oauthUser)
		if err != nil {
			return nil, fmt.Errorf("error updating oauth provider: %w", err)
		}
		return user, nil
	}
	// Si el error no es "User Not Found", es un error real
	if err != ErrUserNotFound { // Asumiendo que está en errors_service.go
		return nil, err
	}

	// Si no existe (ErrUserNotFound), crear nuevo usuario CON ROL 0
	newUser := &User{
		Email:    oauthUser.Email,
		Name:     oauthUser.Name,
		Avatar:   oauthUser.Avatar,
		Provider: string(oauthUser.Provider),
		OAuthID:  oauthUser.ID,
	}

	// ============================================
	// ¡¡AQUÍ TAMBIÉN AÑADIMOS EL ROL 0!!
	// ============================================
	err = s.db.QueryRow(ctx,
		`INSERT INTO tb_persona (email, nombre, avatar, provider, oauth_id, fecha_registro, id_rol)
		  VALUES ($1, $2, $3, $4, $5, $6, $7)
		  RETURNING id_persona, email, nombre, avatar, provider, oauth_id, fecha_registro, fecha_registro`,
		newUser.Email, newUser.Name, newUser.Avatar, newUser.Provider, newUser.OAuthID, time.Now(), 0, // <-- Se añade el 0
	).Scan(
		&newUser.ID,
		&newUser.Email,
		&newUser.Name,
		&newUser.Avatar,
		&newUser.Provider,
		&newUser.OAuthID,
		&newUser.CreatedAt,
		&newUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating oauth user: %w", err)
	}

	return newUser, nil
}

// updateOAuthProvider actualiza el proveedor OAuth de un usuario existente
func (s *AuthService) updateOAuthProvider(ctx context.Context, userID int64, oauthUser *OAuthUserInfo) error {
	_, err := s.db.Exec(ctx,
		`UPDATE tb_persona SET provider = $1, oauth_id = $2, avatar = COALESCE($3, avatar)
		  WHERE id_persona = $4`,
		string(oauthUser.Provider), oauthUser.ID, oauthUser.Avatar, userID,
	)
	return err
}

// GetUserByEmail obtiene un usuario por email de la tabla tb_persona
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id_persona, email, nombre, avatar, provider, oauth_id, fecha_registro 
		 FROM tb_persona WHERE email = $1`,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Avatar,
		&user.Provider,
		&user.OAuthID,
		&user.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound // Asumiendo que está en errors_service.go
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	user.UpdatedAt = user.CreatedAt

	return user, nil
}
