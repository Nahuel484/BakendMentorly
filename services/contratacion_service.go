package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ContratacionService struct {
	db *pgxpool.Pool
}

func NewContratacionService(db *pgxpool.Pool) *ContratacionService {
	return &ContratacionService{db: db}
}

type Contratacion struct {
	IDContratacion int `json:"id_contratacion"`
	IDPostulacion  int `json:"id_postulacion"`
	IDContratado   int `json:"id_contratado"`
}

type Conversacion struct {
	IDConversacion int       `json:"id_conversacion"`
	IDContratacion int       `json:"id_contratacion"`
	Asunto         string    `json:"asunto"`
	FechaCreacion  time.Time `json:"fecha_creacion"`
}

type ContratacionConConversacion struct {
	Contratacion Contratacion `json:"contratacion"`
	Conversacion Conversacion `json:"conversacion"`
}

// Crea una contratación a partir de una postulación y abre conversación
func (s *ContratacionService) CreateContratacion(
	ctx context.Context,
	idPostulacion int,
) (*ContratacionConConversacion, error) {

	// 1) Obtener datos de la postulación + solicitud
	var (
		idPersonaPostulante int
		idSolicitud         int
		idContratante       int
		tituloSolicitud     string
	)

	err := s.db.QueryRow(ctx, `
        SELECT p.id_persona, p.id_solicitud, s.id_contratante, s.titulo
        FROM tb_postulacion p
        JOIN tb_solicitud s ON s.id_solicitud = p.id_solicitud
        WHERE p.id_postulacion = $1
    `, idPostulacion).Scan(
		&idPersonaPostulante,
		&idSolicitud,
		&idContratante,
		&tituloSolicitud,
	)
	if err != nil {
		return nil, fmt.Errorf("no se encontró la postulación: %w", err)
	}

	// 2) Crear contratación
	var contr Contratacion
	err = s.db.QueryRow(ctx, `
        INSERT INTO tb_contratacion
            (id_postulacion, id_contratado)
        VALUES
            ($1, $2)
        RETURNING id_contratacion, id_postulacion, id_contratado
    `,
		idPostulacion,
		idPersonaPostulante,
	).Scan(
		&contr.IDContratacion,
		&contr.IDPostulacion,
		&contr.IDContratado,
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear la contratación: %w", err)
	}

	// 3) Crear conversación ligada a la contratación
	var conv Conversacion
	asunto := fmt.Sprintf("Conversación sobre: %s", tituloSolicitud)

	err = s.db.QueryRow(ctx, `
        INSERT INTO tb_conversacion
            (id_contratacion, asunto, fecha_creacion)
        VALUES
            ($1, $2, NOW())
        RETURNING id_conversacion, id_contratacion, asunto, fecha_creacion
    `,
		contr.IDContratacion,
		asunto,
	).Scan(
		&conv.IDConversacion,
		&conv.IDContratacion,
		&conv.Asunto,
		&conv.FechaCreacion,
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear la conversación: %w", err)
	}

	// 4) Notificación al mentor (contratado)
	_, err = s.db.Exec(ctx, `
        INSERT INTO tb_notificacion
            (id_persona_destinatario, id_tipo_notificacion, mensaje, fecha_creacion, estado)
        VALUES
            ($1, $2, $3, NOW(), $4)
    `,
		idPersonaPostulante,
		2, // 2 = "fuiste contratado"
		fmt.Sprintf("Fuiste contratado para la solicitud '%s'", tituloSolicitud),
		"pendiente",
	)
	if err != nil {
		fmt.Printf("error creando notificación de contratación: %v\n", err)
	}

	return &ContratacionConConversacion{
		Contratacion: contr,
		Conversacion: conv,
	}, nil
}
