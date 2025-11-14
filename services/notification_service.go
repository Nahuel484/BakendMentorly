package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationService struct {
	db           *pgxpool.Pool
	emailService *EmailService
}

type Notification struct {
	IDNotificacion  int       `json:"id_notificacion"`
	IDPersona       int       `json:"id_persona"`
	Titulo          string    `json:"titulo"`
	Mensaje         string    `json:"mensaje"`
	Tipo            string    `json:"tipo"` // "info", "success", "warning", "error"
	Leida           bool      `json:"leida"`
	EnviadoEmail    bool      `json:"enviado_email"`
	FechaCreacion   time.Time `json:"fecha_creacion"`
	FechaLectura    *time.Time `json:"fecha_lectura,omitempty"`
}

type NotificationRequest struct {
	IDPersona    int    `json:"id_persona"`
	Titulo       string `json:"titulo"`
	Mensaje      string `json:"mensaje"`
	Tipo         string `json:"tipo"`
	EnviarEmail  bool   `json:"enviar_email"`
}

func NewNotificationService(db *pgxpool.Pool) *NotificationService {
	return &NotificationService{
		db:           db,
		emailService: NewEmailService(),
	}
}

// CreateNotification crea una nueva notificación
func (s *NotificationService) CreateNotification(ctx context.Context, req NotificationRequest) (*Notification, error) {
	var notif Notification
	
	query := `INSERT INTO tb_Notificacion (ID_Persona, Titulo, Mensaje, Tipo, Leida, Enviado_Email, Fecha_Creacion) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7) 
	          RETURNING ID_Notificacion, ID_Persona, Titulo, Mensaje, Tipo, Leida, Enviado_Email, Fecha_Creacion`

	err := s.db.QueryRow(ctx, query,
		req.IDPersona,
		req.Titulo,
		req.Mensaje,
		req.Tipo,
		false, // Leida = false por defecto
		false, // Enviado_Email = false inicialmente
		time.Now(),
	).Scan(
		&notif.IDNotificacion,
		&notif.IDPersona,
		&notif.Titulo,
		&notif.Mensaje,
		&notif.Tipo,
		&notif.Leida,
		&notif.EnviadoEmail,
		&notif.FechaCreacion,
	)

	if err != nil {
		return nil, err
	}

	// Si se solicita enviar email, hacerlo de forma asíncrona
	if req.EnviarEmail {
		go s.sendEmailNotification(ctx, &notif)
	}

	return &notif, nil
}

// sendEmailNotification envía la notificación por correo electrónico
func (s *NotificationService) sendEmailNotification(ctx context.Context, notif *Notification) {
	// Obtener el email del usuario
	var email string
	err := s.db.QueryRow(ctx,
		"SELECT email FROM tb_persona WHERE id_persona = $1",
		notif.IDPersona,
	).Scan(&email)

	if err != nil {
		fmt.Printf("Error al obtener email del usuario: %v\n", err)
		return
	}

	// Enviar el email
	err = s.emailService.SendNotificationEmail(email, notif.Titulo, notif.Mensaje)
	if err != nil {
		fmt.Printf("Error al enviar email: %v\n", err)
		return
	}

	// Marcar como enviado en la base de datos
	_, err = s.db.Exec(ctx,
		"UPDATE tb_Notificacion SET Enviado_Email = true WHERE ID_Notificacion = $1",
		notif.IDNotificacion,
	)
	if err != nil {
		fmt.Printf("Error al actualizar estado de email: %v\n", err)
	}
}

// GetUserNotifications obtiene todas las notificaciones de un usuario
func (s *NotificationService) GetUserNotifications(ctx context.Context, idPersona int, soloNoLeidas bool) ([]Notification, error) {
	var notifications []Notification
	
	query := `SELECT ID_Notificacion, ID_Persona, Titulo, Mensaje, Tipo, Leida, Enviado_Email, Fecha_Creacion, Fecha_Lectura 
	          FROM tb_Notificacion 
	          WHERE ID_Persona = $1`
	
	if soloNoLeidas {
		query += " AND Leida = false"
	}
	
	query += " ORDER BY Fecha_Creacion DESC"

	rows, err := s.db.Query(ctx, query, idPersona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var notif Notification
		err := rows.Scan(
			&notif.IDNotificacion,
			&notif.IDPersona,
			&notif.Titulo,
			&notif.Mensaje,
			&notif.Tipo,
			&notif.Leida,
			&notif.EnviadoEmail,
			&notif.FechaCreacion,
			&notif.FechaLectura,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, nil
}

// MarkAsRead marca una notificación como leída
func (s *NotificationService) MarkAsRead(ctx context.Context, idNotificacion int, idPersona int) error {
	query := `UPDATE tb_Notificacion 
	          SET Leida = true, Fecha_Lectura = $1 
	          WHERE ID_Notificacion = $2 AND ID_Persona = $3 AND Leida = false`
	
	result, err := s.db.Exec(ctx, query, time.Now(), idNotificacion, idPersona)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// MarkAllAsRead marca todas las notificaciones de un usuario como leídas
func (s *NotificationService) MarkAllAsRead(ctx context.Context, idPersona int) error {
	query := `UPDATE tb_Notificacion 
	          SET Leida = true, Fecha_Lectura = $1 
	          WHERE ID_Persona = $2 AND Leida = false`
	
	_, err := s.db.Exec(ctx, query, time.Now(), idPersona)
	return err
}

// GetUnreadCount obtiene el número de notificaciones no leídas
func (s *NotificationService) GetUnreadCount(ctx context.Context, idPersona int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tb_Notificacion WHERE ID_Persona = $1 AND Leida = false`
	
	err := s.db.QueryRow(ctx, query, idPersona).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// DeleteNotification elimina una notificación
func (s *NotificationService) DeleteNotification(ctx context.Context, idNotificacion int, idPersona int) error {
	query := `DELETE FROM tb_Notificacion WHERE ID_Notificacion = $1 AND ID_Persona = $2`
	
	result, err := s.db.Exec(ctx, query, idNotificacion, idPersona)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// GetNotificationByID obtiene una notificación específica
func (s *NotificationService) GetNotificationByID(ctx context.Context, idNotificacion int, idPersona int) (*Notification, error) {
	var notif Notification
	
	query := `SELECT ID_Notificacion, ID_Persona, Titulo, Mensaje, Tipo, Leida, Enviado_Email, Fecha_Creacion, Fecha_Lectura 
	          FROM tb_Notificacion 
	          WHERE ID_Notificacion = $1 AND ID_Persona = $2`

	err := s.db.QueryRow(ctx, query, idNotificacion, idPersona).Scan(
		&notif.IDNotificacion,
		&notif.IDPersona,
		&notif.Titulo,
		&notif.Mensaje,
		&notif.Tipo,
		&notif.Leida,
		&notif.EnviadoEmail,
		&notif.FechaCreacion,
		&notif.FechaLectura,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &notif, nil
}