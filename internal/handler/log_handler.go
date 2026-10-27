package handler

import (
	"strconv"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
)

type LogHandler struct{}

// List 请求与追踪日志分页 (多租户隔离防越权)
func (h *LogHandler) List(c *gin.Context) {
	tenantID := GetTenantID(c)
	page, pageSize := GetPaginationParams(c)
	keyword := c.Query("keyword")
	statusStr := c.Query("status")
	appid := c.Query("appid")

	query := database.DB.Model(&model.RequestLog{})

	// 非超级管理员只能查看本租户日志
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if appid != "" {
		query = query.Where("appid = ?", appid)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("path LIKE ? OR client_ip LIKE ? OR trace_id LIKE ?", like, like, like)
	}
	if statusStr != "" {
		st, _ := strconv.Atoi(statusStr)
		if st > 0 {
			query = query.Where("status_code = ?", st)
		}
	}

	var total int64
	query.Count(&total)

	var list []model.RequestLog
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}
