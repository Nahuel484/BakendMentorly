package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageService struct {
	db                  *pgxpool.Pool
	notificationService *NotificationService
}

func NewMessageService(db *pgxpool.Pool, ns *NotificationService) *MessageService {
	return &MessageService{
		db:                  db,
		notificationService: ns,
	}
}

// 👇 Para listar las conversaciones en el front
type ChatConversacion struct {
	IDConversacion int        `json:"id_conversacion"`
	IDContratacion int        `json:"id_contratacion"`
	IDMentor       int        `json:"id_mentor"`
	IDContratante  int        `json:"id_contratante"`
	Cerrada        bool       `json:"cerrada"`
	FechaInicio    time.Time  `json:"fecha_inicio"`
	FechaCierre    *time.Time `json:"fecha_cierre,omitempty"`
}

type SendMessageInput struct {
	IDConversacion int    `json:"id_conversacion"`
	Contenido      string `json:"contenido"`
}

type Message struct {
	IDMensaje      int       `json:"id_mensaje"`
	IDConversacion int       `json:"id_conversacion"`
	IDRemitente    int       `json:"id_remitente"`
	Contenido      string    `json:"contenido"`
	FechaEnvio     time.Time `json:"fecha_envio"`
}

// =============================
//    LÓGICA EXISTENTE
// =============================

// Dado id_conversacion y senderID, buscamos quién es el otro participante
func (s *MessageService) getReceptorForConversacion(
	ctx context.Context,
	idConversacion int,
	senderID int,
) (int, error) {
	var (
		idContratacion int
		idPostulacion  int
		idContratado   int
		idContratante  int
	)

	err := s.db.QueryRow(ctx, `
        SELECT
            c.id_contratacion,
            co.id_postulacion,
            co.id_contratado,
            s2.id_contratante
        FROM tb_conversacion c
        JOIN tb_contratacion co ON co.id_contratacion = c.id_contratacion
        JOIN tb_postulacion p ON p.id_postulacion = co.id_postulacion
        JOIN tb_solicitud s2 ON s2.id_solicitud = p.id_solicitud
        WHERE c.id_conversacion = $1
    `, idConversacion).Scan(
		&idContratacion,
		&idPostulacion,
		&idContratado,
		&idContratante,
	)
	if err != nil {
		return 0, fmt.Errorf("no se pudo resolver la conversación: %w", err)
	}

	if senderID == idContratado {
		return idContratante, nil
	}
	if senderID == idContratante {
		return idContratado, nil
	}

	// Si no coincide con ninguno, algo está mal
	return 0, fmt.Errorf("el remitente no participa en esta conversación")
}

func (s *MessageService) SendMessage(
	ctx context.Context,
	senderID int,
	in SendMessageInput,
) (*Message, error) {

	// 0) Verificar que la conversación no esté cerrada
	var cerrada bool
	err := s.db.QueryRow(ctx, `
			SELECT COALESCE(cerrada, FALSE) AS cerrada
			FROM tb_conversacion
			WHERE id_conversacion = $1
	`, in.IDConversacion).Scan(&cerrada)
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener estado de la conversación: %w", err)
	}
	if cerrada {
		return nil, fmt.Errorf("la conversación está cerrada; no se pueden enviar mensajes")
	}

	// 1) obtener receptor
	idReceptor, err := s.getReceptorForConversacion(ctx, in.IDConversacion, senderID)
	if err != nil {
		return nil, err
	}

	// 2) insertar mensaje
	var msg Message
	err = s.db.QueryRow(ctx, `
        INSERT INTO tb_mensaje
            (id_conversacion, id_remitente, contenido, fecha_envio)
        VALUES
            ($1, $2, $3, NOW())
        RETURNING id_mensaje, id_conversacion, id_remitente, contenido, fecha_envio
    `,
		in.IDConversacion,
		senderID,
		in.Contenido,
	).Scan(
		&msg.IDMensaje,
		&msg.IDConversacion,
		&msg.IDRemitente,
		&msg.Contenido,
		&msg.FechaEnvio,
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el mensaje: %w", err)
	}

	// 3) notificación al receptor (web + email) usando NotificationService
	if s.notificationService != nil {
		preview := in.Contenido
		if len(preview) > 140 {
			preview = preview[:140]
		}

		_, err = s.notificationService.CreateNotification(ctx, NotificationRequest{
			IDPersona:   idReceptor,
			Titulo:      "Nuevo mensaje en una conversación",
			Mensaje:     preview,
			Tipo:        "mensaje",
			EnviarEmail: true,
		})
		if err != nil {
			fmt.Printf("error creando notificación de mensaje: %v\n", err)
		}
	}

	return &msg, nil
}

// =============================
//    NUEVAS FUNCIONALIDADES
// =============================

// ListConversacionesByPersona devuelve las conversaciones donde participa la persona
func (s *MessageService) ListConversacionesByPersona(
	ctx context.Context,
	idPersona int,
) ([]ChatConversacion, error) {

	rows, err := s.db.Query(ctx, `
        SELECT 
            c.id_conversacion,
            c.id_contratacion,
            co.id_contratado                     AS id_mentor,
            s2.id_contratante                    AS id_contratante,
            COALESCE(c.cerrada, FALSE) AS cerrada,
            c.fecha_creacion                     AS fecha_inicio,
            c.fecha_cierre                       AS fecha_cierre
        FROM tb_conversacion c
        JOIN tb_contratacion co ON co.id_contratacion = c.id_contratacion
        JOIN tb_postulacion p ON p.id_postulacion = co.id_postulacion
        JOIN tb_solicitud s2 ON s2.id_solicitud = p.id_solicitud
        WHERE co.id_contratado = $1
           OR s2.id_contratante = $1
        ORDER BY c.fecha_creacion DESC
    `, idPersona)
	if err != nil {
		return nil, fmt.Errorf("error listando conversaciones: %w", err)
	}
	defer rows.Close()

	var result []ChatConversacion

	for rows.Next() {
		var c ChatConversacion
		var fechaCierre *time.Time

		if err := rows.Scan(
			&c.IDConversacion,
			&c.IDContratacion,
			&c.IDMentor,
			&c.IDContratante,
			&c.Cerrada,
			&c.FechaInicio,
			&fechaCierre,
		); err != nil {
			return nil, err
		}

		c.FechaCierre = fechaCierre
		result = append(result, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ListMessagesByConversacion devuelve los mensajes de una conversación
// validando que la persona sea participante.
func (s *MessageService) ListMessagesByConversacion(
	ctx context.Context,
	idConversacion int,
	idPersona int,
) ([]Message, error) {

	// 1) Verificar que el usuario participa en la conversación
	var count int
	err := s.db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM tb_conversacion c
        JOIN tb_contratacion co ON co.id_contratacion = c.id_contratacion
        JOIN tb_postulacion p ON p.id_postulacion = co.id_postulacion
        JOIN tb_solicitud s2 ON s2.id_solicitud = p.id_solicitud
        WHERE c.id_conversacion = $1
          AND (co.id_contratado = $2 OR s2.id_contratante = $2)
    `, idConversacion, idPersona).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("error validando conversación: %w", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("no estás autorizado a ver esta conversación")
	}

	// 2) Traer mensajes
	rows, err := s.db.Query(ctx, `
        SELECT
            id_mensaje,
            id_conversacion,
            id_remitente,
            contenido,
            fecha_envio
        FROM tb_mensaje
        WHERE id_conversacion = $1
        ORDER BY fecha_envio ASC
    `, idConversacion)
	if err != nil {
		return nil, fmt.Errorf("error listando mensajes: %w", err)
	}
	defer rows.Close()

	var result []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.IDMensaje,
			&m.IDConversacion,
			&m.IDRemitente,
			&m.Contenido,
			&m.FechaEnvio,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// CerrarConversacion marca una conversación como cerrada
func (s *MessageService) CerrarConversacion(
	ctx context.Context,
	idConversacion int,
	idPersona int,
) error {

	res, err := s.db.Exec(ctx, `
        UPDATE tb_conversacion c
        SET cerrada = TRUE,
            fecha_cierre = NOW()
        FROM tb_contratacion co
        JOIN tb_postulacion p ON p.id_postulacion = co.id_postulacion
        JOIN tb_solicitud s2 ON s2.id_solicitud = p.id_solicitud
        WHERE c.id_conversacion = $1
          AND c.id_contratacion = co.id_contratacion
          AND (co.id_contratado = $2 OR s2.id_contratante = $2)
    `, idConversacion, idPersona)
	if err != nil {
		return fmt.Errorf("error cerrando conversación: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("no autorizado o conversación inexistente")
	}

	return nil
}
