package model

import (
	"time"

	"gorm.io/gorm"
)

// User 系统管理员及后台用户
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password    string         `gorm:"size:255;not null" json:"-"`
	Nickname    string         `gorm:"size:64" json:"nickname"`
	Avatar      string         `gorm:"size:255" json:"avatar"`
	Role        string         `gorm:"size:32;default:'admin'" json:"role"` // admin, operator, viewer
	Status      int            `gorm:"default:1" json:"status"`             // 1: 正常, 0: 禁用
	SSOProvider string         `gorm:"size:32" json:"sso_provider,omitempty"`
	SSOUID      string         `gorm:"size:128;index" json:"sso_uid,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxAccount 微信公众号主体凭证及配置
type WxAccount struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	AppID          string         `gorm:"size:64;uniqueIndex;not null" json:"appid"`
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

// WxMenu 微信公众号菜单数据
type WxMenu struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AppID       string         `gorm:"size:64;index;not null" json:"appid"`
	MenuType    string         `gorm:"size:32;default:'normal'" json:"menu_type"` // normal: 普通默认菜单, conditional: 个性化条件菜单
	MenuID      string         `gorm:"size:64" json:"menu_id"`                   // 微信返回的 menuid (个性化菜单发布后会有)
	Name        string         `gorm:"size:128" json:"name"`
	MenuData    string         `gorm:"type:text" json:"menu_data"`   // JSON 结构的 button 列表
	MatchRule   string         `gorm:"type:text" json:"match_rule"`  // JSON 结构的匹配规则 (tag_id, sex, client_platform_type, etc.)
	Published   bool           `gorm:"default:false" json:"published"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxMaterial 微信多媒体素材 (图片, 语音, 视频, 缩略图, 图文)
type WxMaterial struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AppID          string         `gorm:"size:64;index;not null" json:"appid"`
	MediaID        string         `gorm:"size:128;index;not null" json:"media_id"`
	MediaType      string         `gorm:"size:32;not null" json:"media_type"`       // image, voice, video, thumb, news
	MediaCategory  string         `gorm:"size:32;default:'permanent'" json:"media_cat"` // permanent: 永久素材, temporary: 临时素材
	Title          string         `gorm:"size:255" json:"title"`
	Description    string         `gorm:"type:text" json:"description"`
	URL            string         `gorm:"size:512" json:"url"`                       // 微信素材网络URL
	LocalPath      string         `gorm:"size:512" json:"local_path"`                // 本地磁盘存储路径或缓存路径
	FileSize       int64          `json:"file_size"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxAutoReply 智能自动回复规则
type WxAutoReply struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AppID       string         `gorm:"size:64;index;not null" json:"appid"`
	RuleType    string         `gorm:"size:32;not null" json:"rule_type"` // subscribe: 关注后回复, default: 收到消息默认回复, keyword: 关键词匹配回复
	RuleName    string         `gorm:"size:128;not null" json:"rule_name"`
	Keywords    string         `gorm:"type:text" json:"keywords"` // JSON 数组: [{"keyword": "你好", "match_type": "full" | "half"}]
	ReplyType   string         `gorm:"size:32;not null" json:"reply_type"` // text, image, voice, video, news
	Content     string         `gorm:"type:text" json:"content"`          // 文本内容
	MediaID     string         `gorm:"size:128" json:"media_id"`          // 关联的素材 MediaID
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxFan 微信关注粉丝用户
type WxFan struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AppID          string         `gorm:"size:64;index;not null" json:"appid"`
	OpenID         string         `gorm:"size:64;index;not null" json:"openid"`
	UnionID        string         `gorm:"size:64;index" json:"unionid"`
	Nickname       string         `gorm:"size:128" json:"nickname"`
	Sex            int            `gorm:"default:0" json:"sex"` // 1: 男性, 2: 女性, 0: 未知
	City           string         `gorm:"size:64" json:"city"`
	Province       string         `gorm:"size:64" json:"province"`
	Country        string         `gorm:"size:64" json:"country"`
	HeadImgURL     string         `gorm:"size:512" json:"headimgurl"`
	Subscribe      int            `gorm:"default:1" json:"subscribe"` // 1: 关注中, 0: 取消关注
	SubscribeTime  int64          `json:"subscribe_time"`
	Remark         string         `gorm:"size:255" json:"remark"`
	TagIDs         string         `gorm:"type:text" json:"tagid_list"` // JSON 数组: [100, 101]
	SubscribeScene string         `gorm:"size:64" json:"subscribe_scene"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxTag 微信粉丝用户标签
type WxTag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	AppID     string         `gorm:"size:64;index;not null" json:"appid"`
	TagID     int            `gorm:"not null" json:"tag_id"` // 微信标签ID
	Name      string         `gorm:"size:64;not null" json:"name"`
	Count     int            `gorm:"default:0" json:"count"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// WxQrcode 带参二维码
type WxQrcode struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	AppID         string         `gorm:"size:64;index;not null" json:"appid"`
	QrcodeType    string         `gorm:"size:32;not null" json:"qrcode_type"` // temp: 临时, limit: 永久
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

// RequestLog API 请求审计与性能分析日志
type RequestLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
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

// AutoMigrate 执行所有实体的数据库自动迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
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
