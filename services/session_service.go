package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionService struct {
	db *pgxpool.Pool
}

type Session struct {
	IDSesion        string    `json:"id_sesion"`
	IDPersona       int       `json:"id_persona"`
	TokenJWT        string    `json:"token_jwt"`
	FechaCreacion   time.Time `json:"fecha_creacion"`
	FechaExpiracion time.Time `json:"fecha_expiracion"`
	Activa          bool      `json:"activa"`
}

func NewSessionService(db *pgxpool.Pool) *SessionService {
	return &SessionService{db: db}
}

// CreateSession crea una nueva sesión para un usuario
func (s *SessionService) CreateSession(ctx context.Context, idPersona int, tokenJWT string) (*Session, error) {
	idSesion := generateSessionID()
	fechaCreacion := time.Now()
	fechaExpiracion := fechaCreacion.AddDate(0, 0, 7) // 7 días

	query := `INSERT INTO tb_Sesion (ID_Sesion, ID_Persona, Token_JWT, Fecha_Creacion, Fecha_Expiracion, Activa) 
	          VALUES ($1, $2, $3, $4, $5, $6) 
	          RETURNING ID_Sesion, ID_Persona, Token_JWT, Fecha_Creacion, Fecha_Expiracion, Activa`

	var session Session
	err := s.db.QueryRow(ctx, query, idSesion, idPersona, tokenJWT, fechaCreacion, fechaExpiracion, true).
		Scan(&session.IDSesion, &session.IDPersona, &session.TokenJWT, &session.FechaCreacion, &session.FechaExpiracion, &session.Activa)

	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSessionByToken obtiene una sesión activa por token
func (s *SessionService) GetSessionByToken(ctx context.Context, tokenJWT string) (*Session, error) {
	var session Session
	query := `SELECT ID_Sesion, ID_Persona, Token_JWT, Fecha_Creacion, Fecha_Expiracion, Activa 
	          FROM tb_Sesion 
	          WHERE Token_JWT = $1 AND Activa = true AND Fecha_Expiracion > NOW()`

	err := s.db.QueryRow(ctx, query, tokenJWT).
		Scan(&session.IDSesion, &session.IDPersona, &session.TokenJWT, &session.FechaCreacion, &session.FechaExpiracion, &session.Activa)

	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// LogoutSession cierra una sesión marcándola como inactiva
func (s *SessionService) LogoutSession(ctx context.Context, tokenJWT string) error {
	query := `UPDATE tb_Sesion SET Activa = false WHERE Token_JWT = $1`
	result, err := s.db.Exec(ctx, query, tokenJWT)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// LogoutAllSessions cierra todas las sesiones de un usuario
func (s *SessionService) LogoutAllSessions(ctx context.Context, idPersona int) error {
	query := `UPDATE tb_Sesion SET Activa = false WHERE ID_Persona = $1`
	_, err := s.db.Exec(ctx, query, idPersona)
	return err
}

// generateSessionID genera un ID único para la sesión
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
