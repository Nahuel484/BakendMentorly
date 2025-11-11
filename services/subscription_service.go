package services

import (
	"context"
	"mentorly-backend/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionService struct {
	db *pgxpool.Pool
}

func NewSubscriptionService(db *pgxpool.Pool) *SubscriptionService {
	return &SubscriptionService{db: db}
}

// CreateSubscription crea una nueva suscripción para un usuario a un plan.
func (s *SubscriptionService) CreateSubscription(ctx context.Context, idPersona, idPlan int) (*models.Subscription, error) {
	var sub models.Subscription
	fechaInicial := time.Now()
	fechaExpiracion := fechaInicial.AddDate(0, 1, 0) // Expira en 1 mes

	query := `INSERT INTO tb_suscripcion (id_persona, id_plan, fecha_inicial, fecha_expiracion) VALUES ($1, $2, $3, $4) RETURNING id_suscripcion, id_persona, id_plan, fecha_inicial, fecha_expiracion`
	err := s.db.QueryRow(ctx, query, idPersona, idPlan, fechaInicial, fechaExpiracion).Scan(&sub.ID, &sub.IDPersona, &sub.IDPlan, &sub.FechaInicial, &sub.FechaExpiracion)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
