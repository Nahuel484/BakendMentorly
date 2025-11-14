package services

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
	fromName string
}

// NewEmailService crea una nueva instancia del servicio de email
func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}

	return &EmailService{
		host:     os.Getenv("SMTP_HOST"),
		port:     port,
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("FROM_EMAIL"),
		fromName: os.Getenv("FROM_NAME"),
	}
}

// SendNotificationEmail envía un email de notificación
func (s *EmailService) SendNotificationEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.fromName, s.from))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", s.formatNotificationHTML(subject, body))

	return s.sendEmail(m)
}

// SendProfileUpdateEmail envía un email cuando se actualiza el perfil
func (s *EmailService) SendProfileUpdateEmail(to, nombre string) error {
	subject := "Perfil actualizado exitosamente"
	body := fmt.Sprintf(`
		<h2>¡Hola %s!</h2>
		<p>Tu perfil ha sido actualizado correctamente en Mentorly.</p>
		<p>Si no realizaste este cambio, por favor contacta con soporte.</p>
	`, nombre)

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.fromName, s.from))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	return s.sendEmail(m)
}

// SendWelcomeEmail envía un email de bienvenida
func (s *EmailService) SendWelcomeEmail(to, nombre string) error {
	subject := "¡Bienvenido a Mentorly!"
	body := fmt.Sprintf(`
		<h2>¡Hola %s!</h2>
		<p>Gracias por registrarte en Mentorly.</p>
		<p>Estamos emocionados de tenerte en nuestra plataforma.</p>
	`, nombre)

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.fromName, s.from))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	return s.sendEmail(m)
}

// formatNotificationHTML formatea el HTML para notificaciones
func (s *EmailService) formatNotificationHTML(title, message string) string {
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background: #4F46E5; color: white; padding: 20px; border-radius: 5px 5px 0 0; }
				.content { background: #f9f9f9; padding: 20px; border-radius: 0 0 5px 5px; }
				.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>%s</h1>
				</div>
				<div class="content">
					<p>%s</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Mentorly. Todos los derechos reservados.</p>
				</div>
			</div>
		</body>
		</html>
	`, title, message)
}

// sendEmail envía el email usando gomail
func (s *EmailService) sendEmail(m *gomail.Message) error {
	d := gomail.NewDialer(s.host, s.port, s.user, s.password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("error al enviar email: %w", err)
	}

	return nil
}
