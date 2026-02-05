package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/infrastructure/persistence/sqlite"

	"github.com/golang-jwt/jwt/v5"
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

func (m *mockUserRepo) FindActive(ctx context.Context) ([]*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.User, 0)
	for _, user := range m.users {
		if user.IsActive {
			result = append(result, user)
		}
	}
	return result, nil
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, id string) error {
	user, ok := m.users[id]
	if !ok {
		return sql.ErrNoRows
	}
	now := time.Now()
	user.LastLoginAt = &now
	return nil
}

func (m *mockUserRepo) Deactivate(ctx context.Context, id string) error {
	user, ok := m.users[id]
	if !ok {
		return sql.ErrNoRows
	}
	user.IsActive = false
	return nil
}

func (m *mockUserRepo) Reactivate(ctx context.Context, id string) error {
	user, ok := m.users[id]
	if !ok {
		return sql.ErrNoRows
	}
	user.IsActive = true
	return nil
}

func (m *mockUserRepo) CountByRole(ctx context.Context, role entity.UserRole) (int, error) {
	count := 0
	for _, user := range m.users {
		if user.Role == role && user.IsActive {
			count++
		}
	}
	return count, nil
}

func (m *mockUserRepo) DeactivateAdminIfNotLast(ctx context.Context, id string) error {
	user, ok := m.users[id]
	if !ok {
		return sqlite.ErrUserNotFound
	}
	// If admin, check if last admin
	if user.Role == entity.RoleAdmin {
		count := 0
		for _, u := range m.users {
			if u.Role == entity.RoleAdmin && u.IsActive && u.ID != id {
				count++
			}
		}
		if count == 0 {
			return sqlite.ErrLastAdmin
		}
	}
	user.IsActive = false
	return nil
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
		IsActive:     true,
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
		IsActive:     true,
	}

	ctx := context.Background()
	_, err := service.Login(ctx, "testuser", "wrongpassword")

	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		IsActive:     false,
	}

	ctx := context.Background()
	_, err := service.Login(ctx, "testuser", "password123")

	if err != ErrUserInactive {
		t.Errorf("Expected ErrUserInactive, got %v", err)
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
		IsActive:     true,
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
		IsActive:     true,
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
	result, err := service.EnsureDefaultAdmin(ctx)

	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}

	// Should indicate that admin was created
	if !result.Created {
		t.Error("Expected Created to be true")
	}

	// Should have generated a password (since no env var set)
	if result.GeneratedPassword == "" {
		t.Error("Expected GeneratedPassword to be set")
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
	result, err := service.EnsureDefaultAdmin(ctx)

	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed: %v", err)
	}

	// Should indicate that no admin was created
	if result.Created {
		t.Error("Expected Created to be false")
	}

	// Should not have generated a password
	if result.GeneratedPassword != "" {
		t.Error("Expected GeneratedPassword to be empty")
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
		IsActive:     true,
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
		IsActive:     true,
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
		IsActive:     true,
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

func TestAuthService_GetAllUsers(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1"}
	repo.users["user-2"] = &entity.User{ID: "user-2", Username: "user2"}

	ctx := context.Background()
	users, err := service.GetAllUsers(ctx)

	if err != nil {
		t.Fatalf("GetAllUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestAuthService_GetUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     entity.RoleOperator,
	}

	ctx := context.Background()
	user, err := service.GetUser(ctx, "user-1")

	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %q, want 'testuser'", user.Username)
	}
}

func TestAuthService_GetUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.GetUser(ctx, "nonexistent")

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_UpdateUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "oldname",
		Email:    "old@example.com",
		Role:     entity.RoleViewer,
	}

	ctx := context.Background()
	user, err := service.UpdateUser(ctx, "user-1", "newname", "new@example.com", entity.RoleOperator)

	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if user.Username != "newname" {
		t.Errorf("Username = %q, want 'newname'", user.Username)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %q, want 'new@example.com'", user.Email)
	}
	if user.Role != entity.RoleOperator {
		t.Errorf("Role = %q, want 'operator'", user.Role)
	}
}

func TestAuthService_UpdateUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.UpdateUser(ctx, "nonexistent", "name", "email@test.com", entity.RoleViewer)

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_UpdateUser_DuplicateUsername(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1", Email: "user1@test.com"}
	repo.users["user-2"] = &entity.User{ID: "user-2", Username: "user2", Email: "user2@test.com"}

	ctx := context.Background()
	_, err := service.UpdateUser(ctx, "user-1", "user2", "user1@test.com", entity.RoleViewer)

	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestAuthService_UpdateUser_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1", Email: "user1@test.com"}
	repo.users["user-2"] = &entity.User{ID: "user-2", Username: "user2", Email: "user2@test.com"}

	ctx := context.Background()
	_, err := service.UpdateUser(ctx, "user-1", "user1", "user2@test.com", entity.RoleViewer)

	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestAuthService_UpdateUser_InvalidRole(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1", Email: "user1@test.com"}

	ctx := context.Background()
	_, err := service.UpdateUser(ctx, "user-1", "user1", "user1@test.com", entity.UserRole("invalid"))

	if err != ErrInvalidRole {
		t.Errorf("Expected ErrInvalidRole, got %v", err)
	}
}

func TestAuthService_UpdateUserRole_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "testuser",
		Role:     entity.RoleViewer,
	}

	ctx := context.Background()
	user, err := service.UpdateUserRole(ctx, "user-1", entity.RoleAdmin)

	if err != nil {
		t.Fatalf("UpdateUserRole failed: %v", err)
	}
	if user.Role != entity.RoleAdmin {
		t.Errorf("Role = %q, want 'admin'", user.Role)
	}
}

func TestAuthService_UpdateUserRole_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.UpdateUserRole(ctx, "nonexistent", entity.RoleAdmin)

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_UpdateUserRole_InvalidRole(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1"}

	ctx := context.Background()
	_, err := service.UpdateUserRole(ctx, "user-1", entity.UserRole("invalid"))

	if err != ErrInvalidRole {
		t.Errorf("Expected ErrInvalidRole, got %v", err)
	}
}

func TestAuthService_DeactivateUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1", Role: entity.RoleViewer, IsActive: true}
	repo.users["admin-1"] = &entity.User{ID: "admin-1", Username: "admin", Role: entity.RoleAdmin, IsActive: true}

	ctx := context.Background()
	err := service.DeactivateUser(ctx, "user-1", "admin-1")

	if err != nil {
		t.Fatalf("DeactivateUser failed: %v", err)
	}
	if repo.users["user-1"].IsActive {
		t.Error("User should be deactivated")
	}
}

func TestAuthService_DeactivateUser_CannotDeactivateSelf(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["admin-1"] = &entity.User{ID: "admin-1", Username: "admin", Role: entity.RoleAdmin, IsActive: true}

	ctx := context.Background()
	err := service.DeactivateUser(ctx, "admin-1", "admin-1")

	if err != ErrCannotDeactivateSelf {
		t.Errorf("Expected ErrCannotDeactivateSelf, got %v", err)
	}
}

func TestAuthService_DeactivateUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.DeactivateUser(ctx, "nonexistent", "admin-1")

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_DeactivateUser_LastAdmin(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["admin-1"] = &entity.User{ID: "admin-1", Username: "admin", Role: entity.RoleAdmin, IsActive: true}

	ctx := context.Background()
	err := service.DeactivateUser(ctx, "admin-1", "other-user")

	if err != ErrLastAdmin {
		t.Errorf("Expected ErrLastAdmin, got %v", err)
	}
}

func TestAuthService_ReactivateUser_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1", IsActive: false}

	ctx := context.Background()
	err := service.ReactivateUser(ctx, "user-1")

	if err != nil {
		t.Fatalf("ReactivateUser failed: %v", err)
	}
	if !repo.users["user-1"].IsActive {
		t.Error("User should be reactivated")
	}
}

func TestAuthService_ReactivateUser_AlreadyActive(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	repo.users["user-1"] = &entity.User{ID: "user-1", Username: "user1", IsActive: true}

	ctx := context.Background()
	err := service.ReactivateUser(ctx, "user-1")

	if err != nil {
		t.Errorf("ReactivateUser should not fail for already active user: %v", err)
	}
}

func TestAuthService_ReactivateUser_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.ReactivateUser(ctx, "nonexistent")

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	oldHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "user1",
		PasswordHash: string(oldHash),
	}

	ctx := context.Background()
	err := service.ResetPassword(ctx, "user-1", "newpassword")

	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	// Verify new password works
	err = bcrypt.CompareHashAndPassword([]byte(repo.users["user-1"].PasswordHash), []byte("newpassword"))
	if err != nil {
		t.Error("New password hash verification failed")
	}
}

func TestAuthService_ResetPassword_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.ResetPassword(ctx, "nonexistent", "newpassword")

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	currentHash, _ := bcrypt.GenerateFromPassword([]byte("currentpassword"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "user1",
		PasswordHash: string(currentHash),
	}

	ctx := context.Background()
	err := service.ChangePassword(ctx, "user-1", "currentpassword", "newpassword")

	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	// Verify new password works
	err = bcrypt.CompareHashAndPassword([]byte(repo.users["user-1"].PasswordHash), []byte("newpassword"))
	if err != nil {
		t.Error("New password hash verification failed")
	}
}

func TestAuthService_ChangePassword_InvalidCurrentPassword(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	currentHash, _ := bcrypt.GenerateFromPassword([]byte("currentpassword"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "user1",
		PasswordHash: string(currentHash),
	}

	ctx := context.Background()
	err := service.ChangePassword(ctx, "user-1", "wrongpassword", "newpassword")

	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_ChangePassword_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.ChangePassword(ctx, "nonexistent", "current", "new")

	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_ValidateToken_ExpiredToken(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	// Manually create an expired token
	expiredClaims := jwt.MapClaims{
		"sub":  "user-1",
		"role": "admin",
		"type": "access",
		"iat":  time.Now().Add(-2 * time.Hour).Unix(),
		"exp":  time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString([]byte("test-secret"))

	_, err := service.ValidateToken(expiredTokenString)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestAuthService_Login_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.Login(ctx, "user", "pass")

	if err == nil {
		t.Error("Expected error from Login")
	}
	if err.Error() != "database error" {
		t.Errorf("Expected 'database error', got %v", err)
	}
}

func TestAuthService_GetCurrentUser_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.GetCurrentUser(ctx, "user-1")

	if err == nil {
		t.Error("Expected error from GetCurrentUser")
	}
}

func TestAuthService_Refresh_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	service := NewAuthService(repo, "test-secret")

	// Create valid refresh token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("pass"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		IsActive:     true,
	}

	ctx := context.Background()
	tokens, _ := service.Login(ctx, "testuser", "pass")

	// Now set error
	repo.findErr = errors.New("database error")

	_, err := service.Refresh(ctx, tokens.RefreshToken)
	if err == nil {
		t.Error("Expected error from Refresh")
	}
}

func TestAuthService_GetUser_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.GetUser(ctx, "user-1")

	if err == nil {
		t.Error("Expected error from GetUser")
	}
}

func TestAuthService_UpdateUser_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.UpdateUser(ctx, "user-1", "name", "email@test.com", entity.RoleViewer)

	if err == nil {
		t.Error("Expected error from UpdateUser")
	}
}

func TestAuthService_UpdateUserRole_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.UpdateUserRole(ctx, "user-1", entity.RoleAdmin)

	if err == nil {
		t.Error("Expected error from UpdateUserRole")
	}
}

func TestAuthService_DeactivateUser_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.DeactivateUser(ctx, "user-1", "admin-1")

	if err == nil {
		t.Error("Expected error from DeactivateUser")
	}
}

func TestAuthService_ReactivateUser_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.ReactivateUser(ctx, "user-1")

	if err == nil {
		t.Error("Expected error from ReactivateUser")
	}
}

func TestAuthService_ResetPassword_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.ResetPassword(ctx, "user-1", "newpassword")

	if err == nil {
		t.Error("Expected error from ResetPassword")
	}
}

func TestAuthService_ChangePassword_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.findErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	err := service.ChangePassword(ctx, "user-1", "current", "new")

	if err == nil {
		t.Error("Expected error from ChangePassword")
	}
}

func TestAuthService_CreateUser_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	repo.createErr = errors.New("database error")
	service := NewAuthService(repo, "test-secret")

	ctx := context.Background()
	_, err := service.CreateUser(ctx, "newuser", "new@test.com", "password", entity.RoleViewer)

	if err == nil {
		t.Error("Expected error from CreateUser")
	}
}
