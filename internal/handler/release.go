package handler

import (
	"net/http"
	"strconv"

	"aaru/internal/service"
	"github.com/gin-gonic/gin"
)

type ReleaseHandler struct {
	releaseService *service.ReleaseService
}

func NewReleaseHandler(rs *service.ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{releaseService: rs}
}

type CreateReleaseRequest struct {
	Title          string   `json:"title" binding:"required"`
	DeployUnitCode string   `json:"deploy_unit_code" binding:"required"`
	Version        string   `json:"version" binding:"required"`
	Environments   []string `json:"environments"`
	BlueprintID    *uint    `json:"blueprint_id"`
}

func (h *ReleaseHandler) CreateRelease(c *gin.Context) {
	var req CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint("user_id")
	release, err := h.releaseService.CreateRelease(
		req.Title, req.DeployUnitCode, req.Version, userID,
		req.Environments, req.BlueprintID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, release)
}

func (h *ReleaseHandler) ListReleases(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	releases, total, err := h.releaseService.ListReleases(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"releases": releases, "total": total, "page": page})
}

func (h *ReleaseHandler) GetRelease(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	release, err := h.releaseService.GetRelease(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *ReleaseHandler) StartRelease(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")
	release, err := h.releaseService.StartRelease(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *ReleaseHandler) ApproveStage(c *gin.Context) {
	stageID, _ := strconv.ParseUint(c.Param("stageId"), 10, 64)
	var req struct{ Comment string `json:"comment"` }
	c.ShouldBindJSON(&req)
	userID := c.GetUint("user_id")
	release, err := h.releaseService.ApproveStage(uint(stageID), userID, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *ReleaseHandler) RejectStage(c *gin.Context) {
	stageID, _ := strconv.ParseUint(c.Param("stageId"), 10, 64)
	var req struct{ Comment string `json:"comment"` }
	c.ShouldBindJSON(&req)
	userID := c.GetUint("user_id")
	release, err := h.releaseService.RejectStage(uint(stageID), userID, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *ReleaseHandler) RollbackRelease(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")
	release, err := h.releaseService.RollbackRelease(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *ReleaseHandler) PromoteToNext(c *gin.Context) {
	stageID, _ := strconv.ParseUint(c.Param("stageId"), 10, 64)
	userID := c.GetUint("user_id")
	release, err := h.releaseService.PromoteToNext(uint(stageID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, release)
}

func (h *ReleaseHandler) PendingApprovals(c *gin.Context) {
	userID := c.GetUint("user_id")
	stages, err := h.releaseService.GetPendingApprovals(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stages": stages})
}

// WebhookPromote 外部系统通过webhook触发自动晋级
func (h *ReleaseHandler) WebhookPromote(c *gin.Context) {
	token := c.Query("token")
	stageID, _ := strconv.ParseUint(c.Param("stageId"), 10, 64)
	if token == "" || stageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	release, err := h.releaseService.WebhookPromote(uint(stageID), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, release)
}
