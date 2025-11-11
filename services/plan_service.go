package services

import (
	"context"
	"fmt"
	"mentorly-backend/models"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PlanService struct {
	db *pgxpool.Pool
}

func NewPlanService(db *pgxpool.Pool) *PlanService {
	return &PlanService{db: db}
}

// CreatePlan crea un nuevo plan en la base de datos.
func (s *PlanService) CreatePlan(ctx context.Context, plan models.Plan) (*models.Plan, error) {
	query := `INSERT INTO tb_plan (nombre_plan, precio, descripcion, activo) VALUES ($1, $2, $3, $4) RETURNING id_plan`
	err := s.db.QueryRow(ctx, query, plan.Nombre, plan.Precio, plan.Descripcion, plan.Activo).Scan(&plan.ID)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetAllPlans obtiene todos los planes de la base de datos.
func (s *PlanService) GetAllPlans(ctx context.Context) ([]models.Plan, error) {
	var plans []models.Plan
	query := `SELECT id_plan, nombre_plan, precio, descripcion, activo FROM tb_plan ORDER BY id_plan`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Plan
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Activo); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// GetPlanByID obtiene un plan por su ID.
func (s *PlanService) GetPlanByID(ctx context.Context, id int) (*models.Plan, error) {
	var p models.Plan
	query := `SELECT id_plan, nombre_plan, precio, descripcion, activo FROM tb_plan WHERE id_plan = $1`
	err := s.db.QueryRow(ctx, query, id).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Activo)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePlan actualiza un plan existente.
func (s *PlanService) UpdatePlan(ctx context.Context, id int, plan models.Plan) error {
	// Construcción dinámica para actualizar solo los campos necesarios (similar a UpdateProfile)
	var setClauses []string
	var args []interface{}
	argID := 1

	// Para una actualización PUT, donde se envían todos los campos
	setClauses = append(setClauses, fmt.Sprintf("nombre_plan = $%d", argID))
	args = append(args, plan.Nombre)
	argID++

	setClauses = append(setClauses, fmt.Sprintf("precio = $%d", argID))
	args = append(args, plan.Precio)
	argID++

	setClauses = append(setClauses, fmt.Sprintf("descripcion = $%d", argID))
	args = append(args, plan.Descripcion)
	argID++

	setClauses = append(setClauses, fmt.Sprintf("activo = $%d", argID))
	args = append(args, plan.Activo)
	argID++

	if len(setClauses) == 0 {
		return nil // No hay nada que actualizar
	}

	query := fmt.Sprintf("UPDATE tb_plan SET %s WHERE id_plan = $%d", strings.Join(setClauses, ", "), argID)
	args = append(args, id)
	_, err := s.db.Exec(ctx, query, args...)
	return err
}

// DeletePlan elimina un plan de la base de datos.
func (s *PlanService) DeletePlan(ctx context.Context, id int) error {
	query := `DELETE FROM tb_plan WHERE id_plan = $1`
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
