package services

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"forgecrud-backend/shared/config"
)

// EmailRequest represents a simple email request
type EmailRequest struct {
	To           []string               `json:"to" binding:"required"`
	CC           []string               `json:"cc,omitempty"`
	BCC          []string               `json:"bcc,omitempty"`
	Subject      string                 `json:"subject" binding:"required"`
	Body         string                 `json:"body"`
	IsHTML       bool                   `json:"is_html"`
	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateVars map[string]interface{} `json:"template_vars,omitempty"`
}

// EmailResponse represents the response after sending an email
type EmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SentAt  string `json:"sent_at"`
}

// EmailService handles sending emails
type EmailService struct {
	config          *config.Config
	templateService *TemplateService
}

// NewEmailService creates a new email service
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		config:          cfg,
		templateService: NewTemplateService(cfg),
	}
}

// SendEmail sends an email immediately using SMTP
func (es *EmailService) SendEmail(request EmailRequest) (*EmailResponse, error) {
	startTime := time.Now()

	// Validate email request
	if len(request.To) == 0 {
		return nil, fmt.Errorf("recipient list cannot be empty")
	}

	if request.Subject == "" {
		return nil, fmt.Errorf("subject cannot be empty")
	}

	// If template is specified, render it
	if request.TemplateID != "" && request.TemplateVars != nil {
		renderedBody, err := es.templateService.RenderTemplate(request.TemplateID, request.TemplateVars)
		if err != nil {
			log.Printf("Failed to render template: %v", err)
			return nil, fmt.Errorf("failed to render template: %v", err)
		}
		request.Body = renderedBody
		request.IsHTML = true // Templates are HTML by default
	}

	if request.Body == "" {
		return nil, fmt.Errorf("body cannot be empty")
	}

	// Send email immediately
	err := es.sendSMTPEmail(request)
	if err != nil {
		log.Printf("Failed to send email to %v: %v", request.To, err)
		return &EmailResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to send email: %v", err),
			SentAt:  startTime.Format(time.RFC3339),
		}, err
	}

	log.Printf("Email sent successfully to %v", request.To)
	return &EmailResponse{
		Success: true,
		Message: "Email sent successfully",
		SentAt:  startTime.Format(time.RFC3339),
	}, nil
}

// sendSMTPEmail sends email via SMTP
func (es *EmailService) sendSMTPEmail(request EmailRequest) error {
	// Build message
	message := es.buildEmailMessage(request)

	// SMTP configuration from config
	host := es.config.SMTPHost
	port := es.config.SMTPPort
	username := es.config.SMTPUsername
	password := es.config.SMTPPassword
	from := es.config.EmailFrom

	// Validate SMTP config
	if host == "" || username == "" || password == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	// SMTP auth
	auth := smtp.PlainAuth("", username, password, host)

	// Connect to server
	addr := fmt.Sprintf("%s:%s", host, port)

	// Recipients
	recipients := append(request.To, request.CC...)
	recipients = append(recipients, request.BCC...)

	// Port 465 uses implicit TLS (SSL), other ports may use explicit TLS (STARTTLS)
	if port == "465" || es.config.SMTPUseTLS {
		return es.sendWithTLS(addr, auth, from, recipients, []byte(message))
	}

	// Regular SMTP without TLS
	return smtp.SendMail(addr, auth, from, recipients, []byte(message))
}

// sendWithTLS sends email with TLS
func (es *EmailService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Connect to server
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         strings.Split(addr, ":")[0],
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Quit()

	// Auth
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return err
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = w.Write(msg)
	return err
}

// buildEmailMessage builds email message
func (es *EmailService) buildEmailMessage(request EmailRequest) string {
	from := es.config.EmailFrom
	fromName := es.config.EmailFromName

	var msg strings.Builder

	// Headers
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(request.To, ", ")))

	if len(request.CC) > 0 {
		msg.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(request.CC, ", ")))
	}

	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", request.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if request.IsHTML {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	msg.WriteString("\r\n")
	msg.WriteString(request.Body)

	return msg.String()
}

// Helper methods for common email templates

// SendWelcomeEmail sends a welcome email with verification code
func (es *EmailService) SendWelcomeEmail(to, name, verificationCode string) (*EmailResponse, error) {
	request := EmailRequest{
		To:         []string{to},
		Subject:    "Welcome to ForgeCRUD - Please Verify Your Email",
		TemplateID: "welcome_verification",
		TemplateVars: map[string]interface{}{
			"Name":             name,
			"VerificationCode": verificationCode,
		},
	}

	return es.SendEmail(request)
}

// SendPasswordResetEmail sends password reset email
func (es *EmailService) SendPasswordResetEmail(to, name, resetCode string) (*EmailResponse, error) {
	request := EmailRequest{
		To:         []string{to},
		Subject:    "Password Reset Request - ForgeCRUD",
		TemplateID: "password_reset",
		TemplateVars: map[string]interface{}{
			"Name":      name,
			"ResetCode": resetCode,
		},
	}

	return es.SendEmail(request)
}
