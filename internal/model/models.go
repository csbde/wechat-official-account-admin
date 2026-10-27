package model

import (
	"time"

	"gorm.io/gorm"
)

// Tenant 租户企业/组织
type Tenant struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Code        string         `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Description string         `gorm:"size:255" json:"description"`
	Status      int            `gorm:"default:1" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// User 系统管理员及后台用户 (支持多租户隔离与 OIDC 邮箱绑定)
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TenantID    uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	Username    string         `gorm:"size:64;not null" json:"username"`
	Email       string         `gorm:"size:128;index" json:"email"` // 邮箱 (用于与 OIDC 邮箱自动匹配绑定)
	Password    string         `gorm:"size:255;not null" json:"-"`
	Nickname    string         `gorm:"size:64" json:"nickname"`
	Avatar      string         `gorm:"size:255" json:"avatar"`
	Role        string         `gorm:"size:32;default:'operator'" json:"role"` // superadmin, admin, operator
	Status      int            `gorm:"default:1" json:"status"`                // 1: 正常, 0: 禁用
	SSOProvider string         `gorm:"size:32" json:"sso_provider,omitempty"`
	SSOUID      string         `gorm:"size:128;index" json:"sso_uid,omitempty"`
	OIDCIssuer  string         `gorm:"size:255" json:"oidc_issuer,omitempty"`
	OIDCSub     string         `gorm:"size:128;index" json:"oidc_sub,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxOpenComponent 微信开放平台-第三方平台凭证与配置
type WxOpenComponent struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	TenantID            uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	Name                string         `gorm:"size:128;not null" json:"name"`
	ComponentAppID      string         `gorm:"size:64;not null;index" json:"component_appid"`
	ComponentAppSecret  string         `gorm:"size:128;not null" json:"component_appsecret"`
	Token               string         `gorm:"size:128;not null" json:"token"`
	EncodingAESKey      string         `gorm:"size:128;not null" json:"encoding_aes_key"`
	VerifyTicket        string         `gorm:"size:255" json:"verify_ticket"`
	TicketUpdatedAt     *time.Time     `json:"ticket_updated_at"`
	AccessToken         string         `gorm:"size:512" json:"access_token,omitempty"`
	AccessTokenExpireAt *time.Time     `json:"access_token_expire_at,omitempty"`
	Enabled             bool           `gorm:"default:true" json:"enabled"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxAccount 微信公众号主体凭证及配置 (支持直连与开放平台第三方授权)
type WxAccount struct {
	ID                     uint           `gorm:"primaryKey" json:"id"`
	TenantID               uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	Name                   string         `gorm:"size:100;not null" json:"name"`
	AppID                  string         `gorm:"size:64;not null;index" json:"appid"`
	AppSecret              string         `gorm:"size:128" json:"appsecret"`
	Token                  string         `gorm:"size:128" json:"token"`
	EncodingAESKey         string         `gorm:"size:128" json:"encoding_aes_key"`
	OriginalID             string         `gorm:"size:64" json:"original_id"`
	QRCodeURL              string         `gorm:"size:255" json:"qrcode_url"`
	AvatarURL              string         `gorm:"size:255" json:"avatar_url"`
	AccountType            string         `gorm:"size:32;default:'direct'" json:"account_type"` // direct, authorized
	ComponentAppID         string         `gorm:"size:64;index" json:"component_appid,omitempty"`
	AuthorizerRefreshToken string         `gorm:"size:255" json:"authorizer_refresh_token,omitempty"`
	FuncInfo               string         `gorm:"type:text" json:"func_info,omitempty"`
	Enabled                bool           `gorm:"default:true" json:"enabled"`
	IsDefault              bool           `gorm:"default:false" json:"is_default"`
	Description            string         `gorm:"size:255" json:"description"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxMenu 微信公众号菜单数据
type WxMenu struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TenantID    uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID       string         `gorm:"size:64;index;not null" json:"appid"`
	MenuType    string         `gorm:"size:32;default:'normal'" json:"menu_type"` // normal, conditional
	MenuID      string         `gorm:"size:64" json:"menu_id"`
	Name        string         `gorm:"size:128" json:"name"`
	MenuData    string         `gorm:"type:text" json:"menu_data"`
	MatchRule   string         `gorm:"type:text" json:"match_rule"`
	Published   bool           `gorm:"default:false" json:"published"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxMaterial 微信多媒体素材
type WxMaterial struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	TenantID      uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID         string         `gorm:"size:64;index;not null" json:"appid"`
	MediaID       string         `gorm:"size:128;index;not null" json:"media_id"`
	MediaType     string         `gorm:"size:32;not null" json:"media_type"`
	MediaCategory string         `gorm:"size:32;default:'permanent'" json:"media_cat"`
	Title         string         `gorm:"size:255" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	URL           string         `gorm:"size:512" json:"url"`
	LocalPath     string         `gorm:"size:512" json:"local_path"`
	FileSize      int64          `json:"file_size"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxAutoReply 智能自动回复规则
type WxAutoReply struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	TenantID  uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID     string         `gorm:"size:64;index;not null" json:"appid"`
	RuleType  string         `gorm:"size:32;not null" json:"rule_type"`
	RuleName  string         `gorm:"size:128;not null" json:"rule_name"`
	Keywords  string         `gorm:"type:text" json:"keywords"`
	ReplyType string         `gorm:"size:32;not null" json:"reply_type"`
	Content   string         `gorm:"type:text" json:"content"`
	MediaID   string         `gorm:"size:128" json:"media_id"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxFan 微信关注粉丝用户
type WxFan struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TenantID       uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID          string         `gorm:"size:64;index;not null" json:"appid"`
	OpenID         string         `gorm:"size:64;index;not null" json:"openid"`
	UnionID        string         `gorm:"size:64;index" json:"unionid"`
	Nickname       string         `gorm:"size:128" json:"nickname"`
	Sex            int            `gorm:"default:0" json:"sex"`
	City           string         `gorm:"size:64" json:"city"`
	Province       string         `gorm:"size:64" json:"province"`
	Country        string         `gorm:"size:64" json:"country"`
	HeadImgURL     string         `gorm:"size:512" json:"headimgurl"`
	Subscribe      int            `gorm:"default:1" json:"subscribe"`
	SubscribeTime  int64          `json:"subscribe_time"`
	Remark         string         `gorm:"size:255" json:"remark"`
	TagIDs         string         `gorm:"type:text" json:"tagid_list"`
	SubscribeScene string         `gorm:"size:64" json:"subscribe_scene"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxTag 微信粉丝用户标签
type WxTag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	TenantID  uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID     string         `gorm:"size:64;index;not null" json:"appid"`
	TagID     int            `gorm:"not null" json:"tag_id"`
	Name      string         `gorm:"size:64;not null" json:"name"`
	Count     int            `gorm:"default:0" json:"count"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxQrcode 带参二维码
type WxQrcode struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	TenantID      uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID         string         `gorm:"size:64;index;not null" json:"appid"`
	QrcodeType    string         `gorm:"size:32;not null" json:"qrcode_type"`
	SceneID       int            `json:"scene_id"`
	SceneStr      string         `gorm:"size:64" json:"scene_str"`
	Ticket        string         `gorm:"size:255;not null" json:"ticket"`
	URL           string         `gorm:"size:512" json:"url"`
	ExpireSeconds int            `json:"expire_seconds"`
	ExpireAt      *time.Time     `json:"expire_at"`
	ScanCount     int            `gorm:"default:0" json:"scan_count"`
	Remark        string         `gorm:"size:255" json:"remark"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// RequestLog API 请求审计日志
type RequestLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TenantID     uint      `gorm:"index;default:1" json:"tenant_id"`
	AppID        string    `gorm:"size:64;index" json:"appid"`
	TraceID      string    `gorm:"size:64;index" json:"trace_id"`
	Method       string    `gorm:"size:16;not null" json:"method"`
	Path         string    `gorm:"size:255;not null;index" json:"path"`
	StatusCode   int       `json:"status_code"`
	ClientIP     string    `gorm:"size:64" json:"client_ip"`
	LatencyMs    int64     `json:"latency_ms"`
	RequestBody  string    `gorm:"type:text" json:"request_body,omitempty"`
	ResponseBody string    `gorm:"type:text" json:"response_body,omitempty"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

// IsThirdPartyManaged 判断该账号是否由微信开放平台第三方平台代管 (扫码授权托管)。
// 兼容两类账号: authorized (公众号) 与 miniprogram (小程序，含快速注册开通与扫码授权接入)。
// 判定依据是「是否已绑定第三方平台」，因此不依赖 account_type 的字面值，避免账号类型标注变化后凭据获取逻辑失效。
func (a *WxAccount) IsThirdPartyManaged() bool {
	return a.ComponentAppID != "" && (a.AccountType == "authorized" || a.AccountType == "miniprogram")
}

// AutoMigrate 执行所有多租户与开放平台实体的数据库自动迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Tenant{},
		&User{},
		&WxOpenComponent{},
		&WxAccount{},
		&WxMenu{},
		&WxMaterial{},
		&WxAutoReply{},
		&WxAISetting{},
		&WxFan{},
		&WxTag{},
		&WxQrcode{},
		&RequestLog{},
		&SystemOIDCSetting{},
		&WxMiniRegisterTask{},
		&WxMiniSetting{},
	)
}

// WxAISetting 微信公众号 AI 智能助手设置 (兼容 ChatGPT / OpenAI 规范)
type WxAISetting struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	TenantID     uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID        string         `gorm:"size:64;index;not null" json:"appid"`
	Enabled      bool           `gorm:"default:false" json:"enabled"`
	Provider     string         `gorm:"size:64;default:'openai'" json:"provider"` // openai, deepseek, qwen, moonshot, ollama, custom
	BaseURL      string         `gorm:"size:255;default:'https://api.openai.com/v1'" json:"base_url"`
	APIKey       string         `gorm:"size:255" json:"api_key"`
	Model        string         `gorm:"size:64;default:'gpt-4o-mini'" json:"model"`
	SystemPrompt string         `gorm:"type:text" json:"system_prompt"`
	Temperature  float64        `gorm:"default:0.7" json:"temperature"`
	MaxTokens    int            `gorm:"default:800" json:"max_tokens"`
	MatchMode    string         `gorm:"size:32;default:'fallback'" json:"match_mode"` // fallback: 未命中关键词时AI回复, all: 全部消息由AI回复
	TimeoutSec   int            `gorm:"default:4" json:"timeout_sec"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// SystemOIDCSetting OIDC 单点登录系统可视化持久化配置
type SystemOIDCSetting struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	TenantID        uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	Enabled         bool           `gorm:"default:false" json:"enabled"`
	Issuer          string         `gorm:"size:255" json:"issuer"`
	ClientID        string         `gorm:"size:128" json:"client_id"`
	ClientSecret    string         `gorm:"size:255" json:"client_secret"`
	RedirectURI     string         `gorm:"size:255" json:"redirect_uri"`
	Scopes          string         `gorm:"size:255;default:'openid profile email'" json:"scopes"`
	AutoRegister    bool           `gorm:"default:false" json:"auto_register"`             // 是否允许未录入邮箱自动创建账号 (默认关闭，防止非授权人员获取权限)
	DefaultRole     string         `gorm:"size:32;default:'operator'" json:"default_role"` // 自动注册账号时的默认权限角色 (operator / viewer)
	DefaultTenantID uint           `gorm:"index;default:1" json:"default_tenant_id"`       // 自动注册账号归属的租户组织 (默认 1)
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxMiniRegisterTask 微信开放平台快速注册小程序任务
type WxMiniRegisterTask struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	TenantID           uint           `gorm:"index;not null;default:1" json:"tenant_id"`    // 租户隔离
	Name               string         `gorm:"size:128;not null" json:"name"`                // 企业名称（需与营业执照一致）
	Code               string         `gorm:"size:64;not null" json:"code"`                 // 企业代码 (统一社会信用代码)
	CodeType           int            `gorm:"default:1" json:"code_type"`                   // 1: 统一社会信用代码, 2: 组织机构代码, 3: 营业执照注册号
	LegalPersonaWechat string         `gorm:"size:64;not null" json:"legal_persona_wechat"` // 法人微信号
	LegalPersonaName   string         `gorm:"size:64;not null" json:"legal_persona_name"`   // 法人姓名
	ComponentPhone     string         `gorm:"size:32" json:"component_phone"`               // 第三方平台联系电话
	Status             int            `gorm:"default:0" json:"status"`                      // 0: 待法人人脸核身, 1: 注册成功, 2: 微信驳回
	StatusDesc         string         `gorm:"size:255" json:"status_desc"`                  // 状态说明文本
	AppID              string         `gorm:"size:64" json:"appid"`                         // 注册成功后微信下发的小程序 AppID
	RejectReason       string         `gorm:"size:512" json:"reject_reason,omitempty"`      // 驳回原因
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxMiniSetting 小程序代开发配置与服务器域名
type WxMiniSetting struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TenantID       uint           `gorm:"index;not null;default:1" json:"tenant_id"`
	AppID          string         `gorm:"size:64;uniqueIndex;not null" json:"appid"` // 小程序 AppID
	RequestDomain  string         `gorm:"type:text" json:"requestdomain"`            // request 合法域名 (换行或逗号分隔)
	WsDomain       string         `gorm:"type:text" json:"wsrequestdomain"`          // socket 合法域名
	UploadDomain   string         `gorm:"type:text" json:"uploaddomain"`             // uploadFile 合法域名
	DownloadDomain string         `gorm:"type:text" json:"downloaddomain"`           // downloadFile 合法域名
	WebviewDomain  string         `gorm:"type:text" json:"webviewdomain"`            // 业务域名 (webview)
	CurrentVersion string         `gorm:"size:32" json:"current_version,omitempty"`
	AuditStatus    int            `gorm:"default:0" json:"audit_status"` // 0: 未提审, 1: 审核中, 2: 审核通过, 3: 审核被驳回
	AuditReason    string         `gorm:"size:512" json:"audit_reason,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
