package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

type QrcodeHandler struct{}

type CreateQrcodeRequest struct {
	QrcodeType    string `json:"qrcode_type" binding:"required"` // temp, limit
	SceneStr      string `json:"scene_str"`
	SceneID       int    `json:"scene_id"`
	ExpireSeconds int    `json:"expire_seconds"` // 针对 temp，默认 2592000
	Remark        string `json:"remark"`
}

// List 带参二维码列表
// @Summary 获取带参二维码列表
// @Tags 带参二维码
// @Security BearerAuth
// @Produce json
// @Router /api/qrcode [get]
func (h *QrcodeHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	page, pageSize := GetPaginationParams(c)

	var total int64
	query := database.DB.Model(&model.WxQrcode{}).Where("app_id = ?", appid)
	query.Count(&total)

	var list []model.WxQrcode
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}

// Create 生成带参二维码
// @Summary 创建新二维码
// @Tags 带参二维码
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/qrcode [post]
func (h *QrcodeHandler) Create(c *gin.Context) {
	appid := GetAppID(c)
	var req CreateQrcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	wxClient, err := service.NewWxClient(appid)
	ticket := fmt.Sprintf("mock_ticket_%d", time.Now().UnixNano())
	qrUrl := fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s", ticket)

	if err == nil {
		t, u, qErr := wxClient.CreateQRCode(c.Request.Context(), req.QrcodeType, req.SceneID, req.SceneStr, req.ExpireSeconds)
		if qErr == nil {
			ticket = t
			qrUrl = u
		}
	}

	var expireAt *time.Time
	if req.QrcodeType == "temp" {
		sec := req.ExpireSeconds
		if sec <= 0 {
			sec = 2592000
		}
		t := time.Now().Add(time.Duration(sec) * time.Second)
		expireAt = &t
	}

	item := model.WxQrcode{
		AppID:         appid,
		QrcodeType:    req.QrcodeType,
		SceneID:       req.SceneID,
		SceneStr:      req.SceneStr,
		Ticket:        ticket,
		URL:           qrUrl,
		ExpireSeconds: req.ExpireSeconds,
		ExpireAt:      expireAt,
		Remark:        req.Remark,
	}

	database.DB.Create(&item)
	Ok(c, item)
}
