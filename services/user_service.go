package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	db           *pgxpool.Pool
	emailService *EmailService
}

type UserProfile struct {
	IDPersona int
	Nombre    string
	Apellido  string
	Email     string
	Telefono  *string // Opcional
	Bio       *string // Opcional
	Avatar    *string // Opcional
	IDRol     int
	Rol       string
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{
		db:           db,
		emailService: NewEmailService(),
	}
}

// GetUserProfile obtiene el perfil completo del usuario
func (s *UserService) GetUserProfile(ctx context.Context, idPersona int) (*UserProfile, error) {
	var nombre, apellido, email string
	var telefono, bio, avatar *string
	var idRol int

	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, apellido, email, telefono, bio, avatar, id_rol FROM tb_persona WHERE id_persona = $1",
		idPersona,
	).Scan(&idPersona, &nombre, &apellido, &email, &telefono, &bio, &avatar, &idRol)

	if err != nil {
		log.Printf("Error al obtener perfil de usuario: %v", err)
		return nil, ErrUserNotFound
	}

	// Mapear el rol
	rol := s.GetRoleNameByID(ctx, idRol)

	return &UserProfile{
		IDPersona: idPersona,
		Nombre:    nombre,
		Apellido:  apellido,
		Email:     email,
		Telefono:  telefono,
		Bio:       bio,
		Avatar:    avatar,
		IDRol:     idRol,
		Rol:       rol,
	}, nil
}

// GetRoleNameByID obtiene el nombre del rol por su ID
func (s *UserService) GetRoleNameByID(ctx context.Context, idRol int) string {
	var nombreRol string
	err := s.db.QueryRow(ctx,
		"SELECT nombre_rol FROM tb_rol WHERE id_rol = $1",
		idRol,
	).Scan(&nombreRol)

	if err != nil {
		return "Sin rol"
	}

	return nombreRol
}

// UpdateUserRole actualiza el rol del usuario
func (s *UserService) UpdateUserRole(ctx context.Context, idPersona int, idRol int) error {
	_, err := s.db.Exec(ctx,
		"UPDATE tb_persona SET id_rol = $1 WHERE id_persona = $2",
		idRol, idPersona,
	)

	if err != nil {
		log.Printf("Error al actualizar rol de usuario: %v", err)
		return err
	}

	return nil
}

// UpdateUserProfile actualiza los campos del perfil de un usuario.
// Solo actualiza los campos que no son cadenas vacías o nil.
func (s *UserService) UpdateUserProfile(ctx context.Context, idPersona int, nombre, apellido string, telefono, bio, avatar *string) error {
	var setClauses []string
	var args []interface{}
	argID := 1

	if nombre != "" {
		setClauses = append(setClauses, fmt.Sprintf("nombre = $%d", argID))
		args = append(args, nombre)
		argID++
	}
	if apellido != "" {
		setClauses = append(setClauses, fmt.Sprintf("apellido = $%d", argID))
		args = append(args, apellido)
		argID++
	}
	if telefono != nil {
		setClauses = append(setClauses, fmt.Sprintf("telefono = $%d", argID))
		args = append(args, *telefono)
		argID++
	}
	if bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio = $%d", argID))
		args = append(args, *bio)
		argID++
	}
	if avatar != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar = $%d", argID))
		args = append(args, *avatar)
		argID++
	}

	// Si no hay campos para actualizar, no hacer nada.
	if len(setClauses) == 0 {
		return nil
	}

	// Construir la consulta final
	query := fmt.Sprintf("UPDATE tb_persona SET %s WHERE id_persona = $%d", strings.Join(setClauses, ", "), argID)
	args = append(args, idPersona)

	// Ejecutar la consulta
	_, err := s.db.Exec(ctx, query, args...)

	if err != nil {
		return err
	}

	// Enviar email de notificación (opcional, de forma asíncrona)
	go s.sendProfileUpdateNotification(ctx, idPersona, nombre)

	return nil
}

// sendProfileUpdateNotification envía una notificación cuando se actualiza el perfil
func (s *UserService) sendProfileUpdateNotification(ctx context.Context, idPersona int, nombre string) {
	var email string
	err := s.db.QueryRow(ctx,
		"SELECT email FROM tb_persona WHERE id_persona = $1",
		idPersona,
	).Scan(&email)

	if err != nil {
		log.Printf("Error al obtener email para notificación: %v", err)
		return
	}

	// Enviar email de confirmación
	err = s.emailService.SendProfileUpdateEmail(email, nombre)
	if err != nil {
		log.Printf("Error al enviar email de actualización de perfil: %v", err)
	}
}

// GetRoleIDByName obtiene el ID del rol por su nombre
func (s *UserService) GetRoleIDByName(ctx context.Context, nombreRol string) (int, error) {
	var idRol int
	err := s.db.QueryRow(ctx,
		"SELECT id_rol FROM tb_rol WHERE nombre_rol = $1",
		nombreRol,
	).Scan(&idRol)

	if err != nil {
		log.Printf("Error al obtener ID de rol: %v", err)
		return 0, err
	}

	return idRol, nil
}

// GetUserByEmail obtiene un usuario por email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*UserProfile, error) {
	var idPersona int
	var nombre, apellido string
	var telefono, bio, avatar *string
	var idRol int

	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, apellido, email, telefono, bio, avatar, id_rol FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona, &nombre, &apellido, &email, &telefono, &bio, &avatar, &idRol)

	if err != nil {
		return nil, ErrUserNotFound
	}

	rol := s.GetRoleNameByID(ctx, idRol)

	return &UserProfile{
		IDPersona: idPersona,
		Nombre:    nombre,
		Apellido:  apellido,
		Email:     email,
		Telefono:  telefono,
		Bio:       bio,
		Avatar:    avatar,
		IDRol:     idRol,
		Rol:       rol,
	}, nil
}
