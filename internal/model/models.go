package model

import (
	"time"

	"gorm.io/gorm"
)

// Tenant 租户企业/组织
type Tenant struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Code        string         `gorm:"size:64;uniqueIndex;not null" json:"code"` // 租户唯一标识
	Description string         `gorm:"size:255" json:"description"`
	Status      int            `gorm:"default:1" json:"status"` // 1: 启用, 0: 禁用
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// User 系统管理员及后台用户 (多租户隔离)
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TenantID    uint           `gorm:"index;not null;default:1" json:"tenant_id"` // 所属租户
	Username    string         `gorm:"size:64;not null" json:"username"`          // 同一租户下唯一
	Password    string         `gorm:"size:255;not null" json:"-"`
	Nickname    string         `gorm:"size:64" json:"nickname"`
	Avatar      string         `gorm:"size:255" json:"avatar"`
	Role        string         `gorm:"size:32;default:'operator'" json:"role"` // superadmin, admin, operator
	Status      int            `gorm:"default:1" json:"status"`                // 1: 正常, 0: 禁用
	SSOProvider string         `gorm:"size:32" json:"sso_provider,omitempty"`
	SSOUID      string         `gorm:"size:128;index" json:"sso_uid,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxAccount 微信公众号主体凭证及配置 (多租户隔离)
type WxAccount struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TenantID       uint           `gorm:"index;not null;default:1" json:"tenant_id"` // 强制多租户隔离
	Name           string         `gorm:"size:100;not null" json:"name"`
	AppID          string         `gorm:"size:64;not null" json:"appid"`
	AppSecret      string         `gorm:"size:128;not null" json:"appsecret"`
	Token          string         `gorm:"size:128" json:"token"`
	EncodingAESKey string         `gorm:"size:128" json:"encoding_aes_key"`
	OriginalID     string         `gorm:"size:64" json:"original_id"`
	QRCodeURL      string         `gorm:"size:255" json:"qrcode_url"`
	AvatarURL      string         `gorm:"size:255" json:"avatar_url"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	IsDefault      bool           `gorm:"default:false" json:"is_default"`
	Description    string         `gorm:"size:255" json:"description"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxMenu 微信公众号菜单数据 (多租户隔离)
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

// WxMaterial 微信多媒体素材 (多租户隔离)
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

// WxAutoReply 智能自动回复规则 (多租户隔离)
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

// WxFan 微信关注粉丝用户 (多租户隔离)
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

// WxTag 微信粉丝用户标签 (多租户隔离)
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

// WxQrcode 带参二维码 (多租户隔离)
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

// RequestLog API 请求审计日志 (多租户隔离)
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

// AutoMigrate 执行所有多租户实体的数据库自动迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Tenant{},
		&User{},
		&WxAccount{},
		&WxMenu{},
		&WxMaterial{},
		&WxAutoReply{},
		&WxFan{},
		&WxTag{},
		&WxQrcode{},
		&RequestLog{},
	)
}
