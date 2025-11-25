package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SuscripcionService struct {
	db *pgxpool.Pool
}

type Suscripcion struct {
	IDPersona       int
	FechaInicial    time.Time
	FechaExpiracion *time.Time
	IDPlan          int
}

const DefaultFreePlanID = 0

func NewSuscripcionService(db *pgxpool.Pool) *SuscripcionService {
	return &SuscripcionService{db}
}

func (s *SuscripcionService) CreateFreeSubscription(ctx context.Context, personaID int) error {
	_, err := s.db.Exec(ctx, `
        INSERT INTO tb_suscripcion (id_persona, id_plan, fecha_inicial, fecha_expiracion)
        VALUES ($1, $2, NOW(), NULL)
        ON CONFLICT (id_persona) DO NOTHING
    `, personaID, DefaultFreePlanID)

	return err
}

func (s *SuscripcionService) Activate(userID, planID int) error {
	now := time.Now().UTC()
	expires := now.AddDate(0, 1, 0) // +30 días

	_, err := s.db.Exec(
		context.Background(),
		`
		INSERT INTO tb_suscripcion (id_persona, fecha_inicial, fecha_expiracion, id_plan)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id_persona)
		DO UPDATE SET 
			id_plan = EXCLUDED.id_plan,
			fecha_inicial = EXCLUDED.fecha_inicial,
			fecha_expiracion = EXCLUDED.fecha_expiracion
		`,
		userID, now, expires, planID,
	)

	return err
}

func (s *SuscripcionService) GetActiveByPersona(ctx context.Context, personaID int) (*Suscripcion, error) {
	var sub Suscripcion
	var fechaExp *time.Time

	err := s.db.QueryRow(ctx, `
        SELECT id_persona, fecha_inicial, fecha_expiracion, id_plan
        FROM tb_suscripcion
        WHERE id_persona = $1
          AND (fecha_expiracion IS NULL OR fecha_expiracion > NOW())
        LIMIT 1
    `, personaID).Scan(
		&sub.IDPersona,
		&sub.FechaInicial,
		&fechaExp,
		&sub.IDPlan,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	sub.FechaExpiracion = fechaExp
	return &sub, nil
}
