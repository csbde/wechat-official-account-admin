package router

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/handler"
	"github.com/anchel/wechat-official-account-admin/internal/middleware"
	"github.com/anchel/wechat-official-account-admin/internal/wxgate"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func SetupRouter(frontendFS *embed.FS) *gin.Engine {
	cfg := &config.GlobalConfig
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// 设置信任的上级反向代理列表，确保正确提取真实 Client IP 与反代头
	if len(cfg.Server.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.Server.TrustedProxies)
	}

	// 基础中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(otelgin.Middleware(cfg.Telemetry.ServiceName))
	r.Use(middleware.RequestAudit())

	// 媒体文件本地磁盘服务
	r.Static("/files/upload-image", "./files/upload-image")
	r.Static("/files/wx-download-media", "./files/wx-download-media")

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ==========================================
	// 1. 微信普通公众号回调网关 (微信直连，无需 JWT)
	// ==========================================
	gatewayHandler := &handler.GatewayHandler{}
	wxmp := r.Group("/wxmp")
	{
		wxmp.GET("/:appid/handler", gatewayHandler.Verify)
		wxmp.POST("/:appid/handler", gatewayHandler.ReceiveMessage)
	}

	// ==========================================
	// 2. 微信开放平台-第三方平台网关 (微信直连，无需 JWT)
	// ==========================================
	openHandler := &handler.OpenComponentHandler{}
	openGate := r.Group("/open")
	{
		// 授权事件接收 URL
		openGate.POST("/component/notify", openHandler.ReceiveAuthNotify)
		// 代公众号接收消息与事件 URL
		openGate.POST("/component/:appid/callback", openHandler.ReceiveAppMsgCallback)
	}

	// ==========================================
	// 3. WXGate 核心网关功能 (高并发凭证中心、突破单域名限制中转、JSSDK 签名)
	// ==========================================
	gateProxy := wxgate.NewProxyHandler()
	gateOAuth := wxgate.NewOAuthHandler()
	gateJSSDK := wxgate.NewJSSDKHandler()

	// 3.1 微信 OAuth2 网页授权中转代理 (突破单一授权回调域名限制，自动回传脱敏用户信息)
	connectGroup := r.Group("/connect/oauth2")
	{
		connectGroup.GET("/authorize", gateOAuth.HandleAuthorize)
		connectGroup.POST("/authorize", gateOAuth.HandleAuthorize)
		connectGroup.GET("/callback/:session_id", gateOAuth.HandleCallback)
		connectGroup.GET("/callback", gateOAuth.HandleCallback)
	}
	compatAuth := r.Group("/wechat/auth")
	{
		compatAuth.GET("/authorize", gateOAuth.HandleAuthorize)
		compatAuth.POST("/authorize", gateOAuth.HandleAuthorize)
		compatAuth.GET("/callback/:session_id", gateOAuth.HandleCallback)
		compatAuth.GET("/callback", gateOAuth.HandleCallback)
	}

	// 3.2 JSSDK 网页签名配置接口与静态脚本代理
	r.GET("/api/jssdk/config", gateJSSDK.HandleGetConfig)
	r.POST("/api/jssdk/config", gateJSSDK.HandleGetConfig)
	r.GET("/jssdk/config", gateJSSDK.HandleGetConfig)
	r.POST("/jssdk/config", gateJSSDK.HandleGetConfig)
	r.GET("/jweixin.js", gateJSSDK.HandleJSSDKScript)
	r.GET("/jweixin-1.6.0.js", gateJSSDK.HandleJSSDKScript)

	// 3.3 微信全局 Token 与 Ticket 原生接口代理 (两级缓存 + 防穿透单飞合并，彻底告别Token冲刷互踢)
	cgiGroup := r.Group("/cgi-bin")
	{
		cgiGroup.GET("/token", gateProxy.HandleTokenRequest)
		cgiGroup.POST("/token", gateProxy.HandleTokenRequest)
		cgiGroup.POST("/stable_token", gateProxy.HandleStableTokenRequest)
		cgiGroup.GET("/ticket/getticket", gateJSSDK.HandleGetTicket)
		cgiGroup.POST("/ticket/getticket", gateJSSDK.HandleGetTicket)
	}

	// 3.4 微信 sns 接口透明转发
	snsGroup := r.Group("/sns")
	{
		snsGroup.Any("/*any", gateProxy.HandleGenericProxy)
	}

	// ==========================================
	// 4. 业务 RESTful API 分组
	// ==========================================
	api := r.Group("/api")

	authHandler := &handler.AuthHandler{}
	systemHandler := &handler.SystemHandler{}
	qrcodeHandler := &handler.QrcodeHandler{}
	miniHandler := &handler.MiniProgramHandler{}

	// 公开接口 (无需认证)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/sso/callback", authHandler.SSOCallback)
		authGroup.GET("/oidc/login", authHandler.OIDCLogin)
		authGroup.GET("/oidc/callback", authHandler.OIDCCallback)
	}
	api.GET("/system/public-ip", systemHandler.PublicIP)
	api.GET("/system/health", systemHandler.HealthCheck)
	api.GET("/qrcode/:id/image", qrcodeHandler.GetImage)
	api.POST("/miniprogram/code2session", miniHandler.Code2Session)

	// 微信扫码授权重定向回调端点 (由微信浏览器重定向直连访问，无需 JWT Header)
	api.GET("/open/auth-callback", openHandler.AuthCallback)

	// WXGate 网关监控与缓存管理
	api.GET("/wxgate/stats", gateProxy.GetGateStats)
	api.DELETE("/wxgate/cache/:app_id", gateProxy.ClearAppCache)

	// 受保护的 API 接口 (需 JWT + Casbin 鉴权，支持多租户隔离)
	protected := api.Group("")
	protected.Use(middleware.JWTAuth())
	protected.Use(middleware.CasbinEnforcer())
	{
		// 认证相关
		protected.GET("/auth/userinfo", authHandler.UserInfo)
		protected.POST("/auth/logout", authHandler.Logout)

		// 微信开放平台第三方平台配置与授权
		protected.GET("/open/component", openHandler.GetComponent)
		protected.POST("/open/component", openHandler.SaveComponent)
		protected.GET("/open/auth-url", openHandler.GetAuthURL)
		protected.POST("/open/request-ticket", openHandler.RequestPushTicket)
		protected.POST("/open/manual-ticket", openHandler.ManualSetTicket)

		// 公众号管理 (租户隔离)
		accountHandler := &handler.AccountHandler{}
		protected.GET("/accounts", accountHandler.List)
		protected.POST("/accounts", accountHandler.Create)
		protected.PUT("/accounts/:id", accountHandler.Update)
		protected.DELETE("/accounts/:id", accountHandler.Delete)
		protected.POST("/accounts/:id/set-default", accountHandler.SetDefault)
		protected.GET("/accounts/:id/authorizer-info", accountHandler.GetAuthorizerDetail)
		// 迁移公众号归属租户 (仅超级管理员；事务内连同该 AppID 的业务数据一起搬家)
		protected.POST("/accounts/:id/transfer-tenant", accountHandler.TransferTenant)

		// 系统用户与日志管理 (租户隔离)
		logHandler := &handler.LogHandler{}
		protected.GET("/logs", logHandler.List)
		// 租户与组织管理 (仅超级管理员可增删改，普通用户只读本租户)
		tenantHandler := &handler.TenantHandler{}
		protected.GET("/system/tenants", tenantHandler.List)
		protected.GET("/system/tenants/options", tenantHandler.Options)
		protected.POST("/system/tenants", tenantHandler.Create)
		protected.PUT("/system/tenants/:id", tenantHandler.Update)
		protected.DELETE("/system/tenants/:id", tenantHandler.Delete)

		protected.GET("/system/users", systemHandler.UserList)
		protected.POST("/system/users", systemHandler.CreateUser)
		protected.PUT("/system/users/:id", systemHandler.UpdateUser)
		protected.DELETE("/system/users/:id", systemHandler.DeleteUser)
		protected.GET("/system/oidc", systemHandler.GetOIDCSetting)
		protected.POST("/system/oidc", systemHandler.SaveOIDCSetting)
		protected.POST("/system/oidc/test", systemHandler.TestOIDCDiscovery)

		// 微信小程序 (租户/平台级通用能力: 快速注册任务追踪与代码模板库/草稿箱)
		protected.GET("/miniprogram/tasks", miniHandler.ListTasks)
		protected.POST("/miniprogram/tasks", miniHandler.CreateTask)
		protected.POST("/miniprogram/tasks/:id/refresh", miniHandler.RefreshTask)
		protected.GET("/miniprogram/drafts", miniHandler.GetTemplateDrafts)
		protected.GET("/miniprogram/templates", miniHandler.GetTemplates)
		protected.POST("/miniprogram/templates/add-from-draft", miniHandler.AddTemplateFromDraft)
		protected.DELETE("/miniprogram/templates/:id", miniHandler.DeleteTemplate)
		protected.GET("/miniprogram/apps", miniHandler.ListMiniApps)

		// 微信业务相关接口 (必须携带 AppID，且经由 TenantAppID 严格防越权校验)
		tenant := protected.Group("")
		tenant.Use(middleware.TenantAppID(true))
		{
			// 自定义菜单
			menuHandler := &handler.MenuHandler{}
			tenant.GET("/menu", menuHandler.Get)
			tenant.POST("/menu", menuHandler.Save)
			tenant.POST("/menu/publish", menuHandler.Publish)
			tenant.POST("/menu/sync", menuHandler.SyncFromWx)
			tenant.DELETE("/menu", menuHandler.Delete)
			tenant.GET("/menu/conditional", menuHandler.GetConditionals)
			tenant.POST("/menu/conditional", menuHandler.SaveConditional)
			tenant.DELETE("/menu/conditional/:id", menuHandler.DeleteConditional)

			// 素材管理
			materialHandler := &handler.MaterialHandler{}
			tenant.GET("/materials", materialHandler.List)
			tenant.POST("/materials/upload", materialHandler.Upload)
			tenant.POST("/materials/news", materialHandler.CreateNews)
			tenant.DELETE("/materials/:id", materialHandler.Delete)
			tenant.GET("/materials/:id/preview", materialHandler.Preview)

			// 自动回复与 AI 智能助手接入 (兼容 ChatGPT 规范)
			replyHandler := &handler.ReplyHandler{}
			tenant.GET("/autoreply", replyHandler.List)
			tenant.POST("/autoreply", replyHandler.Save)
			tenant.DELETE("/autoreply/:id", replyHandler.Delete)
			tenant.PUT("/autoreply/:id/toggle", replyHandler.Toggle)
			tenant.GET("/autoreply/ai", replyHandler.GetAISetting)
			tenant.POST("/autoreply/ai", replyHandler.SaveAISetting)
			tenant.POST("/autoreply/ai/test", replyHandler.TestAIConnection)

			// 粉丝与标签
			fanHandler := &handler.FanHandler{}
			tenant.GET("/fans", fanHandler.List)
			tenant.POST("/fans/sync", fanHandler.Sync)
			tenant.GET("/fans/sync-status", fanHandler.GetSyncStatus)
			tenant.POST("/fans/remark", fanHandler.UpdateRemark)
			tenant.GET("/fans/tags", fanHandler.GetTags)
			tenant.POST("/fans/tags", fanHandler.CreateTag)
			tenant.DELETE("/fans/tags/:id", fanHandler.DeleteTag)
			tenant.POST("/fans/tags/batch", fanHandler.BatchTagging)

			// 带参二维码
			qrcodeHandler := &handler.QrcodeHandler{}
			tenant.GET("/qrcode", qrcodeHandler.List)
			tenant.POST("/qrcode", qrcodeHandler.Create)

			// 微信小程序代开发 (指定账号级别接口: 合法域名与公众号关联)
			tenant.GET("/miniprogram/domain", miniHandler.GetDomain)
			tenant.POST("/miniprogram/domain", miniHandler.SetDomain)
			tenant.GET("/miniprogram/linked", miniHandler.GetLinkedMiniPrograms)
			tenant.POST("/miniprogram/link", miniHandler.LinkMiniProgram)
			tenant.POST("/miniprogram/unlink", miniHandler.UnlinkMiniProgram)

			// 小程序代码管理与发布 (提交代码 / 体验版 / 提审 / 发布 / 灰度 / 类目 / 体验者)
			tenant.POST("/miniprogram/code/commit", miniHandler.CommitCode)
			tenant.GET("/miniprogram/code/pages", miniHandler.GetCodePages)
			tenant.GET("/miniprogram/code/trial-qrcode", miniHandler.GetTrialQrcode)
			tenant.POST("/miniprogram/code/submit-audit", miniHandler.SubmitAudit)
			tenant.GET("/miniprogram/code/audit-status", miniHandler.GetAuditStatus)
			tenant.GET("/miniprogram/code/latest-audit", miniHandler.GetLatestAudit)
			tenant.GET("/miniprogram/code/privacy-info", miniHandler.GetCodePrivacyInfo)
			tenant.POST("/miniprogram/code/undo-audit", miniHandler.UndoAudit)
			tenant.POST("/miniprogram/code/release", miniHandler.ReleaseCode)
			tenant.GET("/miniprogram/code/visit-status", miniHandler.GetVisitStatus)
			tenant.POST("/miniprogram/code/visit-status", miniHandler.SetVisitStatus)
			tenant.GET("/miniprogram/code/versions", miniHandler.GetHistoryVersions)
			tenant.POST("/miniprogram/code/revert", miniHandler.RevertRelease)
			tenant.GET("/miniprogram/code/support-version", miniHandler.GetSupportVersion)
			tenant.POST("/miniprogram/code/support-version", miniHandler.SetSupportVersion)

			// 小程序服务类目与体验者成员管理
			tenant.GET("/miniprogram/categories", miniHandler.GetCategories)
			tenant.POST("/miniprogram/categories", miniHandler.AddCategory)
			tenant.POST("/miniprogram/categories/delete", miniHandler.DeleteCategory)
			tenant.GET("/miniprogram/testers", miniHandler.GetTesters)
			tenant.POST("/miniprogram/testers/bind", miniHandler.BindTester)
			tenant.POST("/miniprogram/testers/unbind", miniHandler.UnbindTester)

			// 消息模板体系: 公众号模板消息 (行业 / 模板库 / 发送 / 一次性订阅 / 拦截查询)
			notifyHandler := &handler.NotifyHandler{}
			tenant.GET("/notify/oa/industry", notifyHandler.GetIndustry)
			tenant.POST("/notify/oa/industry", notifyHandler.SetIndustry)
			tenant.GET("/notify/oa/templates", notifyHandler.ListTemplates)
			tenant.POST("/notify/oa/templates", notifyHandler.AddTemplate)
			tenant.POST("/notify/oa/templates/delete", notifyHandler.DeleteTemplate)
			tenant.POST("/notify/oa/send", notifyHandler.SendTemplateMsg)
			tenant.POST("/notify/oa/one-time", notifyHandler.SendOneTimeSubscribe)
			tenant.POST("/notify/oa/blocked", notifyHandler.QueryBlocked)

			// 消息模板体系: 订阅消息 (类目 / 公共模板库 / 我的模板库 / 发送)
			tenant.GET("/notify/sub/categories", notifyHandler.SubCategories)
			tenant.GET("/notify/sub/pub-titles", notifyHandler.SubPubTitles)
			tenant.GET("/notify/sub/pub-keywords", notifyHandler.SubPubKeywords)
			tenant.GET("/notify/sub/templates", notifyHandler.SubMyTemplates)
			tenant.POST("/notify/sub/templates", notifyHandler.SubAddTemplate)
			tenant.POST("/notify/sub/templates/delete", notifyHandler.SubDeleteTemplate)
			tenant.POST("/notify/sub/send", notifyHandler.SubSend)
			tenant.POST("/notify/sub/bizsend", notifyHandler.SubBizSend)

			// 公众号群发消息 (按 openid / 按标签或全体 / 预览 / 状态查询 / 删除 / 速度)
			massHandler := &handler.MassHandler{}
			tenant.POST("/notify/mass/send", massHandler.Send)
			tenant.POST("/notify/mass/sendall", massHandler.SendAll)
			tenant.POST("/notify/mass/preview", massHandler.Preview)
			tenant.POST("/notify/mass/status", massHandler.Status)
			tenant.POST("/notify/mass/delete", massHandler.Delete)
			tenant.GET("/notify/mass/speed", massHandler.GetSpeed)
			tenant.POST("/notify/mass/speed", massHandler.SetSpeed)
			tenant.POST("/notify/mass/uploadimg", massHandler.UploadNewsImage)
			tenant.POST("/notify/mass/uploadnews", massHandler.UploadNews)
			tenant.POST("/notify/mass/uploadthumb", massHandler.UploadThumb)

			// 服务提醒卡片 (订阅消息·服务卡片，小程序/服务号)
			tenant.POST("/notify/sub/service-card", massHandler.SetServiceCard)
			tenant.POST("/notify/sub/service-card/query", massHandler.GetServiceCard)
		}
	}

	// 嵌入式前端 SPA
	if frontendFS != nil {
		distSub, err := fs.Sub(frontendFS, "wechat-official-account-admin-fe/dist")
		if err == nil {
			indexHTML, _ := fs.ReadFile(distSub, "index.html")
			fileServer := http.FileServer(http.FS(distSub))

			r.NoRoute(func(c *gin.Context) {
				path := strings.TrimPrefix(c.Request.URL.Path, "/")

				if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, "wxmp/") || strings.HasPrefix(path, "open/") || strings.HasPrefix(path, "connect/") || strings.HasPrefix(path, "cgi-bin/") || strings.HasPrefix(path, "swagger/") {
					c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API not found"})
					return
				}

				if path != "" {
					if file, err := distSub.Open(path); err == nil {
						_ = file.Close()
						ext := filepath.Ext(path)
						if ctype := mime.TypeByExtension(ext); ctype != "" {
							c.Header("Content-Type", ctype)
						}
						fileServer.ServeHTTP(c.Writer, c.Request)
						return
					}
				}

				if len(indexHTML) > 0 {
					c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
					return
				}

				c.String(http.StatusOK, "WeChat Official Account Admin Ready")
			})
		}
	}

	return r
}
