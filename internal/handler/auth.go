package handler

import (
	"net/http"

	"aaru/internal/model"
	"aaru/internal/service"
	"aaru/internal/store"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
	store       *store.DBStore
	mockUsers   []string
}

func NewAuthHandler(authService *service.AuthService, store *store.DBStore, mockUsers []string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		store:       store,
		mockUsers:   mockUsers,
	}
}

// MockLogin 模拟Gitlab SSO登录页面
func (h *AuthHandler) MockLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Users": h.mockUsers,
	})
}

// MockCallback 模拟Gitlab SSO回调
func (h *AuthHandler) MockCallback(c *gin.Context) {
	username := c.PostForm("username")
	if username == "" {
		username = c.Query("username")
	}
	if username == "" {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if !h.authService.MockGitlabLogin(username, h.mockUsers) {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Users":   h.mockUsers,
			"Error":   "无效的用户名",
			"Message": "请选择有效的用户登录",
		})
		return
	}

	// 查找或创建用户
	user, err := h.store.GetUserByUsername(username)
	if err != nil {
		// 创建新用户（用用户名生成唯一 GitLabID）
		var hash int64
		for _, c := range username {
			hash = hash*31 + int64(c)
		}
		if hash < 0 {
			hash = -hash
		}
		user = &model.User{
			Username:  username,
			Email:     username + "@example.com",
			GitlabID:  hash,
			AvatarURL: "",
		}
		if err := h.store.CreateUser(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create user failed"})
			return
		}
		// 新用户自动分配admin角色
		roles, _ := h.store.ListRoles()
		for _, role := range roles {
			if role.Name == "admin" {
				h.store.SetUserRoles(user.ID, []uint{role.ID})
				break
			}
		}
	}

	token, err := h.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate token failed"})
		return
	}

	// 设置cookie
	c.SetCookie("token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// CurrentUser 获取当前登录用户信息
func (h *AuthHandler) CurrentUser(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}
	user, err := h.store.GetUserWithRoles(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// Logout 退出登录
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/auth/login")
}
