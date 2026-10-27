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

// List 获取素材列表
// @Summary 获取多媒体素材列表
// @Tags 素材管理
// @Security BearerAuth
// @Produce json
// @Router /api/materials [get]
func (h *MaterialHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	mediaType := c.Query("type") // image, voice, video, news
	page, pageSize := GetPaginationParams(c)

	query := database.DB.Model(&model.WxMaterial{}).Where("app_id = ?", appid)
	if mediaType != "" {
		query = query.Where("media_type = ?", mediaType)
	}

	var total int64
	query.Count(&total)

	var list []model.WxMaterial
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}

// Upload 上传素材
// @Summary 上传多媒体素材文件
// @Tags 素材管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Router /api/materials/upload [post]
func (h *MaterialHandler) Upload(c *gin.Context) {
	appid := GetAppID(c)
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请选择上传文件")
		return
	}

	mediaType := c.DefaultPostForm("media_type", "image")
	title := c.DefaultPostForm("title", file.Filename)

	saveDir := filepath.Join("files", "upload-image", appid)
	_ = os.MkdirAll(saveDir, 0755)

	ext := filepath.Ext(file.Filename)
	uniqueFilename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
	localPath := filepath.Join(saveDir, uniqueFilename)

	if err := c.SaveUploadedFile(file, localPath); err != nil {
		Fail(c, http.StatusInternalServerError, "保存文件失败: "+err.Error())
		return
	}

	webUrl := fmt.Sprintf("/files/upload-image/%s/%s", appid, uniqueFilename)
	mediaID := fmt.Sprintf("mock_media_%d", time.Now().UnixNano())

	material := model.WxMaterial{
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

// Delete 删除素材
// @Summary 删除指定素材
// @Tags 素材管理
// @Security BearerAuth
// @Produce json
// @Router /api/materials/{id} [delete]
func (h *MaterialHandler) Delete(c *gin.Context) {
	appid := GetAppID(c)
	id := c.Param("id")

	var material model.WxMaterial
	if err := database.DB.Where("app_id = ? AND id = ?", appid, id).First(&material).Error; err != nil {
		Fail(c, http.StatusNotFound, "素材不存在")
		return
	}

	if material.LocalPath != "" {
		_ = os.Remove(material.LocalPath)
	}

	database.DB.Delete(&material)
	Ok(c, gin.H{"id": id, "deleted": true})
}

// Preview 素材预览或下载直链
// @Summary 预览素材图片或媒体
// @Tags 素材管理
// @Produce image/*
// @Router /api/materials/{id}/preview [get]
func (h *MaterialHandler) Preview(c *gin.Context) {
	appid := GetAppID(c)
	id := c.Param("id")

	var material model.WxMaterial
	if err := database.DB.Where("app_id = ? AND id = ?", appid, id).First(&material).Error; err != nil {
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
