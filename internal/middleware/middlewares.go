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
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-App-ID")
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

// RequestAudit 审计中间件：记录请求并写入 RequestLog，支持 OpenTelemetry TraceID 关联
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

		// 获取 trace ID
		span := trace.SpanFromContext(c.Request.Context())
		traceID := ""
		if span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}

		appid, _ := c.Get("appid")
		appidStr, _ := appid.(string)

		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") && !strings.Contains(path, "/request-log") {
			respBody := ""
			if w.body.Len() < 2048 {
				respBody = w.body.String()
			}
			go func() {
				reqLog := model.RequestLog{
					AppID:        appidStr,
					TraceID:      traceID,
					Method:       c.Request.Method,
					Path:         path,
					StatusCode:   c.Writer.Status(),
					ClientIP:     c.ClientIP(),
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

// JWTAuth 鉴权中间件
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

		c.Set("user_id", claims.UserID)
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

// TenantAppID 公众号租户中间件：自动从 Header `X-App-ID` 或 Query `appid` 提取
func TenantAppID(required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		appid := c.GetHeader("X-App-ID")
		if appid == "" {
			appid = c.Query("appid")
		}

		if appid == "" && required {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Header X-App-ID or query appid is required for this operation"})
			c.Abort()
			return
		}

		c.Set("appid", appid)
		c.Next()
	}
}
