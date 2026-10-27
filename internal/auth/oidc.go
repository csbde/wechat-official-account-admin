package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

var DefaultOIDCService = &OIDCService{
	http: resty.New().SetTimeout(12 * time.Second),
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
			Email:             "admin@kkio.com", // 默认邮箱用于匹配默认管理员
			Name:              "OIDC 认证测试员",
			PreferredUsername: "oidc_user",
		}
	} else {
		// 2. 向 OIDC Token 端点换取 access_token
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

		// 3. 向 userinfo 端点拉取用户信息
		userInfoEndpoint := ""
		if disc != nil && disc.UserinfoEndpoint != "" {
			userInfoEndpoint = disc.UserinfoEndpoint
		} else {
			userInfoEndpoint = strings.TrimSuffix(cfg.Issuer, "/") + "/protocol/openid-connect/userinfo"
		}

		uResp, err := s.http.R().SetContext(ctx).
			SetHeader("Authorization", "Bearer "+tokData.AccessToken).
			Get(userInfoEndpoint)

		if err != nil {
			return nil, "", fmt.Errorf("request oidc userinfo error: %w", err)
		}

		if err := json.Unmarshal(uResp.Body(), &oidcUser); err != nil {
			return nil, "", fmt.Errorf("parse oidc userinfo error: %w", err)
		}
	}

	logger.Info("OIDC Authenticated User Profile",
		zap.String("sub", oidcUser.Sub),
		zap.String("email", oidcUser.Email),
		zap.String("name", oidcUser.Name),
	)

	// ==========================================
	// 4. 核心：根据邮箱与 OIDC Sub 进行匹配与自动绑定
	// ==========================================
	var user model.User
	found := false

	// 规则 1：优先按照 email 匹配已有用户并自动绑定
	if oidcUser.Email != "" {
		err := database.DB.Where("email = ?", oidcUser.Email).First(&user).Error
		if err == nil {
			found = true
			logger.Info("OIDC: User matched by email", zap.String("email", oidcUser.Email), zap.Uint("user_id", user.ID))
			// 绑定 OIDC 标识
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

	// 规则 3：若均未找到，自动创建新用户并关联邮箱与租户
	if !found {
		username := oidcUser.PreferredUsername
		if username == "" && oidcUser.Email != "" {
			username = strings.Split(oidcUser.Email, "@")[0]
		}
		if username == "" {
			username = "oidc_" + oidcUser.Sub
		}

		// 检查用户名是否重复
		var count int64
		database.DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
		if count > 0 {
			username = fmt.Sprintf("%s_%d", username, time.Now().Unix()%10000)
		}

		nickname := oidcUser.Name
		if nickname == "" {
			nickname = username
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte("oidc_user@2025"), bcrypt.DefaultCost)
		user = model.User{
			TenantID:   1, // 归属于默认租户
			Username:   username,
			Email:      oidcUser.Email,
			Password:   string(hash),
			Nickname:   nickname,
			Role:       "admin", // OIDC 企业登录用户默认享有租户管理员完整权限 (租户行级隔离保护)
			Status:     1,
			OIDCIssuer: cfg.Issuer,
			OIDCSub:    oidcUser.Sub,
		}
		database.DB.Create(&user)
		logger.Info("OIDC: New user created and bound", zap.String("email", user.Email), zap.Uint("user_id", user.ID))
	}

	// 下发本系统的 JWT Token
	token, err := GenerateToken(user.ID, user.TenantID, user.Username, user.Role, "oidc")
	if err != nil {
		return nil, "", err
	}

	return &user, token, nil
}
