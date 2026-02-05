package middleware

import (
	"net/http"
	"strings"

	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Error messages
const (
	errRoleNotFound            = "role not found"
	errInvalidRoleFormat       = "invalid role format"
	errInsufficientPermissions = "insufficient permissions"
)

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret   string
	AgentSecret string
}

// NoAuthMiddleware creates a middleware that sets default user context when auth is disabled
// This allows handlers that check for user_id to work in development mode
func NoAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set default anonymous user with admin role for full access
		c.Set("user_id", "anonymous")
		c.Set("role", "admin")
		c.Next()
	}
}

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// Reject refresh tokens - only access tokens are valid for API authentication
		tokenType, _ := claims["type"].(string)
		if tokenType != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["sub"])
		c.Set("role", claims["role"])

		c.Next()
	}
}

// AgentAuthMiddleware creates an agent authentication middleware
func AgentAuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentKey := c.GetHeader("X-Agent-Key")

		if agentKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "agent key required"})
			c.Abort()
			return
		}

		// For mTLS, the certificate validation happens at TLS layer
		// This is a secondary check using a pre-shared key
		if agentKey != config.AgentSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent key"})
			c.Abort()
			return
		}

		c.Set("is_agent", true)
		c.Next()
	}
}

// RoleMiddleware creates a role-based authorization middleware
func RoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": errRoleNotFound})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": errInvalidRoleFormat})
			c.Abort()
			return
		}

		for _, r := range requiredRoles {
			if r == roleStr {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": errInsufficientPermissions})
		c.Abort()
	}
}

// PermissionMiddleware creates a permission-based authorization middleware
func PermissionMiddleware(requiredPermissions ...entity.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": errRoleNotFound})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": errInvalidRoleFormat})
			c.Abort()
			return
		}

		userRole := entity.UserRole(roleStr)

		// Check if user has ALL required permissions
		for _, perm := range requiredPermissions {
			if !entity.HasPermission(userRole, perm) {
				c.JSON(http.StatusForbidden, gin.H{
					"error":     errInsufficientPermissions,
					"required":  string(perm),
					"user_role": roleStr,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireAnyPermission creates a middleware that requires at least one of the given permissions
func RequireAnyPermission(permissions ...entity.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": errRoleNotFound})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": errInvalidRoleFormat})
			c.Abort()
			return
		}

		userRole := entity.UserRole(roleStr)

		// Check if user has ANY of the required permissions
		for _, perm := range permissions {
			if entity.HasPermission(userRole, perm) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":     errInsufficientPermissions,
			"user_role": roleStr,
		})
		c.Abort()
	}
}
