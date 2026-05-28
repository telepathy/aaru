package service

import (
	"fmt"
	"time"

	"aaru/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	jwtSecret []byte
}

func NewAuthService(secret string) *AuthService {
	return &AuthService{jwtSecret: []byte(secret)}
}

// GenerateToken 生成JWT token
func (a *AuthService) GenerateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ParseToken 解析JWT token
func (a *AuthService) ParseToken(tokenStr string) (uint, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	if err != nil {
		return 0, "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, "", fmt.Errorf("invalid token")
	}
	userID := uint(claims["user_id"].(float64))
	username := claims["username"].(string)
	return userID, username, nil
}

// MockGitlabLogin 模拟Gitlab SSO登录
// 在一个真实系统中，这里会重定向到Gitlab进行OAuth认证
// 在mock模式下，直接根据用户名创建/查找用户
func (a *AuthService) MockGitlabLogin(username string, users []string) bool {
	for _, u := range users {
		if u == username {
			return true
		}
	}
	return false
}
