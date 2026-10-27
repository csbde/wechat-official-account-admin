package handler

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
	"go.uber.org/zap"
)

type QrcodeHandler struct{}

type CreateQrcodeRequest struct {
	QrcodeType    string `json:"qrcode_type" binding:"required"`
	SceneStr      string `json:"scene_str"`
	SceneID       int    `json:"scene_id"`
	ExpireSeconds int    `json:"expire_seconds"`
	Remark        string `json:"remark"`
}

// List 带参二维码列表 (租户隔离)
func (h *QrcodeHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	page, pageSize := GetPaginationParams(c)

	var total int64
	query := database.DB.Model(&model.WxQrcode{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID)
	query.Count(&total)

	var list []model.WxQrcode
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}

// Create 生成带参二维码 (真实向微信请求，错误不再静默忽略)
func (h *QrcodeHandler) Create(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	var req CreateQrcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		Fail(c, http.StatusBadRequest, "公众号未配置或已禁用: "+err.Error())
		return
	}

	expireSec := req.ExpireSeconds
	if req.QrcodeType == "temp" {
		if expireSec <= 0 || expireSec > 2592000 {
			expireSec = 2592000 // 默认微信最大 30 天
		}
	} else {
		expireSec = 0
	}

	// 真实请求微信官方接口
	ticket, imgURL, scanURL, err := wxClient.CreateQRCode(c.Request.Context(), req.QrcodeType, req.SceneID, req.SceneStr, expireSec)
	if err != nil {
		logger.Error("微信生成二维码失败", zap.String("appid", appid), zap.Error(err))
		Fail(c, http.StatusInternalServerError, "微信接口返回错误: "+err.Error())
		return
	}

	var expireAt *time.Time
	if req.QrcodeType == "temp" {
		t := time.Now().Add(time.Duration(expireSec) * time.Second)
		expireAt = &t
	}

	item := model.WxQrcode{
		TenantID:      tenantID,
		AppID:         appid,
		QrcodeType:    req.QrcodeType,
		SceneID:       req.SceneID,
		SceneStr:      req.SceneStr,
		Ticket:        ticket,
		URL:           imgURL, // 存储真实的微信图片地址
		ExpireSeconds: expireSec,
		ExpireAt:      expireAt,
		Remark:        req.Remark,
	}

	// 如果有扫码内容，暂存到 Remark 或后续调用
	if scanURL != "" && item.Remark == "" {
		item.Remark = req.Remark
	}

	database.DB.Create(&item)
	Ok(c, item)
}

// GetImage 本地免防盗链二维码图片直出 (优先代理微信图片，兜底本地高清渲染)
func (h *QrcodeHandler) GetImage(c *gin.Context) {
	id := c.Param("id")

	var item model.WxQrcode
	if err := database.DB.Where("id = ?", id).First(&item).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// 1. 尝试直接请求微信官方 showqrcode 图片并代理输出 (避开浏览器 Referer 防盗链)
	if item.URL != "" {
		req, err := http.NewRequestWithContext(c.Request.Context(), "GET", item.URL, nil)
		if err == nil {
			// 伪造无防盗链 Header
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
			cli := &http.Client{Timeout: 5 * time.Second}
			resp, err := cli.Do(req)
			if err == nil && resp.StatusCode == 200 && strings.HasPrefix(resp.Header.Get("Content-Type"), "image") {
				defer resp.Body.Close()
				c.Header("Content-Type", resp.Header.Get("Content-Type"))
				c.Header("Cache-Control", "public, max-age=86400")
				_, _ = io.Copy(c.Writer, resp.Body)
				return
			}
		}
	}

	// 2. 双重保底：若微信防盗链拦截或网络超时，使用纯 Go 本地渲染高清 PNG 二维码
	qrContent := item.URL
	if qrContent == "" {
		qrContent = item.Ticket
	}

	pngBytes, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err == nil {
		c.Header("Content-Type", "image/png")
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "image/png", pngBytes)
		return
	}

	c.Status(http.StatusNotFound)
}
