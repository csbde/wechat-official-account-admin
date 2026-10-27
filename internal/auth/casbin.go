package auth

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

var Enforcer *casbin.Enforcer

// Casbin RBAC 模型定义 (内置模型)
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
m = g(r.sub, p.sub) && (keyMatch2(r.obj, p.obj) || regexMatch(r.obj, p.obj)) && regexMatch(r.act, p.act)
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

	// 初始化默认规则
	initDefaultPolicies(e)

	Enforcer = e
	return Enforcer, nil
}

func initDefaultPolicies(e *casbin.Enforcer) {
	rules := [][]string{
		{"superadmin", "/api/.*", ".*"},
		{"admin", "/api/.*", ".*"},
		{"operator", "/api/accounts.*", "GET|POST|PUT"},
		{"operator", "/api/menu.*", ".*"},
		{"operator", "/api/materials.*", ".*"},
		{"operator", "/api/autoreply.*", ".*"},
		{"operator", "/api/fans.*", ".*"},
		{"operator", "/api/qrcode.*", ".*"},
		{"operator", "/api/auth/userinfo", "GET"},
		{"operator", "/api/system/public-ip", "GET"},
	}

	for _, rule := range rules {
		if has, _ := e.HasPolicy(rule[0], rule[1], rule[2]); !has {
			_, _ = e.AddPolicy(rule[0], rule[1], rule[2])
		}
	}
}
