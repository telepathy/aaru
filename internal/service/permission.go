package service

import (
	"aaru/internal/model"
	"aaru/internal/store"
)

type PermissionService struct {
	store *store.DBStore
}

func NewPermissionService(s *store.DBStore) *PermissionService {
	return &PermissionService{store: s}
}

// Can 检查用户是否对指定部署单元有指定操作权限
func (p *PermissionService) Can(userID uint, deployUnitCode string, action string) bool {
	user, err := p.store.GetUserWithRoles(userID)
	if err != nil || user == nil {
		return false
	}
	for _, role := range user.Roles {
		var permissions []model.Permission
		if err := p.store.DB().Model(&role).Association("Permissions").Find(&permissions); err != nil {
			continue
		}
		for _, perm := range permissions {
			if perm.Action == action && (perm.DeployUnitCode == "" || perm.DeployUnitCode == "*" || perm.DeployUnitCode == deployUnitCode) {
				return true
			}
		}
	}
	return false
}

// CanAction 更通用的权限检查
func (p *PermissionService) CanAction(userID uint, action string) bool {
	user, err := p.store.GetUserWithRoles(userID)
	if err != nil || user == nil {
		return false
	}
	for _, role := range user.Roles {
		var permissions []model.Permission
		if err := p.store.DB().Model(&role).Association("Permissions").Find(&permissions); err != nil {
			continue
		}
		for _, perm := range permissions {
			if perm.Action == action {
				return true
			}
		}
	}
	return false
}

// GetUserPermittedDUs 获取用户有权限的部署单元列表
func (p *PermissionService) GetUserPermittedDUs(userID uint) (map[string]bool, error) {
	user, err := p.store.GetUserWithRoles(userID)
	if err != nil || user == nil {
		return nil, err
	}
	result := make(map[string]bool)
	for _, role := range user.Roles {
		var permissions []model.Permission
		if err := p.store.DB().Model(&role).Association("Permissions").Find(&permissions); err != nil {
			continue
		}
		for _, perm := range permissions {
			if perm.DeployUnitCode == "*" || perm.DeployUnitCode == "" {
				result["*"] = true
			} else {
				result[perm.DeployUnitCode] = true
			}
		}
	}
	return result, nil
}
