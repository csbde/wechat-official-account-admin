package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/auth"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type, X-Tenant-Fallback")
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
		c.Set("home_tenant_id", claims.TenantID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		// 超级管理员可用 X-Tenant-ID 切换"租户视角"，跨租户代运营与排障；
		// 非超级管理员携带该 Header 一律忽略，且必须校验目标租户存在并已启用，防止越权。
		if claims.Role == "superadmin" {
			if raw := strings.TrimSpace(c.GetHeader("X-Tenant-ID")); raw != "" {
				tid, convErr := strconv.ParseUint(raw, 10, 64)
				if convErr != nil || tid == 0 {
					c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "X-Tenant-ID 请求头格式不合法 (需为正整数租户 ID)"})
					c.Abort()
					return
				}
				var cnt int64
				var tenantRow model.Tenant
				tenantErr := database.DB.First(&tenantRow, tid).Error
				if tenantErr == nil {
					if tenantRow.Status == 1 {
						cnt = 1
					}
				} else if errors.Is(tenantErr, gorm.ErrRecordNotFound) {
					// 租户记录缺失(历史数据或被硬删除)，但其 ID 仍被账号/成员引用：
					// 允许超级管理员切到该"幽灵租户"视角，把遗留数据找回并纠正归属。
					var refCount int64
					database.DB.Unscoped().Model(&model.WxAccount{}).Where("tenant_id = ?", tid).Count(&refCount)
					if refCount == 0 {
						database.DB.Unscoped().Model(&model.User{}).Where("tenant_id = ?", tid).Count(&refCount)
					}
					cnt = refCount
				}
				if cnt > 0 {
					c.Set("tenant_id", uint(tid))
					// 只要显式携带了有效的 X-Tenant-ID，就按该视角严格收口(即使等于本人租户)，
					// 这样超级管理员可以主动选择"只管理某个租户"；不带该头时保留平台级跨租户能力。
					c.Set("tenant_view", true)
					if uint(tid) != claims.TenantID {
						logger.Warn("Superadmin switched tenant view",
							zap.Uint("home_tenant_id", claims.TenantID),
							zap.Uint("view_tenant_id", uint(tid)),
							zap.String("username", claims.Username),
						)
					}
				} else {
					// 目标租户已被删除或停用：安全回退到超级管理员本人租户，
					// 并回写 X-Tenant-Fallback 头，便于前端自动纠正视角、避免卡在不可用视角。
					logger.Warn("Requested tenant view unavailable, fallback to home tenant",
						zap.Uint("requested_tenant_id", uint(tid)),
						zap.Uint("home_tenant_id", claims.TenantID),
						zap.String("username", claims.Username),
					)
					c.Set("tenant_id", claims.TenantID)
					c.Header("X-Tenant-Fallback", strconv.FormatUint(uint64(claims.TenantID), 10))
				}
			}
		}
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

		// 超级管理员未显式切换视角时保持平台运维语义(可访问任意租户的 AppID)；
		// 一旦显式切换视角或为普通成员，则严格按“当前租户视角”校验归属，越权直接 403。
		switched := false
		if v, ok := c.Get("tenant_view"); ok {
			switched, _ = v.(bool)
		}
		if role == "superadmin" && !switched {
			c.Set("appid", appid)
			c.Next()
			return
		}

		var count int64
		database.DB.Model(&model.WxAccount{}).
			Where("app_id = ? AND tenant_id = ?", appid, tenantID).
			Count(&count)

		if count == 0 {
			homeTenantID, _ := c.Get("home_tenant_id")
			logger.Warn("Cross-tenant access attempted!",
				zap.Uint("tenant_id", tenantID),
				zap.Any("home_tenant_id", homeTenantID),
				zap.String("username", c.GetString("username")),
				zap.String("appid", appid),
				zap.String("client_ip", c.ClientIP()),
			)
			extra := ""
			if role == "superadmin" {
				extra = "（你已切换到租户视角，请先切回该账号所属租户视角）"
			}
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "越权访问被拒绝：指定微信公众号不属于当前租户组织" + extra,
			})
			c.Abort()
			return
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
