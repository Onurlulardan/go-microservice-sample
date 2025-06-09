package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"forgecrud-backend/shared/clients"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/auth"
	utils "forgecrud-backend/shared/utils/auth"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// Login Request/Response structs
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"admin@forgecrud.com"`
	Password string `json:"password" binding:"required" example:"admin123"`
}

type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	User         UserInfo  `json:"user"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UserInfo struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	OrganizationID uuid.UUID `json:"organization_id"`
	RoleID         uuid.UUID `json:"role_id"`
	RoleName       string    `json:"role_name"`
	Status         string    `json:"status"`
}

// Register Request struct
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email" example:"user@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"securepassword123"`
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" binding:"required" example:"Doe"`
}

// Refresh Request struct
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// Refresh Response struct
type RefreshResponse struct {
	Token        string    `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string    `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt    time.Time `json:"expires_at" example:"2025-06-02T19:37:11.076935+03:00"`
}

// Validate Request struct
type ValidateRequest struct {
	Token string `json:"token" binding:"required"`
}

// Validate Response struct
type ValidateResponse struct {
	Valid     bool      `json:"valid"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Email     string    `json:"email,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Blacklist Request struct
type BlacklistRequest struct {
	Token string `json:"token" binding:"required"`
}

// CreateVerificationTokenRequest represents the request for creating verification token
type CreateVerificationTokenRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// CreateVerificationTokenResponse represents the response for creating verification token
type CreateVerificationTokenResponse struct {
	Token     string `json:"token"`
	FirstName string `json:"first_name"`
}

// POST /api/auth/login
// @Summary User login
// @Description Authenticate a user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login credentials"
// @Success 200 {object} handlers.LoginResponse "Successful login"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 429 {object} map[string]string "Too many login attempts"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Rate limiting Control (login attempt)
	clientIP := c.ClientIP()
	if err := h.checkRateLimit(req.Email, clientIP); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many login attempts. Please try again later."})
		return
	}

	// Find User by email
	var user models.User
	if err := h.db.Preload("Organization").Preload("Role").Where("email = ?", req.Email).First(&user).Error; err != nil {
		h.recordFailedLogin(req.Email, clientIP, "User not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if user is active
	if user.Status != "ACTIVE" {
		h.recordFailedLogin(req.Email, clientIP, "User inactive")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is inactive"})
		return
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		h.recordFailedLogin(req.Email, clientIP, "Invalid password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create JWT token
	var orgID, roleID uuid.UUID
	if user.OrganizationID != nil {
		orgID = *user.OrganizationID
	}
	if user.RoleID != nil {
		roleID = *user.RoleID
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, orgID, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	// Create Refresh Token
	refreshToken, err := utils.GenerateRefreshJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token"})
		return
	}

	// Set up user session
	sessionID, _ := utils.GenerateSessionID()
	expireDuration := utils.GetJWTExpireDuration()
	userSession := auth.UserSession{
		UserID:       user.ID,
		SessionID:    sessionID,
		TokenHash:    token[:32],
		RefreshToken: refreshToken,
		IPAddress:    clientIP,
		UserAgent:    c.GetHeader("User-Agent"),
		ExpiresAt:    time.Now().Add(expireDuration),
		IsActive:     true,
	}

	if err := h.db.Create(&userSession).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create session"})
		return
	}

	h.recordSuccessfulLogin(user.Email, clientIP)

	var roleName string
	if user.RoleID != nil {
		roleName = user.Role.Name
	}

	response := LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(expireDuration),
		User: UserInfo{
			ID:             user.ID,
			Email:          user.Email,
			FirstName:      user.FirstName,
			LastName:       user.LastName,
			OrganizationID: orgID,
			RoleID:         roleID,
			RoleName:       roleName,
			Status:         user.Status,
		},
	}

	c.JSON(http.StatusOK, response)
}

// POST /api/auth/logout
// @Summary User logout
// @Description Logout the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Logged out successfully"
// @Failure 400 {object} map[string]string "Token required"
// @Failure 401 {object} map[string]string "Invalid token"
// @Failure 500 {object} map[string]string "Could not logout"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token required"})
		return
	}

	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Validate JWT token
	claims, err := utils.ValidateJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Set Session passive
	tokenHash := tokenString[:32]
	userID, _ := uuid.Parse(claims.UserID)
	if err := h.db.Model(&auth.UserSession{}).
		Where("user_id = ? AND token_hash = ? AND is_active = ?", userID, tokenHash, true).
		Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// POST /api/auth/register
// @Summary Register new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param register body RegisterRequest true "User registration data"
// @Success 201 {object} handlers.LoginResponse "User registered successfully"
// @Failure 400 {object} map[string]string "Invalid request format or validation error"
// @Failure 409 {object} map[string]string "Email already exists"
// @Failure 429 {object} map[string]string "Too many registration attempts"
// @Failure 500 {object} map[string]string "Failed to register user"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Email validation
	if err := utils.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Password validation
	if err := utils.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check email uniqueness
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	// Create User
	user := models.User{
		Email:         req.Email,
		Password:      hashedPassword,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Status:        "ACTIVE",
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	// Send verification email automatically after registration
	verificationToken, err := utils.CreateEmailVerificationToken(h.db, user.ID)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully but verification email failed to send",
			"user": gin.H{
				"id":         user.ID,
				"email":      user.Email,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
			},
		})
		return
	}

	// Send verification email
	notificationClient := clients.NewNotificationClient()

	if err := notificationClient.SendWelcomeEmail(user.Email, user.FirstName, verificationToken.Token); err != nil {
		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully but verification email failed to send",
			"user": gin.H{
				"id":         user.ID,
				"email":      user.Email,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully. Please check your email to verify your account.",
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
		},
	})
}

// POST /api/auth/refresh
// @Summary Refresh JWT token
// @Description Refresh an expired JWT token using a valid refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body RefreshRequest true "Refresh token"
// @Success 200 {object} handlers.RefreshResponse "Successfully refreshed tokens"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 401 {object} map[string]string "Invalid refresh token or user inactive"
// @Failure 500 {object} map[string]string "Failed to generate new tokens"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Refresh token validation
	claims, err := utils.ValidateRefreshJWT(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// UserSession'ı bul
	userID, _ := uuid.Parse(claims.UserID)
	var userSession auth.UserSession
	if err := h.db.Where("user_id = ? AND refresh_token = ? AND is_active = ?",
		userID, req.RefreshToken, true).First(&userSession).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token not found or expired"})
		return
	}

	// User bilgilerini al
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Kullanıcı aktif mi kontrol et
	if user.Status != "ACTIVE" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is inactive"})
		return
	}

	// Yeni access token ve refresh token oluştur
	var orgID, roleID uuid.UUID
	if user.OrganizationID != nil {
		orgID = *user.OrganizationID
	}
	if user.RoleID != nil {
		roleID = *user.RoleID
	}

	newToken, err := utils.GenerateJWT(user.ID, user.Email, orgID, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	newRefreshToken, err := utils.GenerateRefreshJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token"})
		return
	}

	expireDuration := utils.GetJWTExpireDuration()
	userSession.TokenHash = newToken[:32]
	userSession.RefreshToken = newRefreshToken
	userSession.ExpiresAt = time.Now().Add(expireDuration)
	userSession.UpdatedAt = time.Now()

	if err := h.db.Save(&userSession).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update session"})
		return
	}

	response := RefreshResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(expireDuration),
	}

	c.JSON(http.StatusOK, response)
}

// POST /api/auth/validate
// @Summary Validate JWT token
// @Description Validate a JWT token and return its claims
// @Tags auth
// @Accept json
// @Produce json
// @Param validate body ValidateRequest true "JWT token to validate"
// @Success 200 {object} handlers.ValidateResponse "Token validation result"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Router /auth/validate [post]
func (h *AuthHandler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := utils.ValidateJWT(req.Token)
	if err != nil {
		c.JSON(http.StatusOK, ValidateResponse{
			Valid: false,
		})
		return
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		c.JSON(http.StatusOK, ValidateResponse{
			Valid: false,
		})
		return
	}

	userID, _ := uuid.Parse(claims.UserID)
	tokenHash := req.Token[:32]

	// Check if token is blacklisted
	var blacklistedToken auth.BlacklistedToken
	if err := h.db.Where("user_id = ? AND token_hash = ?", userID, tokenHash).First(&blacklistedToken).Error; err == nil {
		c.JSON(http.StatusOK, ValidateResponse{
			Valid: false,
		})
		return
	}

	var userSession auth.UserSession
	if err := h.db.Where("user_id = ? AND token_hash = ? AND is_active = ?",
		userID, tokenHash, true).First(&userSession).Error; err != nil {
		c.JSON(http.StatusOK, ValidateResponse{
			Valid: false,
		})
		return
	}

	response := ValidateResponse{
		Valid:     true,
		UserID:    userID,
		Email:     claims.Email,
		ExpiresAt: claims.ExpiresAt.Time,
	}

	c.JSON(http.StatusOK, response)
}

// POST /api/auth/blacklist
// @Summary Blacklist JWT token
// @Description Add a JWT token to the blacklist to invalidate it immediately
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param blacklist body BlacklistRequest true "JWT token to blacklist"
// @Success 200 {object} map[string]string "Token blacklisted successfully"
// @Failure 400 {object} map[string]string "Invalid request format"
// @Failure 401 {object} map[string]string "Invalid or expired token"
// @Failure 500 {object} map[string]string "Failed to blacklist token"
// @Router /auth/blacklist [post]
func (h *AuthHandler) Blacklist(c *gin.Context) {
	var req BlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate JWT token
	claims, err := utils.ValidateJWT(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Check if token is already expired
	if claims.ExpiresAt.Time.Before(time.Now()) {
		c.JSON(http.StatusOK, gin.H{"message": "Token already expired"})
		return
	}

	// Get token hash and user ID
	tokenHash := req.Token[:32]
	userID, _ := uuid.Parse(claims.UserID)

	// Create blacklisted token record
	blacklistedToken := auth.BlacklistedToken{
		UserID:        userID,
		TokenHash:     tokenHash,
		ExpiresAt:     claims.ExpiresAt.Time,
		BlacklistedAt: time.Now(),
	}

	// First check if token is already blacklisted
	var existingToken auth.BlacklistedToken
	if err := h.db.Where("user_id = ? AND token_hash = ?", userID, tokenHash).First(&existingToken).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Token already blacklisted"})
		return
	}

	// Save blacklisted token
	if err := h.db.Create(&blacklistedToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not blacklist token"})
		return
	}

	// Set related session to inactive
	h.db.Model(&auth.UserSession{}).
		Where("user_id = ? AND token_hash = ? AND is_active = ?", userID, tokenHash, true).
		Update("is_active", false)

	c.JSON(http.StatusOK, gin.H{"message": "Token blacklisted successfully"})
}

// CreateVerificationToken creates a new verification token for email verification
// @Summary Create verification token
// @Description Create a new verification token for user email verification
// @Tags auth
// @Accept json
// @Produce json
// @Param request body CreateVerificationTokenRequest true "Create verification token request"
// @Success 200 {object} CreateVerificationTokenResponse "Verification token created successfully"
// @Failure 400 {object} map[string]string "Invalid request or email already verified"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Failed to create verification token"
// @Router /auth/create-verification-token [post]
func (h *AuthHandler) CreateVerificationToken(c *gin.Context) {
	var req CreateVerificationTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already verified"})
		return
	}

	// Invalidate old verification tokens
	if err := utils.InvalidateOldVerificationTokens(h.db, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate old tokens"})
		return
	}

	// Create new verification token
	verificationToken, err := utils.CreateEmailVerificationToken(h.db, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification token"})
		return
	}

	c.JSON(http.StatusOK, CreateVerificationTokenResponse{
		Token:     verificationToken.Token,
		FirstName: user.FirstName,
	})
}

// VerifyEmail verifies the email using the provided token
// @Summary Verify email
// @Description Verify user's email using the provided token
// @Tags auth
// @Accept json
// @Produce json
// @Param token path string true "Verification token"
// @Success 200 {object} map[string]interface{} "Email verified successfully with auth tokens"
// @Failure 400 {object} map[string]string "Invalid token"
// @Failure 500 {object} map[string]string "Failed to verify email"
// @Router /auth/verify-email/{token} [get]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	user, err := utils.VerifyEmailToken(h.db, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orgID, roleID uuid.UUID
	if user.OrganizationID != nil {
		orgID = *user.OrganizationID
	}
	if user.RoleID != nil {
		roleID = *user.RoleID
	}

	authToken, err := utils.GenerateJWT(user.ID, user.Email, orgID, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token"})
		return
	}

	userResponse := map[string]interface{}{
		"id":         user.ID,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
		"status":     "ACTIVE",
	}

	if user.OrganizationID != nil {
		userResponse["organization_id"] = *user.OrganizationID
	}
	if user.RoleID != nil {
		userResponse["role_id"] = *user.RoleID
		var role models.Role
		if err = h.db.First(&role, *user.RoleID).Error; err == nil {
			userResponse["role_name"] = role.Name
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Email verified successfully",
		"user":          userResponse,
		"token":         authToken,
		"refresh_token": refreshToken,
		"expires_at":    time.Now().Add(utils.GetJWTExpireDuration()),
	})
}

// Rate limiting helper functions
func (h *AuthHandler) checkRateLimit(email, ipAddress string) error {
	var count int64
	h.db.Model(&auth.LoginAttempt{}).
		Where("(email = ? OR ip_address = ?) AND successful = ? AND created_at > ?",
			email, ipAddress, false, time.Now().Add(-15*time.Minute)).
		Count(&count)

	if count >= 5 {
		return gin.Error{Err: nil, Type: gin.ErrorTypePublic}
	}
	return nil
}

func (h *AuthHandler) recordFailedLogin(email, ipAddress, failureType string) {
	attempt := auth.LoginAttempt{
		Email:       email,
		IPAddress:   ipAddress,
		UserAgent:   "",
		Successful:  false,
		FailureType: failureType,
		Attempts:    1,
		LastAttempt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	h.db.Create(&attempt)
}

func (h *AuthHandler) recordSuccessfulLogin(email, ipAddress string) {
	attempt := auth.LoginAttempt{
		Email:       email,
		IPAddress:   ipAddress,
		UserAgent:   "",
		Successful:  true,
		Attempts:    1,
		LastAttempt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	h.db.Create(&attempt)
}
