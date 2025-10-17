package services

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db *pgxpool.Pool
}

func NewAuthService(db *pgxpool.Pool) *AuthService {
	return &AuthService{db: db}
}

// RegisterUser crea un nuevo usuario en la base de datos
func (s *AuthService) RegisterUser(ctx context.Context, nombre, apellido, email string, hashedPassword []byte) (int, error) {
	// Verificar si el email ya existe
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM tb_persona WHERE email = $1)",
		email,
	).Scan(&exists)
	if err != nil {
		log.Printf("Error al verificar email: %v", err)
		return 0, err
	}

	if exists {
		return 0, ErrEmailAlreadyExists
	}

	// Obtener el id_rol de "Usuario"
	var idRolUsuario int
	err = s.db.QueryRow(ctx,
		"SELECT id_rol FROM tb_rol WHERE nombre_rol = 'Usuario'",
	).Scan(&idRolUsuario)
	if err != nil {
		log.Printf("Error al obtener rol Usuario: %v", err)
		return 0, err
	}

	// Insertar usuario
	_, err = s.db.Exec(ctx,
		`INSERT INTO tb_persona (nombre, apellido, email, contrasena, fecha_registro, id_rol)
		 VALUES ($1, $2, $3, $4, CURRENT_DATE, $5)`,
		nombre, apellido, email, string(hashedPassword), idRolUsuario,
	)
	if err != nil {
		log.Printf("Error al insertar usuario: %v", err)
		return 0, err
	}

	// Obtener el ID del usuario recién creado
	var idPersona int
	err = s.db.QueryRow(ctx,
		"SELECT id_persona FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona)
	if err != nil {
		log.Printf("Error al obtener ID de usuario: %v", err)
		return 0, err
	}

	return idPersona, nil
}

// LoginUser verifica las credenciales del usuario
func (s *AuthService) LoginUser(ctx context.Context, email, password string) (int, string, error) {
	var idPersona int
	var nombre string
	var hashedPassword string

	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, contrasena FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona, &nombre, &hashedPassword)

	if err != nil {
		return 0, "", ErrInvalidCredentials
	}

	// Verificar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return 0, "", ErrInvalidCredentials
	}

	return idPersona, nombre, nil
}

// HashPassword genera un hash bcrypt de la contraseña
func (s *AuthService) HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}