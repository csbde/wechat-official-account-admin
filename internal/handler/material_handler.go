package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MaterialHandler struct{}

// List 获取素材列表 (租户隔离)
func (h *MaterialHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	mediaType := c.Query("type")
	page, pageSize := GetPaginationParams(c)

	query := database.DB.Model(&model.WxMaterial{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID)
	if mediaType != "" {
		query = query.Where("media_type = ?", mediaType)
	}

	var total int64
	query.Count(&total)

	var list []model.WxMaterial
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}

// Upload 上传素材 (绑定租户)
func (h *MaterialHandler) Upload(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)

	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请选择上传文件")
		return
	}

	mediaType := c.DefaultPostForm("media_type", "image")
	title := c.DefaultPostForm("title", file.Filename)

	saveDir := filepath.Join("files", "upload-image", fmt.Sprintf("tenant_%d", tenantID), appid)
	_ = os.MkdirAll(saveDir, 0755)

	ext := filepath.Ext(file.Filename)
	uniqueFilename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
	localPath := filepath.Join(saveDir, uniqueFilename)

	if err := c.SaveUploadedFile(file, localPath); err != nil {
		Fail(c, http.StatusInternalServerError, "保存文件失败: "+err.Error())
		return
	}

	webUrl := fmt.Sprintf("/files/upload-image/tenant_%d/%s/%s", tenantID, appid, uniqueFilename)
	mediaID := fmt.Sprintf("mock_media_%d", time.Now().UnixNano())

	material := model.WxMaterial{
		TenantID:      tenantID,
		AppID:         appid,
		MediaID:       mediaID,
		MediaType:     mediaType,
		MediaCategory: "permanent",
		Title:         title,
		URL:           webUrl,
		LocalPath:     localPath,
		FileSize:      file.Size,
	}

	database.DB.Create(&material)
	Ok(c, material)
}

// Delete 删除素材 (防越权校验)
func (h *MaterialHandler) Delete(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var material model.WxMaterial
	if err := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", id, appid, tenantID).First(&material).Error; err != nil {
		Fail(c, http.StatusNotFound, "素材不存在或无权操作")
		return
	}

	if material.LocalPath != "" {
		_ = os.Remove(material.LocalPath)
	}

	database.DB.Delete(&material)
	Ok(c, gin.H{"id": id, "deleted": true})
}

// Preview 素材预览 (租户隔离)
func (h *MaterialHandler) Preview(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var material model.WxMaterial
	if err := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", id, appid, tenantID).First(&material).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if material.LocalPath != "" && strings.HasPrefix(material.LocalPath, "files/") {
		c.File(material.LocalPath)
		return
	}

	if material.URL != "" && strings.HasPrefix(material.URL, "http") {
		resp, err := http.Get(material.URL)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			c.Header("Content-Type", resp.Header.Get("Content-Type"))
			_, _ = io.Copy(c.Writer, resp.Body)
			return
		}
	}

	c.Status(http.StatusNotFound)
}
