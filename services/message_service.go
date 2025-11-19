package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageService struct {
	db *pgxpool.Pool
}

func NewMessageService(db *pgxpool.Pool) *MessageService {
	return &MessageService{db: db}
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

	// 3) notificación al receptor
	preview := in.Contenido
	if len(preview) > 140 {
		preview = preview[:140]
	}

	_, err = s.db.Exec(ctx, `
        INSERT INTO tb_notificacion
            (id_persona_destinatario, id_tipo_notificacion, mensaje, fecha_creacion, estado)
        VALUES
            ($1, $2, $3, NOW(), $4)
    `,
		idReceptor,
		3, // 3 = "nuevo mensaje en conversación"
		preview,
		"pendiente",
	)
	if err != nil {
		fmt.Printf("error creando notificación de mensaje: %v\n", err)
	}

	return &msg, nil
}
