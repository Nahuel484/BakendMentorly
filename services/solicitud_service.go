package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SolicitudService struct {
	db *pgxpool.Pool
}

func NewSolicitudService(db *pgxpool.Pool) *SolicitudService {
	return &SolicitudService{db: db}
}

type Solicitud struct {
	IDSolicitud      int       `json:"id_solicitud"`
	IDContratante    int       `json:"id_contratante"`
	Titulo           string    `json:"titulo"`
	Descripcion      string    `json:"descripcion"`
	FechaPublicacion time.Time `json:"fecha_publicacion"`
	Estado           string    `json:"estado"`
}

type CreateSolicitudInput struct {
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
}

// Postulación con datos básicos del mentor
type PostulacionWithMentor struct {
	IDPostulacion int    `json:"id_postulacion"`
	IDPersona     int    `json:"id_persona"`
	MentorNombre  string `json:"mentor_nombre"`
	MentorEmail   string `json:"mentor_email"`
}

// Solicitud con la lista de postulaciones
type SolicitudWithPostulaciones struct {
	IDSolicitud      int                     `json:"id_solicitud"`
	IDContratante    int                     `json:"id_contratante"`
	Titulo           string                  `json:"titulo"`
	Descripcion      string                  `json:"descripcion"`
	FechaPublicacion time.Time               `json:"fecha_publicacion"`
	Estado           string                  `json:"estado"`
	Postulaciones    []PostulacionWithMentor `json:"postulaciones"`
}

// Crea una solicitud nueva del contratante (startup/emprendedor)
func (s *SolicitudService) CreateSolicitud(
	ctx context.Context,
	idContratante int,
	in CreateSolicitudInput,
) (*Solicitud, error) {

	const estadoInicial = "abierta"

	var sol Solicitud

	err := s.db.QueryRow(ctx, `
        INSERT INTO tb_solicitud
            (id_contratante, titulo, descripcion, fecha_publicacion, estado)
        VALUES
            ($1, $2, $3, NOW(), $4)
        RETURNING
            id_solicitud, id_contratante, titulo, descripcion, fecha_publicacion, estado
    `,
		idContratante,
		in.Titulo,
		in.Descripcion,
		estadoInicial,
	).Scan(
		&sol.IDSolicitud,
		&sol.IDContratante,
		&sol.Titulo,
		&sol.Descripcion,
		&sol.FechaPublicacion,
		&sol.Estado,
	)

	if err != nil {
		return nil, fmt.Errorf("no se pudo crear la solicitud: %w", err)
	}

	return &sol, nil
}

// Lista solicitudes abiertas para que las vean mentores
func (s *SolicitudService) ListSolicitudesAbiertas(ctx context.Context) ([]Solicitud, error) {
	rows, err := s.db.Query(ctx, `
        SELECT id_solicitud, id_contratante, titulo, descripcion, fecha_publicacion, estado
        FROM tb_solicitud
        WHERE estado = 'abierta'
        ORDER BY fecha_publicacion DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("error al listar solicitudes: %w", err)
	}
	defer rows.Close()

	var result []Solicitud
	for rows.Next() {
		var sol Solicitud
		if err := rows.Scan(
			&sol.IDSolicitud,
			&sol.IDContratante,
			&sol.Titulo,
			&sol.Descripcion,
			&sol.FechaPublicacion,
			&sol.Estado,
		); err != nil {
			return nil, err
		}
		result = append(result, sol)
	}

	return result, nil
}

// Lista las solicitudes de un contratante junto con las postulaciones recibidas
func (s *SolicitudService) ListSolicitudesByContratante(
	ctx context.Context,
	idContratante int,
) ([]SolicitudWithPostulaciones, error) {

	// 1) Traemos todas las solicitudes de ese contratante
	rows, err := s.db.Query(ctx, `
        SELECT id_solicitud, id_contratante, titulo, descripcion, fecha_publicacion, estado
        FROM tb_solicitud
        WHERE id_contratante = $1
        ORDER BY fecha_publicacion DESC
    `, idContratante)
	if err != nil {
		return nil, fmt.Errorf("error al listar solicitudes del contratante: %w", err)
	}
	defer rows.Close()

	var result []SolicitudWithPostulaciones

	for rows.Next() {
		var sol SolicitudWithPostulaciones
		if err := rows.Scan(
			&sol.IDSolicitud,
			&sol.IDContratante,
			&sol.Titulo,
			&sol.Descripcion,
			&sol.FechaPublicacion,
			&sol.Estado,
		); err != nil {
			return nil, err
		}

		// 2) Para cada solicitud, traemos las postulaciones + persona
		postRows, err := s.db.Query(ctx, `
            SELECT p.id_postulacion, p.id_persona, pe.nombre, pe.apellido, pe.email
            FROM tb_postulacion p
            JOIN tb_persona pe ON pe.id_persona = p.id_persona
            WHERE p.id_solicitud = $1
        `, sol.IDSolicitud)
		if err != nil {
			return nil, fmt.Errorf("error al listar postulaciones: %w", err)
		}

		var posts []PostulacionWithMentor
		for postRows.Next() {
			var post PostulacionWithMentor
			var nombre, apellido string
			if err := postRows.Scan(
				&post.IDPostulacion,
				&post.IDPersona,
				&nombre,
				&apellido,
				&post.MentorEmail,
			); err != nil {
				postRows.Close()
				return nil, err
			}
			post.MentorNombre = nombre + " " + apellido
			posts = append(posts, post)
		}
		postRows.Close()

		sol.Postulaciones = posts
		result = append(result, sol)
	}

	return result, nil
}
