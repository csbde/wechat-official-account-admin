package auth

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type OIDCUserInfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
}

type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

type OIDCService struct {
	http *resty.Client
}

// 创建针对 OIDC 高稳定性定制的 HTTP 客户端 (关闭 HTTP/2，彻底规避对端 GOAWAY ENHANCE_YOUR_CALM 限频)
func createOIDCClient() *resty.Client {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: false},
		ForceAttemptHTTP2: false,
		// 关键：清空 TLSNextProto，强制禁用 HTTP/2 ALPN 协商，采用极度稳定的 HTTP/1.1 协议
		TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		MaxIdleConns:    100,
		IdleConnTimeout: 30 * time.Second,
	}

	return resty.New().
		SetTransport(transport).
		SetTimeout(10 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(250 * time.Millisecond).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
}

var DefaultOIDCService = &OIDCService{
	http: createOIDCClient(),
}

// parseIDTokenPayload 从 id_token JWT 中直接解析用户信息 payload (免去向 userinfo 发起网络请求，极速且 100% 免疫网络限频)
func parseIDTokenPayload(idToken string) (*OIDCUserInfo, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid jwt format")
	}

	payloadSegment := parts[1]
	// 补齐 Base64 padding
	if l := len(payloadSegment) % 4; l > 0 {
		payloadSegment += strings.Repeat("=", 4-l)
	}

	decoded, err := base64.URLEncoding.DecodeString(payloadSegment)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(payloadSegment)
		if err != nil {
			return nil, err
		}
	}

	var user OIDCUserInfo
	if err := json.Unmarshal(decoded, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetDiscovery 获取 OIDC 服务商的端点发现
func (s *OIDCService) GetDiscovery(ctx context.Context, issuer string) (*OIDCDiscovery, error) {
	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := s.http.R().SetContext(ctx).Get(wellKnown)
	if err != nil {
		return nil, err
	}
	var disc OIDCDiscovery
	if err := json.Unmarshal(resp.Body(), &disc); err != nil {
		return nil, err
	}
	return &disc, nil
}

// BuildAuthURL 构建前往 OIDC 身份认证服务的跳转地址
func (s *OIDCService) BuildAuthURL(ctx context.Context, host string) (string, string, error) {
	cfg := config.GlobalConfig.OIDC
	redirectURI := cfg.RedirectURI
	if redirectURI == "" {
		scheme := "http"
		redirectURI = fmt.Sprintf("%s://%s/api/auth/oidc/callback", scheme, host)
	}

	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// 如果没有配置外部真实 issuer，返回通用模拟快捷授权
	if cfg.Issuer == "" {
		simURL := fmt.Sprintf("/api/auth/oidc/callback?code=mock_oidc_code&state=%s", state)
		return simURL, state, nil
	}

	disc, err := s.GetDiscovery(ctx, cfg.Issuer)
	authEndpoint := ""
	if err == nil && disc.AuthorizationEndpoint != "" {
		authEndpoint = disc.AuthorizationEndpoint
	} else {
		authEndpoint = strings.TrimSuffix(cfg.Issuer, "/") + "/protocol/openid-connect/auth"
	}

	scope := strings.Join(cfg.Scopes, " ")
	if scope == "" {
		scope = "openid profile email"
	}

	u := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		authEndpoint,
		url.QueryEscape(cfg.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scope),
		url.QueryEscape(state),
	)

	return u, state, nil
}

// HandleCallback 处理 OIDC 回调 code，完成邮箱自动匹配与用户绑定
func (s *OIDCService) HandleCallback(ctx context.Context, code, host string) (*model.User, string, error) {
	cfg := config.GlobalConfig.OIDC

	var oidcUser OIDCUserInfo

	// 1. 若为模拟或者未配置外部 issuer
	if cfg.Issuer == "" || code == "mock_oidc_code" {
		oidcUser = OIDCUserInfo{
			Sub:               "oidc_sub_demo_1001",
			Email:             "admin@kkio.com",
			Name:              "OIDC 认证测试员",
			PreferredUsername: "oidc_user",
		}
	} else {
		// 2. 向 OIDC Token 端点换取 id_token 与 access_token
		disc, _ := s.GetDiscovery(ctx, cfg.Issuer)
		tokenEndpoint := ""
		if disc != nil && disc.TokenEndpoint != "" {
			tokenEndpoint = disc.TokenEndpoint
		} else {
			tokenEndpoint = strings.TrimSuffix(cfg.Issuer, "/") + "/protocol/openid-connect/token"
		}

		redirectURI := cfg.RedirectURI
		if redirectURI == "" {
			redirectURI = fmt.Sprintf("http://%s/api/auth/oidc/callback", host)
		}

		tokenResp, err := s.http.R().SetContext(ctx).SetFormData(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     cfg.ClientID,
			"client_secret": cfg.ClientSecret,
			"redirect_uri":  redirectURI,
		}).Post(tokenEndpoint)

		if err != nil {
			return nil, "", fmt.Errorf("request oidc token error: %w", err)
		}

		var tokData struct {
			AccessToken string `json:"access_token"`
			IDToken     string `json:"id_token"`
		}
		if err := json.Unmarshal(tokenResp.Body(), &tokData); err != nil {
			return nil, "", fmt.Errorf("parse oidc token error: %w", err)
		}

		// 3. 【核心优化 1】优先从 id_token JWT 中直接解析用户信息 (纯内存解析，零网络开销，100% 规避对端限频)
		if tokData.IDToken != "" {
			if parsedUser, parseErr := parseIDTokenPayload(tokData.IDToken); parseErr == nil && parsedUser != nil {
				oidcUser = *parsedUser
				logger.Info("[OIDC] 成功从 id_token 本地提取用户信息", zap.String("sub", oidcUser.Sub), zap.String("email", oidcUser.Email))
			}
		}

		// 4. 【核心优化 2】仅当 id_token 中未获取到邮箱时，才尝试请求 userinfo 端点进行补充
		if oidcUser.Email == "" {
			userInfoEndpoint := ""
			if disc != nil && disc.UserinfoEndpoint != "" {
				userInfoEndpoint = disc.UserinfoEndpoint
			} else {
				userInfoEndpoint = strings.TrimSuffix(cfg.Issuer, "/") + "/protocol/openid-connect/userinfo"
			}

			uResp, err := s.http.R().SetContext(ctx).
				SetHeader("Authorization", "Bearer "+tokData.AccessToken).
				Get(userInfoEndpoint)

			if err == nil && uResp.StatusCode() == 200 {
				var infoFromEndpoint OIDCUserInfo
				if err := json.Unmarshal(uResp.Body(), &infoFromEndpoint); err == nil {
					if infoFromEndpoint.Email != "" {
						oidcUser.Email = infoFromEndpoint.Email
					}
					if infoFromEndpoint.Sub != "" && oidcUser.Sub == "" {
						oidcUser.Sub = infoFromEndpoint.Sub
					}
					if infoFromEndpoint.Name != "" && oidcUser.Name == "" {
						oidcUser.Name = infoFromEndpoint.Name
					}
				}
			} else if err != nil {
				logger.Warn("[OIDC] 请求 userinfo 失败，但继续尝试通过已有凭据登录", zap.Error(err))
				// 若已有 sub 则不抛出致命错误中断，继续放行登录！
				if oidcUser.Sub == "" {
					return nil, "", fmt.Errorf("request oidc userinfo error: %w", err)
				}
			}
		}
	}

	logger.Info("OIDC Authenticated User Profile",
		zap.String("sub", oidcUser.Sub),
		zap.String("email", oidcUser.Email),
		zap.String("name", oidcUser.Name),
	)

	// ==========================================
	// 5. 核心：根据邮箱与 OIDC Sub 进行匹配与自动绑定
	// ==========================================
	var user model.User
	found := false

	// 规则 1：优先按照 email 匹配已有用户并自动绑定
	if oidcUser.Email != "" {
		err := database.DB.Where("email = ?", oidcUser.Email).First(&user).Error
		if err == nil {
			found = true
			logger.Info("OIDC: User matched by email", zap.String("email", oidcUser.Email), zap.Uint("user_id", user.ID))
			user.OIDCIssuer = cfg.Issuer
			user.OIDCSub = oidcUser.Sub
			if user.Nickname == "" && oidcUser.Name != "" {
				user.Nickname = oidcUser.Name
			}
			database.DB.Save(&user)
		}
	}

	// 规则 2：若邮箱未匹配，尝试根据 OIDC Sub 查找
	if !found && oidcUser.Sub != "" {
		err := database.DB.Where("oidc_sub = ?", oidcUser.Sub).First(&user).Error
		if err == nil {
			found = true
			logger.Info("OIDC: User matched by sub", zap.String("sub", oidcUser.Sub), zap.Uint("user_id", user.ID))
		}
	}

	// 规则 3：若均未匹配到已有用户
	if !found {
		// 查询数据库中是否开启了“允许自动创建新账号”开关
		autoRegister := cfg.AutoRegister
		defaultRole := cfg.DefaultRole
		if defaultRole == "" {
			defaultRole = "operator"
		}

		var dbSetting model.SystemOIDCSetting
		if err := database.DB.Where("enabled = ?", true).First(&dbSetting).Error; err == nil {
			autoRegister = dbSetting.AutoRegister
			if dbSetting.DefaultRole != "" {
				defaultRole = dbSetting.DefaultRole
			}
		}

		// 安全保护：若未开启自动注册，拒绝未经授权的新用户登录
		if !autoRegister {
			userIdentifier := oidcUser.Email
			if userIdentifier == "" {
				userIdentifier = oidcUser.Sub
			}
			return nil, "", fmt.Errorf("您的认证身份 [%s] 尚未在系统中开通账号，系统未开启自动建号，请联系管理员在用户管理中为您添加并绑定该邮箱", userIdentifier)
		}

		username := oidcUser.PreferredUsername
		if username == "" && oidcUser.Email != "" {
			username = strings.Split(oidcUser.Email, "@")[0]
		}
		if username == "" {
			username = "oidc_" + oidcUser.Sub
		}

		var count int64
		database.DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
		if count > 0 {
			username = fmt.Sprintf("%s_%d", username, time.Now().Unix()%10000)
		}

		nickname := oidcUser.Name
		if nickname == "" {
			nickname = username
		}

		// 确保不能越权成为 superadmin
		if defaultRole == "superadmin" || defaultRole == "admin" {
			defaultRole = "operator"
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte("oidc_user@2025"), bcrypt.DefaultCost)
		user = model.User{
			TenantID:   1,
			Username:   username,
			Email:      oidcUser.Email,
			Password:   string(hash),
			Nickname:   nickname,
			Role:       defaultRole, // 严格按照设定的权限创建 (operator 或 viewer)
			Status:     1,
			OIDCIssuer: cfg.Issuer,
			OIDCSub:    oidcUser.Sub,
		}
		database.DB.Create(&user)
		logger.Info("OIDC: New user created and bound with role", zap.String("email", user.Email), zap.String("role", defaultRole), zap.Uint("user_id", user.ID))
	}

	// 下发本系统的 JWT Token
	token, err := GenerateToken(user.ID, user.TenantID, user.Username, user.Role, "oidc")
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}
