package models

import "time"

// Subscription representa la estructura de una suscripción en la base de datos.
type Subscription struct {
	ID              int       `json:"id_suscripcion"`
	IDPersona       int       `json:"id_persona"`
	IDPlan          int       `json:"id_plan"`
	FechaInicial    time.Time `json:"fecha_inicial"`
	FechaExpiracion time.Time `json:"fecha_expiracion"`
}
