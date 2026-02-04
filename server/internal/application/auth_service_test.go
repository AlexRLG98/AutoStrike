package application

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"autostrike/internal/domain/entity"

	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo implements repository.UserRepository for testing
type mockUserRepo struct {
	users     map[string]*entity.User
	findErr   error
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*entity.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *entity.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	user, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) FindAll(ctx context.Context) ([]*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.User, 0, len(m.users))
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, nil
}

func TestNewAuthService(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	if service == nil {
		t.Fatal("NewAuthService returned nil")
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	// Create a user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Role:         entity.RoleAdmin,
	}

	ctx := context.Background()
	tokens, err := service.Login(ctx, "testuser", "password123")

	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("Tokens should not be nil")
	}
	if tokens.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if tokens.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want 'Bearer'", tokens.TokenType)
	}
	if tokens.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	ctx := context.Background()
	_, err := service.Login(ctx, "testuser", "wrongpassword")

	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.Login(ctx, "nonexistent", "password")

	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Refresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	// Create a user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		Role:         entity.RoleAdmin,
	}

	ctx := context.Background()

	// Login to get tokens
	tokens, err := service.Login(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Refresh tokens
	newTokens, err := service.Refresh(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	if newTokens.AccessToken == "" {
		t.Error("New AccessToken should not be empty")
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.Refresh(ctx, "invalid-token")

	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_Refresh_WrongTokenType(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	ctx := context.Background()
	tokens, _ := service.Login(ctx, "testuser", "password123")

	// Try to refresh with access token (should fail)
	_, err := service.Refresh(ctx, tokens.AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken when using access token for refresh, got %v", err)
	}
}

func TestAuthService_GetCurrentUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     entity.RoleAdmin,
	}

	ctx := context.Background()
	user, err := service.GetCurrentUser(ctx, "user-1")

	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %q, want 'testuser'", user.Username)
	}
}

func TestAuthService_GetCurrentUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.GetCurrentUser(ctx, "nonexistent")

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_CreateUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	user, err := service.CreateUser(ctx, "newuser", "new@example.com", "password123", entity.RoleViewer)

	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Username != "newuser" {
		t.Errorf("Username = %q, want 'newuser'", user.Username)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want 'new@example.com'", user.Email)
	}
	if user.Role != entity.RoleViewer {
		t.Errorf("Role = %q, want 'viewer'", user.Role)
	}
	if user.PasswordHash == "" {
		t.Error("PasswordHash should not be empty")
	}
	if user.PasswordHash == "password123" {
		t.Error("PasswordHash should be hashed, not plain text")
	}
}

func TestAuthService_CreateUser_DuplicateUsername(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "existing",
		Email:    "existing@example.com",
	}

	ctx := context.Background()
	_, err := service.CreateUser(ctx, "existing", "new@example.com", "password123", entity.RoleViewer)

	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestAuthService_CreateUser_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "existing",
		Email:    "existing@example.com",
	}

	ctx := context.Background()
	_, err := service.CreateUser(ctx, "newuser", "existing@example.com", "password123", entity.RoleViewer)

	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestAuthService_EnsureDefaultAdmin_CreatesAdmin(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.EnsureDefaultAdmin(ctx)

	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}

	// Should have created one user
	if len(repo.users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(repo.users))
	}

	// Find the admin user
	var admin *entity.User
	for _, u := range repo.users {
		admin = u
		break
	}

	if admin.Username != "admin" {
		t.Errorf("Username = %q, want 'admin'", admin.Username)
	}
	if admin.Role != entity.RoleAdmin {
		t.Errorf("Role = %q, want 'admin'", admin.Role)
	}
}

func TestAuthService_EnsureDefaultAdmin_SkipsIfUsersExist(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	// Add existing user
	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "existing",
	}

	ctx := context.Background()
	err := service.EnsureDefaultAdmin(ctx)

	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}

	// Should still have only one user (no new admin created)
	if len(repo.users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(repo.users))
	}
}

func TestAuthService_ValidateToken_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		Role:         entity.RoleAdmin,
	}

	ctx := context.Background()
	tokens, _ := service.Login(ctx, "testuser", "password123")

	claims, err := service.ValidateToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims["sub"] != "user-1" {
		t.Errorf("claims['sub'] = %v, want 'user-1'", claims["sub"])
	}
	if claims["role"] != "admin" {
		t.Errorf("claims['role'] = %v, want 'admin'", claims["role"])
	}
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	_, err := service.ValidateToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_HashPassword(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hash, err := service.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "mypassword" {
		t.Error("Hash should not equal the plain password")
	}

	// Verify the hash works with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("mypassword"))
	if err != nil {
		t.Errorf("Hash verification failed: %v", err)
	}
}

func TestTokenResponse_Struct(t *testing.T) {
	resp := TokenResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}

	if resp.AccessToken != "access" {
		t.Errorf("AccessToken = %q, want 'access'", resp.AccessToken)
	}
	if resp.RefreshToken != "refresh" {
		t.Errorf("RefreshToken = %q, want 'refresh'", resp.RefreshToken)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("ExpiresIn = %d, want 900", resp.ExpiresIn)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want 'Bearer'", resp.TokenType)
	}
}

func TestAuthService_Refresh_UserDeleted(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	ctx := context.Background()
	tokens, _ := service.Login(ctx, "testuser", "password123")

	// Delete the user
	delete(repo.users, "user-1")

	// Try to refresh
	_, err := service.Refresh(ctx, tokens.RefreshToken)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_TokenExpiry(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	ctx := context.Background()
	tokens, _ := service.Login(ctx, "testuser", "password123")

	claims, _ := service.ValidateToken(tokens.AccessToken)

	// Check expiry is set (should be ~15 minutes from now)
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim not found or not a float64")
	}

	expTime := time.Unix(int64(exp), 0)
	now := time.Now()

	// Expiry should be between 14 and 16 minutes from now
	diff := expTime.Sub(now)
	if diff < 14*time.Minute || diff > 16*time.Minute {
		t.Errorf("Token expiry = %v from now, expected ~15 minutes", diff)
	}
}
