package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/auth"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GenerateVerificationToken generates a secure random token
func GenerateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateEmailVerificationToken creates a new email verification token for a user
func CreateEmailVerificationToken(db *gorm.DB, userID uuid.UUID) (*auth.EmailVerificationToken, error) {

	token, err := GenerateVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	verificationToken := &auth.EmailVerificationToken{
		UserID:    userID,
		Token:     token,
		Email:     "",
		ExpiresAt: time.Now().Add(GetJWTExpireDuration()),
		Verified:  false,
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	verificationToken.Email = user.Email

	if err := db.Create(verificationToken).Error; err != nil {
		return nil, fmt.Errorf("failed to create verification token: %w", err)
	}

	return verificationToken, nil
}

// VerifyEmailToken verifies the email verification token and marks user as verified
func VerifyEmailToken(db *gorm.DB, token string) (*models.User, error) {
	var verificationToken auth.EmailVerificationToken

	if err := db.Preload("User").Where("token = ? AND verified = ? AND expires_at > ?",
		token, false, time.Now()).First(&verificationToken).Error; err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}

	verificationToken.Verified = true
	verificationToken.VerifiedAt = &time.Time{}
	*verificationToken.VerifiedAt = time.Now()
	if err := db.Save(&verificationToken).Error; err != nil {
		return nil, fmt.Errorf("failed to update token: %w", err)
	}

	verificationToken.User.EmailVerified = true
	if err := db.Save(&verificationToken.User).Error; err != nil {
		return nil, fmt.Errorf("failed to verify user email: %w", err)
	}

	return &verificationToken.User, nil
}

// InvalidateOldVerificationTokens marks all old tokens for a user as verified
func InvalidateOldVerificationTokens(db *gorm.DB, userID uuid.UUID) error {
	return db.Model(&auth.EmailVerificationToken{}).
		Where("user_id = ? AND verified = ?", userID, false).
		Update("verified", true).Error
}

// CleanupExpiredTokens removes expired verification tokens
func CleanupExpiredTokens(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).
		Delete(&auth.EmailVerificationToken{}).Error
}
