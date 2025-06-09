package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"forgecrud-backend/shared/config"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService() *EmailService {
	return &EmailService{
		config: config.GetConfig(),
	}
}

type EmailData struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

func (e *EmailService) SendEmail(emailData EmailData) error {
	addr := fmt.Sprintf("%s:%s", e.config.SMTPHost, e.config.SMTPPort)

	var client *smtp.Client
	var err error

	if e.config.SMTPPort == "465" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         e.config.SMTPHost,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			client, err = smtp.Dial(addr)
			if err != nil {
				return err
			}
		} else {
			client, err = smtp.NewClient(conn, e.config.SMTPHost)
			if err != nil {
				return err
			}
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return err
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: e.config.SMTPHost}
			if err = client.StartTLS(config); err != nil {
				// Non-critical error, continue without TLS
			}
		}
	}
	defer client.Close()

	auth := smtp.PlainAuth("", e.config.SMTPUsername, e.config.SMTPPassword, e.config.SMTPHost)
	if err = client.Auth(auth); err != nil {
		return err
	}

	if err = client.Mail(e.config.EmailFrom); err != nil {
		return err
	}

	if err = client.Rcpt(emailData.To); err != nil {
		return err
	}

	var contentType string
	if emailData.IsHTML {
		contentType = "text/html; charset=UTF-8"
	} else {
		contentType = "text/plain; charset=UTF-8"
	}

	message := fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: %s\r\n"+
		"\r\n"+
		"%s\r\n",
		emailData.To,
		e.config.EmailFromName,
		e.config.EmailFrom,
		emailData.Subject,
		contentType,
		emailData.Body)

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	client.Quit()

	return nil
}

func (e *EmailService) SendVerificationEmail(toEmail, userName, verificationToken string) error {
	baseURL := e.config.APIGatewayURL
	verificationURL := fmt.Sprintf("%s/api/auth/verify-email/%s", baseURL, verificationToken)

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email Verification - ForgeCRUD</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 10px;">
        <h1 style="color: #343a40; text-align: center;">Welcome to ForgeCRUD!</h1>
        
        <p style="color: #6c757d; font-size: 16px;">Hello {{.UserName}},</p>
        
        <p style="color: #6c757d; font-size: 16px;">
            Thank you for registering with ForgeCRUD. To complete your registration and activate your account, 
            please click the verification link below:
        </p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationURL}}" 
               style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; 
                      border-radius: 5px; font-weight: bold; display: inline-block;">
                Verify My Email
            </a>
        </div>
        
        <p style="color: #6c757d; font-size: 14px;">
            If the button doesn't work, copy and paste this link into your browser:
        </p>
        
        <p style="color: #007bff; font-size: 14px; word-break: break-all;">
            {{.VerificationURL}}
        </p>
        
        <hr style="border: none; border-top: 1px solid #dee2e6; margin: 30px 0;">
        
        <p style="color: #6c757d; font-size: 12px; text-align: center;">
            This verification link will expire in 24 hours. If you didn't create an account with ForgeCRUD, 
            please ignore this email.
        </p>
        
        <p style="color: #6c757d; font-size: 12px; text-align: center;">
            Best regards,<br>
            The ForgeCRUD Team
        </p>
    </div>
</body>
</html>`

	textTemplate := `
Welcome to ForgeCRUD!

Hello {{.UserName}},

Thank you for registering with ForgeCRUD. To complete your registration and activate your account, 
please visit the following link:

{{.VerificationURL}}

This verification link will expire in 24 hours. If you didn't create an account with ForgeCRUD, 
please ignore this email.

Best regards,
The ForgeCRUD Team
`

	templateData := struct {
		UserName        string
		VerificationURL string
	}{
		UserName:        userName,
		VerificationURL: verificationURL,
	}

	tmpl, err := template.New("verification").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var htmlBody bytes.Buffer
	if err := tmpl.Execute(&htmlBody, templateData); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	emailData := EmailData{
		To:      toEmail,
		Subject: "Verify Your Email Address - ForgeCRUD",
		Body:    htmlBody.String(),
		IsHTML:  true,
	}

	if err := e.SendEmail(emailData); err != nil {
		textTmpl, _ := template.New("verification_text").Parse(textTemplate)
		var textBody bytes.Buffer
		textTmpl.Execute(&textBody, templateData)

		emailData.Body = textBody.String()
		emailData.IsHTML = false

		return e.SendEmail(emailData)
	}

	return nil
}

func (e *EmailService) SendPasswordResetEmail(toEmail, userName, resetToken string) error {
	baseURL := e.config.APIGatewayURL
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, resetToken)

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset - ForgeCRUD</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 10px;">
        <h1 style="color: #343a40; text-align: center;">Password Reset Request</h1>
        
        <p style="color: #6c757d; font-size: 16px;">Hello {{.UserName}},</p>
        
        <p style="color: #6c757d; font-size: 16px;">
            We received a request to reset your ForgeCRUD account password. To proceed with the password reset, 
            please click the button below:
        </p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetURL}}" 
               style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; 
                      border-radius: 5px; font-weight: bold; display: inline-block;">
                Reset My Password
            </a>
        </div>
        
        <p style="color: #6c757d; font-size: 14px;">
            If the button doesn't work, copy and paste this link into your browser:
        </p>
        
        <p style="color: #007bff; font-size: 14px; word-break: break-all;">
            {{.ResetURL}}
        </p>
        
        <hr style="border: none; border-top: 1px solid #dee2e6; margin: 30px 0;">
        
        <p style="color: #dc3545; font-size: 14px;">
            <strong>Important:</strong> This password reset link will expire in 1 hour. If you didn't request a 
            password reset, please ignore this email or contact support if you have concerns.
        </p>
        
        <p style="color: #6c757d; font-size: 12px; text-align: center;">
            Best regards,<br>
            The ForgeCRUD Team
        </p>
    </div>
</body>
</html>`

	textTemplate := `
Password Reset Request - ForgeCRUD

Hello {{.UserName}},

You have requested to reset your password. Click the link below to reset your password:

{{.ResetURL}}

This link will expire in 1 hour. If you didn't request a password reset, please ignore this email.

Best regards,
The ForgeCRUD Team
`

	templateData := struct {
		UserName string
		ResetURL string
	}{
		UserName: userName,
		ResetURL: resetURL,
	}

	tmpl, err := template.New("password_reset").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse password reset template: %w", err)
	}

	var htmlBody bytes.Buffer
	if err := tmpl.Execute(&htmlBody, templateData); err != nil {
		return fmt.Errorf("failed to execute password reset template: %w", err)
	}

	emailData := EmailData{
		To:      toEmail,
		Subject: "Password Reset Request - ForgeCRUD",
		Body:    htmlBody.String(),
		IsHTML:  true,
	}

	if err := e.SendEmail(emailData); err != nil {
		textTmpl, _ := template.New("password_reset_text").Parse(textTemplate)
		var textBody bytes.Buffer
		textTmpl.Execute(&textBody, templateData)

		emailData.Body = textBody.String()
		emailData.IsHTML = false

		return e.SendEmail(emailData)
	}

	return nil
}

func ValidateEmailFormat(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}
