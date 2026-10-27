package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/config"

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

// GetHomeTenantID 获取当前登录用户真实归属的租户 ID
// (与 GetTenantID 区分：超级管理员切换租户视角时 GetTenantID 会变，本函数始终返回本人所属租户)
func GetHomeTenantID(c *gin.Context) uint {
	val, exists := c.Get("home_tenant_id")
	if !exists {
		return GetTenantID(c)
	}
	if tid, ok := val.(uint); ok && tid > 0 {
		return tid
	}
	return GetTenantID(c)
}

// IsTenantViewSwitched 判断超级管理员是否处于"代管其他租户"视角
func IsTenantViewSwitched(c *gin.Context) bool {
	val, exists := c.Get("tenant_view")
	if !exists {
		return false
	}
	switched, ok := val.(bool)
	return ok && switched
}

// IsSuperAdmin 判断是否超级管理员
func IsSuperAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && role == "superadmin"
}

// GetRequestScheme 获取请求的真实协议 (支持上级反向代理 X-Forwarded-Proto)
func GetRequestScheme(c *gin.Context) string {
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(proto)
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

// GetPublicHost 获取对外公网的 Scheme 和 Host (自适应反向代理及 public_host 配置)
func GetPublicHost(c *gin.Context) (scheme, host string) {
	pubHost := strings.TrimSpace(config.GlobalConfig.Wechat.PublicHost)
	if pubHost != "" {
		if strings.HasPrefix(pubHost, "https://") {
			return "https", strings.TrimPrefix(pubHost, "https://")
		}
		if strings.HasPrefix(pubHost, "http://") {
			return "http", strings.TrimPrefix(pubHost, "http://")
		}
		return GetRequestScheme(c), pubHost
	}

	h := c.GetHeader("X-Forwarded-Host")
	if h == "" {
		h = c.Request.Host
	}
	return GetRequestScheme(c), h
}

// BuildPublicURL 构建对外统一的公网 URL (自动处理 https 与反向代理)
func BuildPublicURL(c *gin.Context, path string) string {
	scheme, host := GetPublicHost(c)
	path = "/" + strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s://%s%s", scheme, strings.TrimRight(host, "/"), path)
}
