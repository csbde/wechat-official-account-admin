package handler

import (
	"strconv"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
)

type LogHandler struct{}

// List 请求与追踪日志分页
// @Summary 获取 API 审计日志与链路追踪
// @Tags 系统与日志
// @Security BearerAuth
// @Produce json
// @Router /api/logs [get]
func (h *LogHandler) List(c *gin.Context) {
	page, pageSize := GetPaginationParams(c)
	keyword := c.Query("keyword")
	statusStr := c.Query("status")
	appid := c.Query("appid")

	query := database.DB.Model(&model.RequestLog{})
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
