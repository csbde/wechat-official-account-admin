package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

type MenuHandler struct{}

type SaveMenuRequest struct {
	MenuData string `json:"menu_data" binding:"required"` // JSON 字符串: {"button": [...]}
}

// Get 获取当前普通自定义菜单
// @Summary 获取当前默认菜单
// @Tags 自定义菜单
// @Security BearerAuth
// @Produce json
// @Router /api/menu [get]
func (h *MenuHandler) Get(c *gin.Context) {
	appid := GetAppID(c)
	var menu model.WxMenu
	err := database.DB.Where("app_id = ? AND menu_type = 'normal'", appid).First(&menu).Error
	if err != nil {
		Ok(c, gin.H{"menu_data": `{"button":[]}`, "published": false})
		return
	}
	Ok(c, menu)
}

// Save 保存草稿
// @Summary 保存菜单草稿
// @Tags 自定义菜单
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/menu [post]
func (h *MenuHandler) Save(c *gin.Context) {
	appid := GetAppID(c)
	var req SaveMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var menu model.WxMenu
	err := database.DB.Where("app_id = ? AND menu_type = 'normal'", appid).First(&menu).Error
	if err != nil {
		menu = model.WxMenu{
			AppID:    appid,
			MenuType: "normal",
			Name:     "默认菜单",
			MenuData: req.MenuData,
		}
		database.DB.Create(&menu)
	} else {
		menu.MenuData = req.MenuData
		database.DB.Save(&menu)
	}

	Ok(c, menu)
}

// Publish 发布到微信服务器
// @Summary 发布菜单至微信服务器
// @Tags 自定义菜单
// @Security BearerAuth
// @Produce json
// @Router /api/menu/publish [post]
func (h *MenuHandler) Publish(c *gin.Context) {
	appid := GetAppID(c)
	var menu model.WxMenu
	if err := database.DB.Where("app_id = ? AND menu_type = 'normal'", appid).First(&menu).Error; err != nil {
		Fail(c, http.StatusBadRequest, "尚未配置菜单内容")
		return
	}

	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var menuPayload interface{}
	if err := json.Unmarshal([]byte(menu.MenuData), &menuPayload); err != nil {
		Fail(c, http.StatusBadRequest, "菜单 JSON 格式无效: "+err.Error())
		return
	}

	if err := wxClient.CreateWxMenu(c.Request.Context(), menuPayload); err != nil {
		Fail(c, http.StatusInternalServerError, "微信发布失败: "+err.Error())
		return
	}

	now := time.Now()
	menu.Published = true
	menu.PublishedAt = &now
	database.DB.Save(&menu)

	Ok(c, gin.H{"status": "published", "published_at": now})
}

// SyncFromWx 从微信服务器同步菜单
// @Summary 从微信拉取线上菜单
// @Tags 自定义菜单
// @Security BearerAuth
// @Produce json
// @Router /api/menu/sync [post]
func (h *MenuHandler) SyncFromWx(c *gin.Context) {
	appid := GetAppID(c)
	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	remoteMenu, err := wxClient.GetWxMenu(c.Request.Context())
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取微信菜单失败: "+err.Error())
		return
	}

	menuObj, exists := remoteMenu["menu"]
	if !exists {
		Fail(c, http.StatusNotFound, "微信服务器未配置自定义菜单")
		return
	}

	bytes, _ := json.Marshal(menuObj)
	var menu model.WxMenu
	err = database.DB.Where("app_id = ? AND menu_type = 'normal'", appid).First(&menu).Error
	now := time.Now()
	if err != nil {
		menu = model.WxMenu{
			AppID:       appid,
			MenuType:    "normal",
			Name:        "同步菜单",
			MenuData:    string(bytes),
			Published:   true,
			PublishedAt: &now,
		}
		database.DB.Create(&menu)
	} else {
		menu.MenuData = string(bytes)
		menu.Published = true
		menu.PublishedAt = &now
		database.DB.Save(&menu)
	}

	Ok(c, menu)
}

// Delete 删除菜单
// @Summary 删除自定义菜单
// @Tags 自定义菜单
// @Security BearerAuth
// @Produce json
// @Router /api/menu [delete]
func (h *MenuHandler) Delete(c *gin.Context) {
	appid := GetAppID(c)
	wxClient, err := service.NewWxClient(appid)
	if err == nil {
		_ = wxClient.DeleteWxMenu(c.Request.Context())
	}
	database.DB.Where("app_id = ? AND menu_type = 'normal'", appid).Delete(&model.WxMenu{})
	Ok(c, gin.H{"deleted": true})
}

// GetConditionals 获取个性化菜单列表
// @Summary 获取个性化菜单列表
// @Tags 自定义菜单
// @Security BearerAuth
// @Produce json
// @Router /api/menu/conditional [get]
func (h *MenuHandler) GetConditionals(c *gin.Context) {
	appid := GetAppID(c)
	var list []model.WxMenu
	database.DB.Where("app_id = ? AND menu_type = 'conditional'", appid).Find(&list)
	Ok(c, list)
}

// SaveConditional 保存并发布个性化菜单
// @Summary 保存并发布个性化菜单
// @Tags 自定义菜单
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/menu/conditional [post]
func (h *MenuHandler) SaveConditional(c *gin.Context) {
	appid := GetAppID(c)
	var req struct {
		Name      string `json:"name" binding:"required"`
		MenuData  string `json:"menu_data" binding:"required"`
		MatchRule string `json:"match_rule" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var payload map[string]interface{}
	var buttons map[string]interface{}
	var matchRule map[string]interface{}
	_ = json.Unmarshal([]byte(req.MenuData), &buttons)
	_ = json.Unmarshal([]byte(req.MatchRule), &matchRule)

	payload = map[string]interface{}{
		"button":     buttons["button"],
		"matchrule": matchRule,
	}

	menuid, err := wxClient.CreateConditionalMenu(c.Request.Context(), payload)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "微信个性化菜单创建失败: "+err.Error())
		return
	}

	now := time.Now()
	menu := model.WxMenu{
		AppID:       appid,
		MenuType:    "conditional",
		MenuID:      menuid,
		Name:        req.Name,
		MenuData:    req.MenuData,
		MatchRule:   req.MatchRule,
		Published:   true,
		PublishedAt: &now,
	}
	database.DB.Create(&menu)

	Ok(c, menu)
}

// DeleteConditional 删除个性化菜单
// @Summary 删除指定个性化菜单
// @Tags 自定义菜单
// @Security BearerAuth
// @Produce json
// @Router /api/menu/conditional/{id} [delete]
func (h *MenuHandler) DeleteConditional(c *gin.Context) {
	appid := GetAppID(c)
	id := c.Param("id")
	var menu model.WxMenu
	if err := database.DB.Where("app_id = ? AND id = ?", appid, id).First(&menu).Error; err != nil {
		Fail(c, http.StatusNotFound, "菜单不存在")
		return
	}

	if menu.MenuID != "" {
		wxClient, err := service.NewWxClient(appid)
		if err == nil {
			_ = wxClient.DeleteConditionalMenu(c.Request.Context(), menu.MenuID)
		}
	}

	database.DB.Delete(&menu)
	Ok(c, gin.H{"id": id, "deleted": true})
}
