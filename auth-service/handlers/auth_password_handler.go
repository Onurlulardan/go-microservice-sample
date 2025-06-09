package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"forgecrud-backend/shared/clients"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/auth"
	utils "forgecrud-backend/shared/utils/auth"
)

// Password Management Request/Response structs

// ChangePasswordRequest represents the request body for changing a password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// ForgotPasswordRequest represents the request body for forgot password
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the request body for resetting a password
type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// ChangePassword changes a user's password after verifying the current password
// @Summary Change password
// @Description Change user's password after verifying current password
// @Tags auth-password
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]string "Password changed successfully"
// @Failure 400 {object} map[string]string "Invalid request format or validation error"
// @Failure 401 {object} map[string]string "User not authenticated or incorrect password"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Failed to update password"
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Validate new password
	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure new password is different from current password
	if req.CurrentPassword == req.NewPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be different from current password"})
		return
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	// Update user's password
	if err := h.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update password"})
		return
	}

	// Optionally, invalidate all user's sessions except the current one
	// This is a security measure to log out the user from all other devices
	currentTokenHash, _ := c.Get("tokenHash")
	if err := h.db.Model(&auth.UserSession{}).
		Where("user_id = ? AND token_hash != ?", userID, currentTokenHash).
		Update("is_active", false).Error; err != nil {
		// Non-critical error, just log it
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ForgotPassword initiates the password reset process by sending a reset link to the user's email
// @Summary Forgot password
// @Description Initiates password reset process by sending a reset link to the user's email
// @Tags auth-password
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email for password reset"
// @Success 200 {object} map[string]string "Password reset email sent"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 429 {object} map[string]string "Too many password reset attempts"
// @Failure 500 {object} map[string]string "Failed to process request"
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the rate limit has been exceeded for this email/IP
	clientIP := c.ClientIP()
	if err := h.checkPasswordResetRateLimit(req.Email, clientIP); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many password reset attempts. Please try again later."})
		return
	}

	// Find user by email
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// For security reasons, don't reveal whether the email exists or not
		c.JSON(http.StatusOK, gin.H{"message": "If a user with this email exists, a password reset link will be sent"})
		return
	}

	// Invalidate old reset tokens for this user
	if err := h.invalidateOldPasswordResetTokens(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process request"})
		return
	}

	// Create a new password reset token
	resetToken, err := h.createPasswordResetToken(user.ID, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create reset token"})
		return
	}

	// Send password reset email
	notificationClient := clients.NewNotificationClient()
	if err := notificationClient.SendPasswordResetEmail(user.Email, user.FirstName, resetToken.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not send reset email"})
		return
	}

	// Record the reset attempt
	h.recordPasswordResetAttempt(req.Email, clientIP, true)

	c.JSON(http.StatusOK, gin.H{"message": "If a user with this email exists, a password reset link will be sent"})
}

// ResetPassword resets a user's password using a valid reset token
// @Summary Reset password
// @Description Reset user's password using a valid reset token
// @Tags auth-password
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Password reset data with token"
// @Success 200 {object} map[string]string "Password reset successful"
// @Failure 400 {object} map[string]string "Invalid request format or token"
// @Failure 500 {object} map[string]string "Failed to update password"
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate token and get user
	user, err := h.validatePasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate new password
	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	// Update user's password
	if err := h.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update password"})
		return
	}

	// Mark the token as used
	if err := h.markResetTokenAsUsed(req.Token); err != nil {
		// Non-critical error, just log it
	}

	// Invalidate all user's sessions
	if err := h.db.Model(&auth.UserSession{}).
		Where("user_id = ?", user.ID).
		Update("is_active", false).Error; err != nil {
		// Non-critical error, just log it
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successful. You can now log in with your new password."})
}

// Helper functions

// checkPasswordResetRateLimit checks if the rate limit has been exceeded for password reset attempts
func (h *AuthHandler) checkPasswordResetRateLimit(email, ipAddress string) error {
	var count int64
	h.db.Model(&auth.PasswordResetAttempt{}).
		Where("(email = ? OR ip_address = ?) AND created_at > ?",
			email, ipAddress, time.Now().Add(-15*time.Minute)).
		Count(&count)

	if count >= 3 {
		return gin.Error{Err: nil, Type: gin.ErrorTypePublic}
	}
	return nil
}

// invalidateOldPasswordResetTokens marks all existing reset tokens for a user as expired
func (h *AuthHandler) invalidateOldPasswordResetTokens(userID uuid.UUID) error {
	return h.db.Model(&auth.PasswordResetToken{}).
		Where("user_id = ? AND used = ?", userID, false).
		Update("expired", true).Error
}

// createPasswordResetToken creates a new password reset token for a user
func (h *AuthHandler) createPasswordResetToken(userID uuid.UUID, ipAddress string) (*auth.PasswordResetToken, error) {
	// Generate a unique token
	tokenString, err := utils.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	// Create reset token record
	resetToken := auth.PasswordResetToken{
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Used:      false,
		Expired:   false,
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}

	// Save to database
	if err := h.db.Create(&resetToken).Error; err != nil {
		return nil, err
	}

	return &resetToken, nil
}

// validatePasswordResetToken validates a password reset token and returns the associated user
func (h *AuthHandler) validatePasswordResetToken(token string) (*models.User, error) {
	// Find token in database
	var resetToken auth.PasswordResetToken
	if err := h.db.Where("token = ? AND used = ? AND expired = ?",
		token, false, false).First(&resetToken).Error; err != nil {
		return nil, err
	}

	// Check if token is expired
	if resetToken.ExpiresAt.Before(time.Now()) {
		// Mark token as expired
		h.db.Model(&resetToken).Update("expired", true)
		return nil, fmt.Errorf("password reset token has expired")
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, resetToken.UserID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// markResetTokenAsUsed marks a password reset token as used
func (h *AuthHandler) markResetTokenAsUsed(token string) error {
	return h.db.Model(&auth.PasswordResetToken{}).
		Where("token = ?", token).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": time.Now(),
		}).Error
}

// recordPasswordResetAttempt records a password reset attempt
func (h *AuthHandler) recordPasswordResetAttempt(email, ipAddress string, successful bool) {
	attempt := auth.PasswordResetAttempt{
		Email:      email,
		IPAddress:  ipAddress,
		UserAgent:  "",
		Successful: successful,
		CreatedAt:  time.Now(),
	}
	h.db.Create(&attempt)
}
