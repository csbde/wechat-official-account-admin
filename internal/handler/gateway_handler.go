package handler

import (
	"io"
	"net/http"

	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

type GatewayHandler struct{}

// Verify 微信开发者服务器验证接入 (GET)
// @Summary 微信回调接入认证
// @Tags 微信回调网关
// @Produce plain
// @Router /wxmp/{appid}/handler [get]
func (h *GatewayHandler) Verify(c *gin.Context) {
	appid := c.Param("appid")
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		c.String(http.StatusForbidden, "invalid appid")
		return
	}

	if wxClient.CheckSignature(signature, timestamp, nonce) {
		c.String(http.StatusOK, echostr)
		return
	}

	c.String(http.StatusForbidden, "signature mismatch")
}

// ReceiveMessage 微信服务器消息及事件推送接收 (POST)
// @Summary 接收微信消息与事件
// @Tags 微信回调网关
// @Accept xml
// @Produce xml
// @Router /wxmp/{appid}/handler [post]
func (h *GatewayHandler) ReceiveMessage(c *gin.Context) {
	appid := c.Param("appid")
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		c.String(http.StatusForbidden, "invalid appid")
		return
	}

	// 签名验证
	if !wxClient.CheckSignature(signature, timestamp, nonce) {
		c.String(http.StatusForbidden, "signature mismatch")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "read body failed")
		return
	}

	replyXml, err := service.HandleWxMessage(c.Request.Context(), appid, body)
	if err != nil || len(replyXml) == 0 {
		c.String(http.StatusOK, "success")
		return
	}

	c.Data(http.StatusOK, "application/xml; charset=utf-8", replyXml)
}
