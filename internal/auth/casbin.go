package auth

import (
	"fmt"

	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

var Enforcer *casbin.Enforcer

// Casbin RBAC 模型定义 (支持多级角色与正则/KeyMatch匹配)
const rbacModelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (r.sub == "superadmin" || g(r.sub, p.sub)) && (keyMatch2(r.obj, p.obj) || regexMatch(r.obj, p.obj)) && regexMatch(r.act, p.act)
`

func InitCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to init casbin gorm adapter: %w", err)
	}

	m, err := model.NewModelFromString(rbacModelText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse casbin model: %w", err)
	}

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	// 初始化与补齐默认规则
	initDefaultPolicies(e)

	Enforcer = e
	return Enforcer, nil
}

func initDefaultPolicies(e *casbin.Enforcer) {
	// 角色策略清单
	rules := [][]string{
		// 超级管理员与租户管理员拥有 /api 下所有接口权限
		{"superadmin", "/api/.*", ".*"},
		{"admin", "/api/.*", ".*"},

		// 普通操作员权限覆盖微信全部业务、日志查看与网关监控
		{"operator", "/api/accounts.*", ".*"},
		{"operator", "/api/menu.*", ".*"},
		{"operator", "/api/materials.*", ".*"},
		{"operator", "/api/autoreply.*", ".*"},
		{"operator", "/api/fans.*", ".*"},
		{"operator", "/api/qrcode.*", ".*"},
		{"operator", "/api/miniprogram.*", ".*"},
		{"operator", "/api/notify.*", ".*"},
		{"operator", "/api/logs.*", "GET"},
		{"operator", "/api/wxgate/.*", ".*"},
		{"operator", "/api/auth/.*", ".*"},
		{"operator", "/api/open/.*", "GET"},
		{"operator", "/api/system/public-ip", "GET"},
		{"operator", "/api/system/health", "GET"},
		{"operator", "/api/system/oidc", "GET"},
	}

	for _, rule := range rules {
		if has, _ := e.HasPolicy(rule[0], rule[1], rule[2]); !has {
			_, _ = e.AddPolicy(rule[0], rule[1], rule[2])
			logger.Info("[Casbin] 动态添加权限策略: " + fmt.Sprintf("%v", rule))
		}
	}
}
