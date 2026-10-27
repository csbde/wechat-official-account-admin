package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageResult struct {
	List       interface{} `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func Ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: msg,
	})
}

func Page(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data: PageResult{
			List: list,
			Pagination: Pagination{
				Page:     page,
				PageSize: pageSize,
				Total:    total,
			},
		},
	})
}

func GetPaginationParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}

func GetAppID(c *gin.Context) string {
	appid, _ := c.Get("appid")
	if s, ok := appid.(string); ok {
		return s
	}
	return ""
}

// GetTenantID 获取当前用户的多租户 ID
func GetTenantID(c *gin.Context) uint {
	val, exists := c.Get("tenant_id")
	if !exists {
		return 1
	}
	if tid, ok := val.(uint); ok && tid > 0 {
		return tid
	}
	return 1
}

// IsSuperAdmin 判断是否超级管理员
func IsSuperAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && role == "superadmin"
}
