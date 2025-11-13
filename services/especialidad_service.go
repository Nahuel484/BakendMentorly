package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EspecialidadService struct {
	db *pgxpool.Pool
}

type Especialidad struct {
	IDEspecialidad    int    `json:"id_especialidad"`
	NombreEspecialidad string `json:"nombre_especialidad"`
}

type UserEspecialidad struct {
	IDUsuarioEspecialidad int       `json:"id_usuario_especialidad"`
	IDPersona             int       `json:"id_persona"`
	IDEspecialidad        int       `json:"id_especialidad"`
	NombreEspecialidad    string    `json:"nombre_especialidad"`
	FechaAgregada         time.Time `json:"fecha_agregada"`
}

func NewEspecialidadService(db *pgxpool.Pool) *EspecialidadService {
	return &EspecialidadService{db: db}
}

// GetAllEspecialidades obtiene todas las especialidades disponibles
func (s *EspecialidadService) GetAllEspecialidades(ctx context.Context) ([]Especialidad, error) {
	var especialidades []Especialidad
	query := `SELECT ID_Especialidad, Nombre_Especialidad FROM tb_Especialidad ORDER BY Nombre_Especialidad`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var esp Especialidad
		if err := rows.Scan(&esp.IDEspecialidad, &esp.NombreEspecialidad); err != nil {
			return nil, err
		}
		especialidades = append(especialidades, esp)
	}
	return especialidades, nil
}

// GetEspecialidadByID obtiene una especialidad por ID
func (s *EspecialidadService) GetEspecialidadByID(ctx context.Context, id int) (*Especialidad, error) {
	var esp Especialidad
	query := `SELECT ID_Especialidad, Nombre_Especialidad FROM tb_Especialidad WHERE ID_Especialidad = $1`

	err := s.db.QueryRow(ctx, query, id).
		Scan(&esp.IDEspecialidad, &esp.NombreEspecialidad)

	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &esp, nil
}

// AddEspecialidadToUser asigna una especialidad a un usuario
func (s *EspecialidadService) AddEspecialidadToUser(ctx context.Context, idPersona, idEspecialidad int) (*UserEspecialidad, error) {
	var userEsp UserEspecialidad
	query := `INSERT INTO tb_Usuario_Especialidad (ID_Persona, ID_Especialidad, Fecha_Agregada) 
	          VALUES ($1, $2, $3) 
	          RETURNING ID_Usuario_Especialidad, ID_Persona, ID_Especialidad, Fecha_Agregada`

	err := s.db.QueryRow(ctx, query, idPersona, idEspecialidad, time.Now()).
		Scan(&userEsp.IDUsuarioEspecialidad, &userEsp.IDPersona, &userEsp.IDEspecialidad, &userEsp.FechaAgregada)

	if err != nil {
		return nil, err
	}
	return &userEsp, nil
}

// GetUserEspecialidades obtiene todas las especialidades de un usuario
func (s *EspecialidadService) GetUserEspecialidades(ctx context.Context, idPersona int) ([]UserEspecialidad, error) {
	var especialidades []UserEspecialidad
	query := `SELECT ue.ID_Usuario_Especialidad, ue.ID_Persona, ue.ID_Especialidad, e.Nombre_Especialidad, ue.Fecha_Agregada 
	          FROM tb_Usuario_Especialidad ue 
	          JOIN tb_Especialidad e ON ue.ID_Especialidad = e.ID_Especialidad 
	          WHERE ue.ID_Persona = $1 
	          ORDER BY e.Nombre_Especialidad`

	rows, err := s.db.Query(ctx, query, idPersona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var esp UserEspecialidad
		if err := rows.Scan(&esp.IDUsuarioEspecialidad, &esp.IDPersona, &esp.IDEspecialidad, &esp.NombreEspecialidad, &esp.FechaAgregada); err != nil {
			return nil, err
		}
		especialidades = append(especialidades, esp)
	}
	return especialidades, nil
}

// RemoveUserEspecialidad elimina una especialidad de un usuario
func (s *EspecialidadService) RemoveUserEspecialidad(ctx context.Context, idUsuarioEspecialidad int) error {
	query := `DELETE FROM tb_Usuario_Especialidad WHERE ID_Usuario_Especialidad = $1`
	result, err := s.db.Exec(ctx, query, idUsuarioEspecialidad)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateEspecialidad crea una nueva especialidad (admin)
func (s *EspecialidadService) CreateEspecialidad(ctx context.Context, nombre string) (*Especialidad, error) {
	var esp Especialidad
	query := `INSERT INTO tb_Especialidad (Nombre_Especialidad) 
	          VALUES ($1) 
	          RETURNING ID_Especialidad, Nombre_Especialidad`

	err := s.db.QueryRow(ctx, query, nombre).
		Scan(&esp.IDEspecialidad, &esp.NombreEspecialidad)

	if err != nil {
		return nil, err
	}
	return &esp, nil
}

// DeleteEspecialidad elimina una especialidad (admin)
func (s *EspecialidadService) DeleteEspecialidad(ctx context.Context, id int) error {
	query := `DELETE FROM tb_Especialidad WHERE ID_Especialidad = $1`
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}