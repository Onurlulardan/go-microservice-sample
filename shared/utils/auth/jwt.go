package utils

import (
	"errors"
	"strconv"
	"time"

	"forgecrud-backend/shared/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID         string `json:"user_id"`
	Email          string `json:"email"`
	OrganizationID string `json:"organization_id"`
	RoleID         string `json:"role_id"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte(getJWTSecret())

func getJWTSecret() string {
	cfg := config.GetConfig()
	if cfg.JWTSecret == "" {
		return "fallback-secret-key-for-development"
	}
	return cfg.JWTSecret
}

// GetJWTExpireDuration gets JWT expiration duration from config
func GetJWTExpireDuration() time.Duration {
	cfg := config.GetConfig()
	if cfg.JWTExpireHours == "" {
		return 24 * time.Hour
	}

	hours, err := strconv.Atoi(cfg.JWTExpireHours)
	if err != nil {
		return 24 * time.Hour
	}

	return time.Duration(hours) * time.Hour
}

// GetJWTRefreshExpireDuration gets JWT refresh token expiration duration from config
func GetJWTRefreshExpireDuration() time.Duration {
	cfg := config.GetConfig()
	if cfg.JWTRefreshExpireDays == "" {
		return 7 * 24 * time.Hour
	}

	days, err := strconv.Atoi(cfg.JWTRefreshExpireDays)
	if err != nil {
		return 7 * 24 * time.Hour
	}

	return time.Duration(days) * 24 * time.Hour
}

// Generate JWT token
func GenerateJWT(userID uuid.UUID, email string, organizationID uuid.UUID, roleID uuid.UUID) (string, error) {
	expireDuration := GetJWTExpireDuration()

	claims := Claims{
		UserID:         userID.String(),
		Email:          email,
		OrganizationID: organizationID.String(),
		RoleID:         roleID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Generate Refresh token
func GenerateRefreshJWT(userID uuid.UUID, email string) (string, error) {
	refreshExpireDuration := GetJWTRefreshExpireDuration()

	claims := Claims{
		UserID: userID.String(),
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshExpireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Validate JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Check if JWT token is expired
func IsTokenExpired(tokenString string) bool {
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return true
	}

	return claims.ExpiresAt.Before(time.Now())
}

// Refresh JWT token validate
func ValidateRefreshJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}
