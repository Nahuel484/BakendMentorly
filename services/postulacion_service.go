package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostulacionService struct {
	db                  *pgxpool.Pool
	notificationService *NotificationService
}

func NewPostulacionService(db *pgxpool.Pool, ns *NotificationService) *PostulacionService {
	return &PostulacionService{
		db:                  db,
		notificationService: ns,
	}
}

type Postulacion struct {
	IDPostulacion int `json:"id_postulacion"`
	IDPersona     int `json:"id_persona"`
	IDSolicitud   int `json:"id_solicitud"`
}

// Crea una postulación y notifica por web + email al contratante
func (s *PostulacionService) CreatePostulacion(
	ctx context.Context,
	idPersona int,
	idSolicitud int,
) (*Postulacion, int, error) {

	// Obtener contratante y título de la solicitud
	var (
		idContratante   int
		tituloSolicitud string
	)

	err := s.db.QueryRow(ctx, `
        SELECT id_contratante, titulo
        FROM tb_solicitud
        WHERE id_solicitud = $1
    `, idSolicitud).Scan(&idContratante, &tituloSolicitud)
	if err != nil {
		return nil, 0, fmt.Errorf("no se encontró la solicitud: %w", err)
	}

	var post Postulacion

	err = s.db.QueryRow(ctx, `
        INSERT INTO tb_postulacion
            (id_persona, id_solicitud)
        VALUES
            ($1, $2)
        RETURNING id_postulacion, id_persona, id_solicitud
    `,
		idPersona,
		idSolicitud,
	).Scan(
		&post.IDPostulacion,
		&post.IDPersona,
		&post.IDSolicitud,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("no se pudo crear la postulación: %w", err)
	}

	// Notificación al contratante (web + email)
	if s.notificationService != nil {
		_, err = s.notificationService.CreateNotification(ctx, NotificationRequest{
			IDPersona:   idContratante,
			Titulo:      "Nueva postulación a tu solicitud",
			Mensaje:     fmt.Sprintf("Recibiste una nueva postulación a la solicitud '%s'.", tituloSolicitud),
			Tipo:        "info",
			EnviarEmail: true,
		})
		if err != nil {
			fmt.Printf("error creando notificación de postulación: %v\n", err)
		}
	}

	return &post, idContratante, nil
}

// Rechazar / no contratar una postulación
func (s *PostulacionService) RejectPostulacion(
	ctx context.Context,
	idPostulacion int,
	idContratante int,
) error {
	// Obtener datos de la postulación + mentor + solicitud
	var (
		idMentor        int
		idSolicitud     int
		mentorNombre    string
		mentorEmail     string
		tituloSolicitud string
		ownerSolicitud  int
	)

	err := s.db.QueryRow(ctx, `
        SELECT 
            p.id_persona, 
            p.id_solicitud,
            pe.nombre,
            pe.apellido,
            pe.email
        FROM tb_postulacion p
        JOIN tb_persona pe ON pe.id_persona = p.id_persona
        WHERE p.id_postulacion = $1
    `, idPostulacion).Scan(
		&idMentor,
		&idSolicitud,
		&mentorNombre,
		&mentorNombre, // apellido, pero lo concatenamos debajo igual
		&mentorEmail,
	)
	if err != nil {
		return fmt.Errorf("no se encontró la postulación: %w", err)
	}

	// Verificamos que la solicitud realmente sea del contratante logueado
	err = s.db.QueryRow(ctx, `
        SELECT id_contratante, titulo
        FROM tb_solicitud
        WHERE id_solicitud = $1
    `, idSolicitud).Scan(&ownerSolicitud, &tituloSolicitud)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la solicitud: %w", err)
	}

	if ownerSolicitud != idContratante {
		return fmt.Errorf("no autorizado para rechazar esta postulación")
	}

	// Eliminamos la postulación
	_, err = s.db.Exec(ctx, `
        DELETE FROM tb_postulacion
        WHERE id_postulacion = $1
    `, idPostulacion)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar la postulación: %w", err)
	}

	// Notificación al mentor (web + email)
	if s.notificationService != nil {
		_, err = s.notificationService.CreateNotification(ctx, NotificationRequest{
			IDPersona:   idMentor,
			Titulo:      "Tu postulación fue rechazada",
			Mensaje:     fmt.Sprintf("Tu postulación a la solicitud '%s' fue rechazada.", tituloSolicitud),
			Tipo:        "warning",
			EnviarEmail: true,
		})
		if err != nil {
			fmt.Printf("error creando notificación de rechazo: %v\n", err)
		}
	}

	return nil
}
