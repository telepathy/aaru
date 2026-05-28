package middleware

import (
	"net/http"
	"strings"

	"aaru/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// RequireAuth 要求用户已登录
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID, username, err := m.authService.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", userID)
		c.Set("username", username)
		c.Next()
	}
}

// OptionalAuth 可选认证（不强制要求）
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr != "" {
			userID, username, err := m.authService.ParseToken(tokenStr)
			if err == nil {
				c.Set("user_id", userID)
				c.Set("username", username)
			}
		}
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	// 从Authorization header取
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// 从cookie取
	token, _ := c.Cookie("token")
	return token
}
