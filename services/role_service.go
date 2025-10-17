package services

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleService struct {
	db *pgxpool.Pool
}

type Role struct {
	IDRol       int
	NombreRol   string
	Descripcion string
}

func NewRoleService(db *pgxpool.Pool) *RoleService {
	return &RoleService{db: db}
}

// GetAllRoles obtiene todos los roles disponibles
func (s *RoleService) GetAllRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.db.Query(ctx,
		"SELECT id_rol, nombre_rol, descripcion FROM tb_rol ORDER BY id_rol",
	)
	if err != nil {
		log.Printf("Error al obtener roles: %v", err)
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		err := rows.Scan(&role.IDRol, &role.NombreRol, &role.Descripcion)
		if err != nil {
			log.Printf("Error al escanear rol: %v", err)
			continue
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetRoleByID obtiene un rol por su ID
func (s *RoleService) GetRoleByID(ctx context.Context, idRol int) (*Role, error) {
	var role Role
	err := s.db.QueryRow(ctx,
		"SELECT id_rol, nombre_rol, descripcion FROM tb_rol WHERE id_rol = $1",
		idRol,
	).Scan(&role.IDRol, &role.NombreRol, &role.Descripcion)

	if err != nil {
		log.Printf("Error al obtener rol: %v", err)
		return nil, ErrRoleNotFound
	}

	return &role, nil
}

// GetRoleByName obtiene un rol por su nombre
func (s *RoleService) GetRoleByName(ctx context.Context, nombreRol string) (*Role, error) {
	var role Role
	err := s.db.QueryRow(ctx,
		"SELECT id_rol, nombre_rol, descripcion FROM tb_rol WHERE nombre_rol = $1",
		nombreRol,
	).Scan(&role.IDRol, &role.NombreRol, &role.Descripcion)

	if err != nil {
		log.Printf("Error al obtener rol por nombre: %v", err)
		return nil, ErrRoleNotFound
	}

	return &role, nil
}