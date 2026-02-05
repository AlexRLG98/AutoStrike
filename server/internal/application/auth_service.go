package application

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/repository"
	"autostrike/internal/infrastructure/persistence/sqlite"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Authentication errors
var (
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrUserInactive        = errors.New("user account is deactivated")
	ErrCannotDeactivateSelf = errors.New("cannot deactivate your own account")
	ErrLastAdmin           = errors.New("cannot deactivate the last admin user")
	ErrInvalidRole         = errors.New("invalid role")
)

// TokenResponse represents the response containing JWT tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Seconds until access token expires
	TokenType    string `json:"token_type"`
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo         repository.UserRepository
	jwtSecret        string
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	bcryptCost       int
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		jwtSecret:        jwtSecret,
		accessTokenTTL:   15 * time.Minute,
		refreshTokenTTL:  7 * 24 * time.Hour, // 7 days
		bcryptCost:       10,
	}
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login timestamp
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return s.generateTokens(user)
}

// Refresh generates new tokens from a valid refresh token
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	claims, err := s.validateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Check if it's a refresh token
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return s.generateTokens(user)
}

// GetCurrentUser returns the user for a given user ID
func (s *AuthService) GetCurrentUser(ctx context.Context, userID string) (*entity.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// ValidateToken validates a JWT token and returns its claims
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	return s.validateToken(tokenString)
}

// CreateUser creates a new user with hashed password
func (s *AuthService) CreateUser(ctx context.Context, username, email, password string, role entity.UserRole) (*entity.User, error) {
	// Check if username already exists
	existing, err := s.userRepo.FindByUsername(ctx, username)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}

	// Check if email already exists
	existing, err = s.userRepo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DefaultAdminResult contains information about the default admin creation
type DefaultAdminResult struct {
	Created           bool   // Whether a new admin was created
	GeneratedPassword string // The password if it was auto-generated (empty if from env var or no user created)
}

// EnsureDefaultAdmin creates a default admin user if no users exist
// Password is sourced from DEFAULT_ADMIN_PASSWORD env var, or a secure random password is generated
// Returns information about whether admin was created and the generated password (if any)
func (s *AuthService) EnsureDefaultAdmin(ctx context.Context) (*DefaultAdminResult, error) {
	users, err := s.userRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// If there are already users, don't create default admin
	if len(users) > 0 {
		return &DefaultAdminResult{Created: false}, nil
	}

	// Get password from environment variable or generate a secure random one
	password := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	generatedPassword := ""
	if password == "" {
		// Generate a secure random password (24 bytes = 32 chars base64)
		randomBytes := make([]byte, 24)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, err
		}
		password = base64.URLEncoding.EncodeToString(randomBytes)
		generatedPassword = password // Track that we generated this password
	}

	// Create default admin user
	_, err = s.CreateUser(ctx, "admin", "admin@autostrike.local", password, entity.RoleAdmin)
	if err != nil {
		return nil, err
	}

	return &DefaultAdminResult{
		Created:           true,
		GeneratedPassword: generatedPassword,
	}, nil
}

// generateTokens creates access and refresh tokens for a user
func (s *AuthService) generateTokens(user *entity.User) (*TokenResponse, error) {
	now := time.Now()

	// Access token
	accessClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": string(user.Role),
		"type": "access",
		"iat":  now.Unix(),
		"exp":  now.Add(s.accessTokenTTL).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(s.refreshTokenTTL).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// validateToken parses and validates a JWT token
func (s *AuthService) validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ===== User Management Methods =====

// GetAllUsers returns all users (for admin)
func (s *AuthService) GetAllUsers(ctx context.Context) ([]*entity.User, error) {
	return s.userRepo.FindAll(ctx)
}

// GetUser returns a user by ID
func (s *AuthService) GetUser(ctx context.Context, id string) (*entity.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateUser updates a user's details
func (s *AuthService) UpdateUser(ctx context.Context, id, username, email string, role entity.UserRole) (*entity.User, error) {
	// Validate role
	if !entity.IsValidRole(string(role)) {
		return nil, ErrInvalidRole
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Check if new username is taken by another user
	if username != user.Username {
		existing, err := s.userRepo.FindByUsername(ctx, username)
		if err == nil && existing != nil && existing.ID != id {
			return nil, ErrUserAlreadyExists
		}
	}

	// Check if new email is taken by another user
	if email != user.Email {
		existing, err := s.userRepo.FindByEmail(ctx, email)
		if err == nil && existing != nil && existing.ID != id {
			return nil, ErrUserAlreadyExists
		}
	}

	user.Username = username
	user.Email = email
	user.Role = role
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUserRole updates only the user's role
func (s *AuthService) UpdateUserRole(ctx context.Context, id string, role entity.UserRole) (*entity.User, error) {
	// Validate role
	if !entity.IsValidRole(string(role)) {
		return nil, ErrInvalidRole
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user.Role = role
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeactivateUser deactivates a user account (soft delete)
func (s *AuthService) DeactivateUser(ctx context.Context, id, currentUserID string) error {
	// Cannot deactivate yourself
	if id == currentUserID {
		return ErrCannotDeactivateSelf
	}

	// Use atomic operation to prevent race condition with concurrent admin deactivations
	err := s.userRepo.DeactivateAdminIfNotLast(ctx, id)
	if err != nil {
		// Map repository errors to application errors
		if errors.Is(err, sqlite.ErrUserNotFound) {
			return ErrUserNotFound
		}
		if errors.Is(err, sqlite.ErrLastAdmin) {
			return ErrLastAdmin
		}
		return err
	}
	return nil
}

// ReactivateUser reactivates a deactivated user account
func (s *AuthService) ReactivateUser(ctx context.Context, id string) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	if user.IsActive {
		return nil // Already active
	}

	return s.userRepo.Reactivate(ctx, id)
}

// ResetPassword resets a user's password
func (s *AuthService) ResetPassword(ctx context.Context, id, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, user)
}

// ChangePassword allows a user to change their own password
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	// Verify current password
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, user)
}
