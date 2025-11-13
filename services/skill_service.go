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
	IDUsuarioHabilidad int       `json:"id_usuario_habilidad"`
	IDPersona          int       `json:"id_persona"`
	IDHabilidad        int       `json:"id_habilidad"`
	NombreHabilidad    string    `json:"nombre_habilidad"`
	NivelDominio       string    `json:"nivel_dominio"` // "beginner", "intermediate", "advanced"
	FechaAgregada      time.Time `json:"fecha_agregada"`
}

func NewSkillService(db *pgxpool.Pool) *SkillService {
	return &SkillService{db: db}
}

// CreateSkill crea una nueva habilidad
func (s *SkillService) CreateSkill(ctx context.Context, nombre string) (*Skill, error) {
	var skill Skill
	query := `INSERT INTO tb_Habilidad (Nombre_Habilidad) 
	          VALUES ($1) 
	          RETURNING ID_Habilidad, Nombre_Habilidad`

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
	query := `SELECT ID_Habilidad, Nombre_Habilidad FROM tb_Habilidad ORDER BY Nombre_Habilidad`

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
	query := `SELECT ID_Habilidad, Nombre_Habilidad FROM tb_Habilidad WHERE ID_Habilidad = $1`

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

// AddSkillToUser asigna una habilidad a un usuario
func (s *SkillService) AddSkillToUser(ctx context.Context, idPersona, idHabilidad int, nivelDominio string) (*UserSkill, error) {
	var userSkill UserSkill
	query := `INSERT INTO tb_Usuario_Habilidad (ID_Persona, ID_Habilidad, Nivel_Dominio, Fecha_Agregada) 
	          VALUES ($1, $2, $3, $4) 
	          RETURNING ID_Usuario_Habilidad, ID_Persona, ID_Habilidad, Nivel_Dominio, Fecha_Agregada`

	err := s.db.QueryRow(ctx, query, idPersona, idHabilidad, nivelDominio, time.Now()).
		Scan(&userSkill.IDUsuarioHabilidad, &userSkill.IDPersona, &userSkill.IDHabilidad, &userSkill.NivelDominio, &userSkill.FechaAgregada)

	if err != nil {
		return nil, err
	}
	return &userSkill, nil
}

// GetUserSkills obtiene todas las habilidades de un usuario
func (s *SkillService) GetUserSkills(ctx context.Context, idPersona int) ([]UserSkill, error) {
	var skills []UserSkill
	query := `SELECT uh.ID_Usuario_Habilidad, uh.ID_Persona, uh.ID_Habilidad, h.Nombre_Habilidad, uh.Nivel_Dominio, uh.Fecha_Agregada 
	          FROM tb_Usuario_Habilidad uh 
	          JOIN tb_Habilidad h ON uh.ID_Habilidad = h.ID_Habilidad 
	          WHERE uh.ID_Persona = $1 
	          ORDER BY h.Nombre_Habilidad`

	rows, err := s.db.Query(ctx, query, idPersona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var skill UserSkill
		if err := rows.Scan(&skill.IDUsuarioHabilidad, &skill.IDPersona, &skill.IDHabilidad, &skill.NombreHabilidad, &skill.NivelDominio, &skill.FechaAgregada); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// RemoveUserSkill elimina una habilidad de un usuario
func (s *SkillService) RemoveUserSkill(ctx context.Context, idUsuarioHabilidad int) error {
	query := `DELETE FROM tb_Usuario_Habilidad WHERE ID_Usuario_Habilidad = $1`
	result, err := s.db.Exec(ctx, query, idUsuarioHabilidad)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateUserSkillLevel actualiza el nivel de dominio de una habilidad
func (s *SkillService) UpdateUserSkillLevel(ctx context.Context, idUsuarioHabilidad int, nivelDominio string) error {
	query := `UPDATE tb_Usuario_Habilidad SET Nivel_Dominio = $1 WHERE ID_Usuario_Habilidad = $2`
	result, err := s.db.Exec(ctx, query, nivelDominio, idUsuarioHabilidad)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}