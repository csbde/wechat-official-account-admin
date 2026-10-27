package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/auth"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-App-ID, X-Tenant-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// RequestAudit 审计中间件：记录多租户请求
func RequestAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var requestBody string
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(bodyBytes) < 2048 {
				requestBody = string(bodyBytes)
			}
		}

		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		latency := time.Since(start).Milliseconds()

		span := trace.SpanFromContext(c.Request.Context())
		traceID := ""
		if span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}

		appid, _ := c.Get("appid")
		appidStr, _ := appid.(string)

		tenantIDVal, _ := c.Get("tenant_id")
		tenantID, _ := tenantIDVal.(uint)
		if tenantID == 0 {
			tenantID = 1
		}

		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") && !strings.Contains(path, "/logs") {
			respBody := ""
			if w.body.Len() < 2048 {
				respBody = w.body.String()
			}
			go func() {
				reqLog := model.RequestLog{
					TenantID:     tenantID,
					AppID:        appidStr,
					TraceID:      traceID,
					Method:       c.Request.Method,
					Path:         path,
					StatusCode:   c.Writer.Status(),
					ClientIP:     GetRealClientIP(c),
					LatencyMs:    latency,
					RequestBody:  requestBody,
					ResponseBody: respBody,
					CreatedAt:    time.Now(),
				}
				if database.DB != nil {
					_ = database.DB.Create(&reqLog).Error
				}
			}()
		}
	}
}

// JWTAuth 鉴权中间件 (提取用户身份与多租户归属)
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Missing authorization token"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		tokenString := ""
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			tokenString = parts[1]
		} else {
			tokenString = parts[0]
		}

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 将用户基础信息与所属租户放入上下文
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// CasbinEnforcer RBAC 权限校验中间件
func CasbinEnforcer() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "Forbidden: no role detected"})
			c.Abort()
			return
		}

		sub := role.(string)
		obj := c.Request.URL.Path
		act := c.Request.Method

		if auth.Enforcer != nil {
			allowed, err := auth.Enforcer.Enforce(sub, obj, act)
			if err != nil || !allowed {
				logger.Warn("Casbin access denied", zap.String("sub", sub), zap.String("obj", obj), zap.String("act", act))
				c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "Forbidden: insufficient permissions"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// TenantAppID 多租户公众号防越权中间件
// 严格校验请求指定的公众号是否真正属于当前登录用户的租户，彻底杜绝跨租户水平越权！
func TenantAppID(required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		appid := c.GetHeader("X-App-ID")
		if appid == "" {
			appid = c.Query("appid")
		}

		if appid == "" {
			if required {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Header X-App-ID or query appid is required"})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// 提取当前用户的租户 ID
		tenantIDVal, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "No tenant context found"})
			c.Abort()
			return
		}
		tenantID := tenantIDVal.(uint)

		roleVal, _ := c.Get("role")
		role, _ := roleVal.(string)

		// 超级管理员放行；非超级管理员强制核查该 AppID 是否属于该租户
		if role != "superadmin" {
			var count int64
			database.DB.Model(&model.WxAccount{}).
				Where("app_id = ? AND tenant_id = ?", appid, tenantID).
				Count(&count)

			if count == 0 {
				logger.Warn("Cross-tenant access attempted!",
					zap.Uint("tenant_id", tenantID),
					zap.String("appid", appid),
					zap.String("client_ip", c.ClientIP()),
				)
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "越权访问被拒绝：指定微信公众号不属于您所在的租户组织",
				})
				c.Abort()
				return
			}
		}

		c.Set("appid", appid)
		c.Next()
	}
}

// GetRealClientIP 提取经过反向代理后的真实客户端 IP
func GetRealClientIP(c *gin.Context) string {
	if cfIP := c.GetHeader("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}
	if fwd := c.GetHeader("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}
	return c.ClientIP()
}
