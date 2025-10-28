package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

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

// ============================================
// MÉTODOS PARA OAUTH
// ============================================

// FindOrCreateOAuthUser busca un usuario existente o crea uno nuevo con OAuth
func (s *AuthService) FindOrCreateOAuthUser(ctx context.Context, oauthUser *OAuthUserInfo) (*User, error) {
	// Primero intentar buscar por email en tabla users
	user, err := s.GetUserByEmail(ctx, oauthUser.Email)
	if err == nil {
		// Usuario existe, actualizar proveedor si es necesario
		err = s.updateOAuthProvider(ctx, user.ID, oauthUser)
		if err != nil {
			return nil, fmt.Errorf("error updating oauth provider: %w", err)
		}
		return user, nil
	}

	// Si no existe, crear nuevo usuario en tabla users
	newUser := &User{
		Email:    oauthUser.Email,
		Name:     oauthUser.Name,
		Avatar:   oauthUser.Avatar,
		Provider: string(oauthUser.Provider),
		OAuthID:  oauthUser.ID,
	}

	err = s.db.QueryRow(ctx,
		`INSERT INTO users (email, name, avatar, provider, oauth_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, email, name, avatar, provider, oauth_id, created_at, updated_at`,
		newUser.Email, newUser.Name, newUser.Avatar, newUser.Provider, newUser.OAuthID, time.Now(), time.Now(),
	).Scan(&newUser.ID, &newUser.Email, &newUser.Name, &newUser.Avatar, &newUser.Provider, &newUser.OAuthID, &newUser.CreatedAt, &newUser.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("error creating oauth user: %w", err)
	}

	return newUser, nil
}

// updateOAuthProvider actualiza el proveedor OAuth de un usuario existente
func (s *AuthService) updateOAuthProvider(ctx context.Context, userID int64, oauthUser *OAuthUserInfo) error {
	_, err := s.db.Exec(ctx,
		`UPDATE users SET provider = $1, oauth_id = $2, avatar = COALESCE($3, avatar), updated_at = $4
		 WHERE id = $5`,
		string(oauthUser.Provider), oauthUser.ID, oauthUser.Avatar, time.Now(), userID,
	)
	return err
}

// GetUserByEmail obtiene un usuario por email de la tabla users
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, name, avatar, provider, oauth_id, created_at, updated_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Avatar, &user.Provider, &user.OAuthID, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	return user, nil
}
