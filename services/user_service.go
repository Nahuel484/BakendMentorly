package services

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	db *pgxpool.Pool
}

type UserProfile struct {
	IDPersona int
	Nombre    string
	Apellido  string
	Email     string
	IDRol     int
	Rol       string
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

// GetUserProfile obtiene el perfil completo del usuario
func (s *UserService) GetUserProfile(ctx context.Context, idPersona int) (*UserProfile, error) {
	var nombre, apellido, email string
	var idRol int

	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, apellido, email, id_rol FROM tb_persona WHERE id_persona = $1",
		idPersona,
	).Scan(&idPersona, &nombre, &apellido, &email, &idRol)

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
