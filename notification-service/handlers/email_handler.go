package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"forgecrud-backend/notification-service/services"
	"forgecrud-backend/shared/config"

	"github.com/gin-gonic/gin"
)

// EmailHandler handles email-related HTTP requests
type EmailHandler struct {
	emailService *services.EmailService
	config       *config.Config
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(emailService *services.EmailService, cfg *config.Config) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
		config:       cfg,
	}
}

// SendEmail godoc
// @Summary Send email
// @Description Send an email through the notification service
// @Tags email
// @Accept json
// @Produce json
// @Param email body services.EmailRequest true "Email request"
// @Success 200 {object} services.EmailResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/email/send [post]
func (eh *EmailHandler) SendEmail(c *gin.Context) {
	var request services.EmailRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := eh.emailService.SendEmail(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendWelcomeEmail godoc
// @Summary Send welcome/verification email
// @Description Send a welcome email with verification code using template
// @Tags email
// @Accept json
// @Produce json
// @Param email body WelcomeEmailRequest true "Welcome email request"
// @Success 200 {object} services.EmailResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/email/welcome [post]
func (eh *EmailHandler) SendWelcomeEmail(c *gin.Context) {
	var request WelcomeEmailRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := eh.emailService.SendWelcomeEmail(request.To, request.Name, request.VerificationCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send welcome email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendPasswordResetEmail godoc
// @Summary Send password reset email
// @Description Send a password reset email with reset code using template
// @Tags email
// @Accept json
// @Produce json
// @Param email body PasswordResetEmailRequest true "Password reset email request"
// @Success 200 {object} services.EmailResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/email/password-reset [post]
func (eh *EmailHandler) SendPasswordResetEmail(c *gin.Context) {
	var request PasswordResetEmailRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := eh.emailService.SendPasswordResetEmail(request.To, request.Name, request.ResetCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send password reset email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// VerificationEmailRequest represents the request for sending verification email
type VerificationEmailRequest struct {
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"first_name" binding:"required"`
	Token     string `json:"token" binding:"required"`
}

// ResendVerificationRequest represents the request for resending verification email
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SendVerificationEmail godoc
// @Summary Send verification email
// @Description Send email verification link to user
// @Tags email
// @Accept json
// @Produce json
// @Param request body VerificationEmailRequest true "Verification email request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/email/verification [post]
func (eh *EmailHandler) SendVerificationEmail(c *gin.Context) {
	var request VerificationEmailRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Create verification URL
	verificationURL := fmt.Sprintf("%s/auth/verify-email/%s", eh.config.FrontendURL, request.Token)

	// Send welcome email with verification link
	emailRequest := services.EmailRequest{
		To:         []string{request.Email},
		Subject:    "Welcome! Please verify your email",
		TemplateID: "welcome_verification",
		TemplateVars: map[string]interface{}{
			"FirstName":       request.FirstName,
			"VerificationURL": verificationURL,
		},
		IsHTML: true,
	}

	response, err := eh.emailService.SendEmail(emailRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send verification email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent successfully",
		"sent_at": response.SentAt,
	})
}

// ResendVerificationEmail godoc
// @Summary Resend verification email
// @Description Resend verification email to user after creating new token
// @Tags email
// @Accept json
// @Produce json
// @Param request body ResendVerificationRequest true "Resend verification request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/notifications/email/resend-verification [post]
func (eh *EmailHandler) ResendVerificationEmail(c *gin.Context) {
	var request ResendVerificationRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Call auth service to create new verification token
	tokenRequest := map[string]interface{}{
		"email": request.Email,
	}

	tokenRequestBytes, _ := json.Marshal(tokenRequest)
	resp, err := http.Post(
		fmt.Sprintf("%s/api/auth/create-verification-token", eh.config.AuthServiceURL),
		"application/json",
		bytes.NewBuffer(tokenRequestBytes),
	)

	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create new verification token",
		})
		return
	}

	var tokenResponse struct {
		Token     string `json:"token"`
		FirstName string `json:"first_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse token response",
		})
		return
	}

	// Send verification email
	verificationRequest := VerificationEmailRequest{
		Email:     request.Email,
		FirstName: tokenResponse.FirstName,
		Token:     tokenResponse.Token,
	}

	// Use the existing SendVerificationEmail logic
	verificationURL := fmt.Sprintf("%s/auth/verify-email/%s", eh.config.FrontendURL, verificationRequest.Token)

	emailRequest := services.EmailRequest{
		To:         []string{verificationRequest.Email},
		Subject:    "Verification Email Resent",
		TemplateID: "welcome_verification",
		TemplateVars: map[string]interface{}{
			"FirstName":       verificationRequest.FirstName,
			"VerificationURL": verificationURL,
		},
		IsHTML: true,
	}

	response, err := eh.emailService.SendEmail(emailRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send verification email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email resent successfully",
		"sent_at": response.SentAt,
	})
}

// Request structures for convenience endpoints
type WelcomeEmailRequest struct {
	To               string `json:"to" binding:"required,email"`
	Name             string `json:"name" binding:"required"`
	VerificationCode string `json:"verification_code" binding:"required"`
}

type PasswordResetEmailRequest struct {
	To        string `json:"to" binding:"required,email"`
	Name      string `json:"name" binding:"required"`
	ResetCode string `json:"reset_code" binding:"required"`
}
