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
	EsPremium        bool      `json:"es_premium"` // se usa en explore
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
	EsPremium     bool   `json:"es_premium"` // mentor premium
	Contratado    bool   `json:"contratado"`
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

// Lista solicitudes abiertas para que las vean mentores (explore)
// 👉 Ahora: contratantes Premium primero
func (s *SolicitudService) ListSolicitudesAbiertas(ctx context.Context) ([]Solicitud, error) {
	// Premium = plan con id_plan >= 3 (ajustá si tu premium es otro id)
	const premiumPlanID = 2

	rows, err := s.db.Query(ctx, `
        SELECT 
            sol.id_solicitud,
            sol.id_contratante,
            sol.titulo,
            sol.descripcion,
            sol.fecha_publicacion,
            sol.estado,
            COALESCE(CASE 
                WHEN sus.id_plan >= $1 THEN true
                ELSE false
            END, false) AS es_premium
        FROM tb_solicitud sol
        LEFT JOIN tb_suscripcion sus ON sus.id_persona = sol.id_contratante
        WHERE sol.estado = 'abierta'
        ORDER BY es_premium DESC, sol.fecha_publicacion DESC
    `, premiumPlanID)
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
			&sol.EsPremium,
		); err != nil {
			return nil, err
		}
		result = append(result, sol)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Lista las solicitudes de un contratante junto con las postulaciones recibidas
// 👉 Ahora: postulaciones de mentores Premium primero
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

	const premiumPlanID = 2 // 👈 tu Premium es id_plan 2

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

		// 2) Para cada solicitud, traemos las postulaciones + persona + si el mentor es Premium + si ya fue contratada
		postRows, err := s.db.Query(ctx, `
            SELECT 
                p.id_postulacion, 
                p.id_persona, 
                pe.nombre, 
                pe.apellido, 
                pe.email,
                COALESCE(CASE 
                    WHEN sus.id_plan >= $2 THEN true
                    ELSE false
                END, false) AS es_premium,
                CASE 
                    WHEN c.id_contratacion IS NOT NULL THEN true
                    ELSE false
                END AS contratado
            FROM tb_postulacion p
            JOIN tb_persona pe ON pe.id_persona = p.id_persona
            LEFT JOIN tb_suscripcion sus ON sus.id_persona = p.id_persona
            LEFT JOIN tb_contratacion c ON c.id_postulacion = p.id_postulacion
            WHERE p.id_solicitud = $1
            ORDER BY es_premium DESC, p.id_postulacion DESC
        `, sol.IDSolicitud, premiumPlanID)
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
				&post.EsPremium,
				&post.Contratado, // 👈 NUEVO
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// CountSolicitudesActivas cuenta cuántas solicitudes/anuncios activos tiene una persona
// 🔧 FIX: usar id_contratante y estado='abierta'
func (s *SolicitudService) CountSolicitudesActivas(ctx context.Context, idPersona int) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM tb_solicitud
        WHERE id_contratante = $1
          AND estado = 'abierta'
    `, idPersona).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteSolicitud marca una solicitud como eliminada.
// No deja eliminar si todavía hay postulaciones sin contratar.
func (s *SolicitudService) DeleteSolicitud(
	ctx context.Context,
	idSolicitud int,
	idContratante int,
) error {
	// 1) Verificar si hay postulaciones PENDIENTES
	// (postulaciones de esa solicitud que no tienen contratación asociada)
	var pendientes int
	err := s.db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM tb_postulacion p
        LEFT JOIN tb_contratacion c
               ON c.id_postulacion = p.id_postulacion
        WHERE p.id_solicitud = $1
          AND c.id_contratacion IS NULL
    `, idSolicitud).Scan(&pendientes)
	if err != nil {
		return fmt.Errorf("error verificando postulaciones pendientes: %w", err)
	}

	if pendientes > 0 {
		return fmt.Errorf("no se puede eliminar la solicitud porque aún tiene postulaciones sin contratar")
	}

	// 2) Soft delete: marcar como eliminada, validando que sea del contratante
	res, err := s.db.Exec(ctx, `
        UPDATE tb_solicitud
        SET estado = 'eliminada'
        WHERE id_solicitud   = $1
          AND id_contratante = $2
    `, idSolicitud, idContratante)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar la solicitud: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("no se encontró la solicitud o no sos el dueño")
	}

	return nil
}
