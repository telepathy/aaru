package handler

import (
	"log"
	"net/http"

	"aaru/internal/service"
	"github.com/gin-gonic/gin"
)

type DMDBHandler struct {
	dmdb *service.DMDBClient
}

func NewDMDBHandler(dmdb *service.DMDBClient) *DMDBHandler {
	return &DMDBHandler{dmdb: dmdb}
}

// ListEnvironments 获取环境列表
func (h *DMDBHandler) ListEnvironments(c *gin.Context) {
	envs, err := h.dmdb.ListEnvironments()
	if err != nil {
		log.Printf("list envs: %v", err)
		c.JSON(http.StatusOK, gin.H{"envs": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"envs": envs})
}

// ListSilos 获取竖井列表
func (h *DMDBHandler) ListSilos(c *gin.Context) {
	silos, err := h.dmdb.ListSilos()
	if err != nil {
		log.Printf("list silos: %v", err)
		c.JSON(http.StatusOK, gin.H{"silos": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"silos": silos})
}

// ListSystems 获取业务系统列表
func (h *DMDBHandler) ListSystems(c *gin.Context) {
	systems, err := h.dmdb.ListSystems()
	if err != nil {
		log.Printf("list systems: %v", err)
		c.JSON(http.StatusOK, gin.H{"systems": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"systems": systems})
}

// QueryDeployUnits 查询部署单元
func (h *DMDBHandler) QueryDeployUnits(c *gin.Context) {
	env := c.Query("env")
	system := c.Query("system")
	silo := c.Query("silo")
	if env == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "env is required"})
		return
	}
	dus, err := h.dmdb.QueryDeployUnits(env, system, silo)
	if err != nil {
		log.Printf("query dus: %v", err)
		c.JSON(http.StatusOK, gin.H{"deploy_units": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deploy_units": dus})
}

// GetDeployUnit 获取部署单元详情
func (h *DMDBHandler) GetDeployUnit(c *gin.Context) {
	code := c.Param("code")
	env := c.Query("env")
	if env == "" {
		// 尝试从所有环境查找
		du, err := h.dmdb.GetDeployUnitByCode("", code)
		if err != nil {
			envs, _ := h.dmdb.ListEnvironments()
			for _, e := range envs {
				du, err = h.dmdb.GetDeployUnitByCode(e.Env, code)
				if err == nil && du != nil && du.BizSerial != "" {
					c.JSON(http.StatusOK, du)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "deploy unit not found"})
			return
		}
		c.JSON(http.StatusOK, du)
		return
	}
	du, err := h.dmdb.GetDeployUnitByCode(env, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deploy unit not found"})
		return
	}
	c.JSON(http.StatusOK, du)
}
