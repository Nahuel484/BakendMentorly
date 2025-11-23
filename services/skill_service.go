package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SkillService struct {
	db *pgxpool.Pool
}

type Skill struct {
	IDHabilidad     int    `json:"id_habilidad"`
	NombreHabilidad string `json:"nombre_habilidad"`
	Descripcion     string `json:"descripcion,omitempty"`
	Categoria       string `json:"categoria,omitempty"`
}

type UserSkill struct {
	IDUsuarioHabilidad   int       `json:"id_usuario_habilidad"`
	IDPersona            int       `json:"id_persona"`
	IDHabilidad          int       `json:"id_habilidad"`
	NombreHabilidad      string    `json:"nombre_habilidad"`
	NivelHabilidad       string    `json:"nivel_habilidad"` // "beginner", "intermediate", "advanced"
	DescripcionHabilidad *string   `json:"descripcion_habilidad,omitempty"`
	FechaAgregada        time.Time `json:"fecha_agregada"`
}

func NewSkillService(db *pgxpool.Pool) *SkillService {
	return &SkillService{db: db}
}

// CreateSkill crea una nueva habilidad
func (s *SkillService) CreateSkill(ctx context.Context, nombre string) (*Skill, error) {
	var skill Skill
	query := `INSERT INTO tb_habilidad (nombre_habilidad) 
	          VALUES ($1) 
	          RETURNING id_habilidad, nombre_habilidad`

	err := s.db.QueryRow(ctx, query, nombre).
		Scan(&skill.IDHabilidad, &skill.NombreHabilidad)

	if err != nil {
		return nil, err
	}
	return &skill, nil
}

// GetAllSkills obtiene todas las habilidades disponibles
func (s *SkillService) GetAllSkills(ctx context.Context) ([]Skill, error) {
	var skills []Skill
	query := `SELECT id_habilidad, nombre_habilidad FROM tb_habilidad ORDER BY nombre_habilidad`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var skill Skill
		if err := rows.Scan(&skill.IDHabilidad, &skill.NombreHabilidad); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// GetSkillByID obtiene una habilidad por ID
func (s *SkillService) GetSkillByID(ctx context.Context, id int) (*Skill, error) {
	var skill Skill
	query := `SELECT id_habilidad, nombre_habilidad FROM tb_habilidad WHERE id_habilidad = $1`

	err := s.db.QueryRow(ctx, query, id).
		Scan(&skill.IDHabilidad, &skill.NombreHabilidad)

	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

// AddSkillToUser asigna una habilidad a un usuario con nivel y descripción
func (s *SkillService) AddSkillToUser(ctx context.Context, idPersona, idHabilidad int, nivelHabilidad string, descripcion *string) (*UserSkill, error) {
	var userSkill UserSkill
	query := `INSERT INTO tb_usuario_habilidad (id_persona, id_habilidad, nivel_habilida, descripcion_habilidad, fecha_agregada) 
	          VALUES ($1, $2, $3, $4, $5) 
	          RETURNING id_usuario_habilidad, id_persona, id_habilidad, nivel_habilida, descripcion_habilidad, fecha_agregada`

	err := s.db.QueryRow(ctx, query, idPersona, idHabilidad, nivelHabilidad, descripcion, time.Now()).
		Scan(&userSkill.IDUsuarioHabilidad, &userSkill.IDPersona, &userSkill.IDHabilidad, &userSkill.NivelHabilidad, &userSkill.DescripcionHabilidad, &userSkill.FechaAgregada)

	if err != nil {
		return nil, err
	}
	return &userSkill, nil
}

// GetUserSkills obtiene todas las habilidades de un usuario con descripción
func (s *SkillService) GetUserSkills(ctx context.Context, idPersona int) ([]UserSkill, error) {
	var skills []UserSkill
	query := `SELECT uh.id_usuario_habilidad, uh.id_persona, uh.id_habilidad, h.nombre_habilidad, 
	                 uh.nivel_habilida, uh.descripcion_habilidad, uh.fecha_agregada 
	          FROM tb_usuario_habilidad uh 
	          JOIN tb_habilidad h ON uh.id_habilidad = h.id_habilidad 
	          WHERE uh.id_persona = $1 
	          ORDER BY h.nombre_habilidad`

	rows, err := s.db.Query(ctx, query, idPersona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var skill UserSkill
		if err := rows.Scan(&skill.IDUsuarioHabilidad, &skill.IDPersona, &skill.IDHabilidad,
			&skill.NombreHabilidad, &skill.NivelHabilidad, &skill.DescripcionHabilidad, &skill.FechaAgregada); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// RemoveUserSkill elimina una habilidad de un usuario
func (s *SkillService) RemoveUserSkill(ctx context.Context, idUsuarioHabilidad int) error {
	query := `DELETE FROM tb_usuario_habilidad WHERE id_usuario_habilidad = $1`
	result, err := s.db.Exec(ctx, query, idUsuarioHabilidad)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateUserSkill actualiza el nivel y descripción de una habilidad del usuario
func (s *SkillService) UpdateUserSkill(ctx context.Context, idUsuarioHabilidad int, nivelHabilidad string, descripcion *string) error {
	query := `UPDATE tb_usuario_habilidad 
	          SET nivel_habilida = $1, descripcion_habilidad = $2 
	          WHERE id_usuario_habilidad = $3`
	result, err := s.db.Exec(ctx, query, nivelHabilidad, descripcion, idUsuarioHabilidad)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateUserSkillLevel actualiza solo el nivel de dominio de una habilidad (mantener compatibilidad)
func (s *SkillService) UpdateUserSkillLevel(ctx context.Context, idUsuarioHabilidad int, nivelHabilidad string) error {
	query := `UPDATE tb_usuario_habilidad SET nivel_habilida = $1 WHERE id_usuario_habilidad = $2`
	result, err := s.db.Exec(ctx, query, nivelHabilidad, idUsuarioHabilidad)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
