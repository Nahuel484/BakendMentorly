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
	var idRol *int

	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, apellido, email, telefono, bio, avatar, id_rol FROM tb_persona WHERE id_persona = $1",
		idPersona,
	).Scan(&idPersona, &nombre, &apellido, &email, &telefono, &bio, &avatar, &idRol)

	if err != nil {
		log.Printf("❌ [GetUserProfile] Error al obtener perfil: %v", err)
		return nil, ErrUserNotFound
	}

	// Mapear el rol (manejar NULL)
	var rol string
	var rolID int
	if idRol != nil {
		rolID = *idRol
		rol = s.GetRoleNameByID(ctx, rolID)
		log.Printf("✅ [GetUserProfile] Usuario %s tiene rol: %s (ID: %d)", email, rol, rolID)
	} else {
		rolID = 0
		rol = ""
		log.Printf("⚠️  [GetUserProfile] Usuario %s NO tiene rol asignado", email)
	}

	return &UserProfile{
		IDPersona: idPersona,
		Nombre:    nombre,
		Apellido:  apellido,
		Email:     email,
		Telefono:  telefono,
		Bio:       bio,
		Avatar:    avatar,
		IDRol:     rolID,
		Rol:       rol,
	}, nil
}

// GetRoleNameByID obtiene el nombre del rol por su ID
func (s *UserService) GetRoleNameByID(ctx context.Context, idRol int) string {
	if idRol == 0 {
		return ""
	}

	var nombreRol string
	err := s.db.QueryRow(ctx,
		"SELECT nombre_rol FROM tb_rol WHERE id_rol = $1",
		idRol,
	).Scan(&nombreRol)

	if err != nil {
		log.Printf("⚠️  [GetRoleNameByID] No se encontró rol con ID %d: %v", idRol, err)
		return ""
	}

	return nombreRol
}

// UpdateUserRole actualiza el rol del usuario
func (s *UserService) UpdateUserRole(ctx context.Context, idPersona int, idRol int) error {
	log.Printf("🎭 [UpdateUserRole] Actualizando rol del usuario %d a rol %d", idPersona, idRol)

	_, err := s.db.Exec(ctx,
		"UPDATE tb_persona SET id_rol = $1 WHERE id_persona = $2",
		idRol, idPersona,
	)

	if err != nil {
		log.Printf("❌ [UpdateUserRole] Error al actualizar rol: %v", err)
		return err
	}

	log.Printf("✅ [UpdateUserRole] Rol actualizado correctamente")
	return nil
}

// UpdateUserProfile actualiza los campos del perfil de un usuario.
// Solo actualiza los campos que no son cadenas vacías o nil.
func (s *UserService) UpdateUserProfile(ctx context.Context, idPersona int, nombre, apellido string, telefono, bio, avatar *string) error {
	var setClauses []string
	var args []interface{}
	argID := 1

	log.Printf("🔄 [UpdateUserProfile] ID: %d", idPersona)

	if nombre != "" {
		setClauses = append(setClauses, fmt.Sprintf("nombre = $%d", argID))
		args = append(args, nombre)
		log.Printf("  ✏️  nombre: '%s'", nombre)
		argID++
	}
	if apellido != "" {
		setClauses = append(setClauses, fmt.Sprintf("apellido = $%d", argID))
		args = append(args, apellido)
		log.Printf("  ✏️  apellido: '%s'", apellido)
		argID++
	}
	if telefono != nil {
		setClauses = append(setClauses, fmt.Sprintf("telefono = $%d", argID))
		args = append(args, *telefono)
		log.Printf("  ✏️  telefono: '%s'", *telefono)
		argID++
	}
	if bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio = $%d", argID))
		args = append(args, *bio)
		log.Printf("  ✏️  bio: '%s'", *bio)
		argID++
	}
	if avatar != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar = $%d", argID))
		args = append(args, *avatar)
		log.Printf("  ✏️  avatar: '%s'", *avatar)
		argID++
	}

	if len(setClauses) == 0 {
		log.Printf("⚠️  [UpdateUserProfile] No hay campos para actualizar")
		return nil
	}

	query := fmt.Sprintf("UPDATE tb_persona SET %s WHERE id_persona = $%d", strings.Join(setClauses, ", "), argID)
	args = append(args, idPersona)

	log.Printf("📝 [SQL] %s", query)
	log.Printf("📝 [Args] %v", args)

	result, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		log.Printf("❌ [UpdateUserProfile] Error: %v", err)
		return fmt.Errorf("error al actualizar perfil: %w", err)
	}

	rowsAffected := result.RowsAffected()
	log.Printf("✅ [UpdateUserProfile] Filas actualizadas: %d", rowsAffected)

	if rowsAffected == 0 {
		return fmt.Errorf("usuario no encontrado")
	}

	// Enviar email de notificación (asíncrono)
	go func() {
		bgCtx := context.Background()
		s.sendProfileUpdateNotification(bgCtx, idPersona, nombre)
	}()

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
		log.Printf("❌ [GetRoleIDByName] Error al obtener ID de rol '%s': %v", nombreRol, err)
		return 0, err
	}

	return idRol, nil
}

// GetUserByEmail obtiene un usuario por email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*UserProfile, error) {
	var idPersona int
	var nombre, apellido string
	var telefono, bio, avatar *string
	var idRol *int

	err := s.db.QueryRow(ctx,
		"SELECT id_persona, nombre, apellido, email, telefono, bio, avatar, id_rol FROM tb_persona WHERE email = $1",
		email,
	).Scan(&idPersona, &nombre, &apellido, &email, &telefono, &bio, &avatar, &idRol)

	if err != nil {
		return nil, ErrUserNotFound
	}

	// Mapear el rol (manejar NULL)
	var rol string
	var rolID int
	if idRol != nil {
		rolID = *idRol
		rol = s.GetRoleNameByID(ctx, rolID)
	} else {
		rolID = 0
		rol = ""
	}

	return &UserProfile{
		IDPersona: idPersona,
		Nombre:    nombre,
		Apellido:  apellido,
		Email:     email,
		Telefono:  telefono,
		Bio:       bio,
		Avatar:    avatar,
		IDRol:     rolID,
		Rol:       rol,
	}, nil
}

type SimpleUser struct {
	IDPersona int     `json:"id_persona"`
	Nombre    string  `json:"nombre"`
	Apellido  string  `json:"apellido"`
	Email     string  `json:"email"`
	Bio       *string `json:"bio,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	Rol       string  `json:"rol"`
}

func (s *UserService) ListUsersByRole(ctx context.Context, roleName string) ([]SimpleUser, error) {
	var idRol int
	err := s.db.QueryRow(ctx,
		"SELECT id_rol FROM tb_rol WHERE nombre_rol = $1",
		roleName,
	).Scan(&idRol)
	if err != nil {
		return nil, fmt.Errorf("rol no encontrado: %w", err)
	}

	rows, err := s.db.Query(ctx, `
        SELECT id_persona, nombre, apellido, email, bio, avatar
        FROM tb_persona
        WHERE id_rol = $1`,
		idRol,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener usuarios: %w", err)
	}
	defer rows.Close()

	var users []SimpleUser
	for rows.Next() {
		var u SimpleUser
		if err := rows.Scan(&u.IDPersona, &u.Nombre, &u.Apellido, &u.Email, &u.Bio, &u.Avatar); err != nil {
			return nil, err
		}
		u.Rol = roleName
		users = append(users, u)
	}
	return users, nil
}
