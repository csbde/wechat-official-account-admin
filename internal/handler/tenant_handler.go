package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// TenantHandler 租户(组织)管理：只有超级管理员可以新建/修改/删除租户，
// 普通后台用户仅能只读查看自己所属的那个租户，避免跨租户探测。
type TenantHandler struct{}

// 租户唯一标识：字母开头，允许字母数字下划线中划线，长度 2-64
var tenantCodePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,63}$`)

// TenantView 租户列表视图（附带资源统计与视角标记）
type TenantView struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Description  string    `json:"description"`
	Status       int       `json:"status"`
	AccountCount int64     `json:"account_count"`
	UserCount    int64     `json:"user_count"`
	IsCurrent    bool      `json:"is_current"` // 是否为当前操作视角租户
	IsHome       bool      `json:"is_home"`    // 是否为当前登录用户真实所属租户
	Ghost        bool      `json:"ghost"`      // 是否为"数据残留但租户记录缺失"的幽灵租户
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type tenantCountRow struct {
	TenantID uint
	Cnt      int64
}

// loadTenantCountMap 一次 group by 统计各租户资源数量，避免列表 N+1 查询
func loadTenantCountMap(modelValue interface{}) map[uint]int64 {
	var rows []tenantCountRow
	database.DB.Model(modelValue).Select("tenant_id, COUNT(*) AS cnt").Group("tenant_id").Scan(&rows)
	result := make(map[uint]int64, len(rows))
	for _, r := range rows {
		result[r.TenantID] = r.Cnt
	}
	return result
}

// List 租户列表 (超级管理员看全部，普通用户只看自己所属租户)
func (h *TenantHandler) List(c *gin.Context) {
	activeTenantID := GetTenantID(c)
	homeTenantID := GetHomeTenantID(c)

	query := database.DB.Model(&model.Tenant{})
	if !IsSuperAdmin(c) {
		query = query.Where("id = ?", homeTenantID)
	} else if tid := strings.TrimSpace(c.Query("tenant_id")); tid != "" {
		query = query.Where("id = ?", tid)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", like, like, like)
	}

	var tenants []model.Tenant
	if err := query.Order("id asc").Find(&tenants).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "查询租户列表失败: "+err.Error())
		return
	}

	accMap := loadTenantCountMap(&model.WxAccount{})
	userMap := loadTenantCountMap(&model.User{})

	views := make([]TenantView, 0, len(tenants)+2)
	for _, t := range tenants {
		views = append(views, TenantView{
			ID:           t.ID,
			Name:         t.Name,
			Code:         t.Code,
			Description:  t.Description,
			Status:       t.Status,
			AccountCount: accMap[t.ID],
			UserCount:    userMap[t.ID],
			IsCurrent:    t.ID == activeTenantID,
			IsHome:       t.ID == homeTenantID,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}
	// 超级管理员额外暴露"幽灵租户": 被账号/成员引用但租户记录缺失的 ID，
	// 这些历史数据否则谁都看不到、也无处纠正，是"我是超管却没超管权限"的常见根因。
	if IsSuperAdmin(c) {
		known := make(map[uint]bool, len(tenants))
		for _, t := range tenants {
			known[t.ID] = true
		}
		for _, id := range referencedTenantIDs() {
			if known[id] || id == 0 {
				continue
			}
			views = append(views, TenantView{
				ID:           id,
				Name:         fmt.Sprintf("未登记租户 #%d (数据残留)", id),
				Code:         fmt.Sprintf("ghost-%d", id),
				Description:  "该租户记录不存在或已被彻底删除，但仍有公众号/成员数据引用它。请切换到该视角把账号迁移到正式租户，或清理无用数据。",
				Status:       1,
				AccountCount: accMap[id],
				UserCount:    userMap[id],
				IsCurrent:    id == activeTenantID,
				Ghost:        true,
			})
		}
	}

	Ok(c, views)
}

// referencedTenantIDs 返回所有被业务数据引用但 tenants 表中不存在的租户 ID
func referencedTenantIDs() []uint {
	type idRow struct {
		TenantID uint
	}
	seen := map[uint]bool{}
	out := make([]uint, 0, 4)
	for _, tableName := range []string{"wx_accounts", "users"} {
		var rows []idRow
		database.DB.Unscoped().Table(tableName).Select("DISTINCT tenant_id").Scan(&rows)
		for _, r := range rows {
			if r.TenantID == 0 || seen[r.TenantID] {
				continue
			}
			var exist int64
			database.DB.Unscoped().Model(&model.Tenant{}).Where("id = ?", r.TenantID).Count(&exist)
			if exist == 0 {
				seen[r.TenantID] = true
				out = append(out, r.TenantID)
			}
		}
	}
	return out
}

// Options 极简租户下拉选项 (超级管理员分配账号/用户归属时使用，不暴露描述等明细)
func (h *TenantHandler) Options(c *gin.Context) {
	query := database.DB.Model(&model.Tenant{}).Where("status = 1")
	if !IsSuperAdmin(c) {
		query = query.Where("id = ?", GetHomeTenantID(c))
	}
	var tenants []model.Tenant
	if err := query.Select("id", "name", "code", "status").Order("id asc").Find(&tenants).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "查询租户选项失败: "+err.Error())
		return
	}
	Ok(c, tenants)
}

// Create 新建租户 (仅超级管理员)
func (h *TenantHandler) Create(c *gin.Context) {
	if !IsSuperAdmin(c) {
		Fail(c, http.StatusForbidden, "越权操作：仅超级管理员可以创建租户组织")
		return
	}

	var req struct {
		Name          string `json:"name" binding:"required"`
		Code          string `json:"code" binding:"required"`
		Description   string `json:"description"`
		Status        *int   `json:"status"`
		AdminUsername string `json:"admin_username"`
		AdminPassword string `json:"admin_password"`
		AdminEmail    string `json:"admin_email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}

	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if name == "" || len([]rune(name)) > 128 {
		Fail(c, http.StatusBadRequest, "租户名称不能为空且长度不能超过 128 个字符")
		return
	}
	if !tenantCodePattern.MatchString(code) {
		Fail(c, http.StatusBadRequest, "租户标识(code)格式不合法：需字母开头，仅允许字母、数字、下划线与中划线，长度 2-64")
		return
	}

	status := 1
	if req.Status != nil {
		status = 0
		if *req.Status == 1 {
			status = 1
		}
	}

	// 软删除记录也占用 code 唯一索引：若命中已删除租户则复用并恢复
	var deleted model.Tenant
	restored := false
	if err := database.DB.Unscoped().Where("code = ? AND deleted_at IS NOT NULL", code).First(&deleted).Error; err == nil {
		deleted.Name = name
		deleted.Description = strings.TrimSpace(req.Description)
		deleted.Status = status
		deleted.DeletedAt = gorm.DeletedAt{}
		if err := database.DB.Unscoped().Omit("deleted_at").Save(&deleted).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "恢复历史租户失败: "+err.Error())
			return
		}
		if err := database.DB.Unscoped().Model(&model.Tenant{}).Where("id = ?", deleted.ID).Update("deleted_at", nil).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "恢复历史租户失败: "+err.Error())
			return
		}
		restored = true
	}

	tenant := deleted
	if !restored {
		var cnt int64
		database.DB.Unscoped().Model(&model.Tenant{}).Where("code = ?", code).Count(&cnt)
		if cnt > 0 {
			Fail(c, http.StatusBadRequest, "租户标识 ["+code+"] 已存在，请换一个标识")
			return
		}
		tenant = model.Tenant{
			Name:        name,
			Code:        code,
			Description: strings.TrimSpace(req.Description),
			Status:      status,
		}
		if err := database.DB.Create(&tenant).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "创建租户失败: "+err.Error())
			return
		}
	}

	// 可选：同时为该租户创建初始管理员，便于新组织直接登录运营
	var adminUser *model.User
	if adminName := strings.TrimSpace(req.AdminUsername); adminName != "" {
		pwd := req.AdminPassword
		if len(pwd) < 6 {
			Fail(c, http.StatusBadRequest, "租户初始化管理员密码长度不能少于 6 位")
			return
		}
		var sameName int64
		database.DB.Model(&model.User{}).Where("username = ?", adminName).Count(&sameName)
		if sameName > 0 {
			Fail(c, http.StatusBadRequest, "用户名 ["+adminName+"] 已存在，租户内用户登录名需全局唯一")
			return
		}
		adminEmail := strings.ToLower(strings.TrimSpace(req.AdminEmail))
		if adminEmail != "" {
			var sameMail int64
			database.DB.Model(&model.User{}).Where("email = ?", adminEmail).Count(&sameMail)
			if sameMail > 0 {
				Fail(c, http.StatusBadRequest, "邮箱 ["+adminEmail+"] 已被其他账号绑定")
				return
			}
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		u := model.User{
			TenantID: tenant.ID,
			Username: adminName,
			Email:    adminEmail,
			Password: string(hash),
			Nickname: name + "管理员",
			Role:     "admin",
			Status:   1,
		}
		if err := database.DB.Create(&u).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "创建租户管理员失败: "+err.Error())
			return
		}
		adminUser = &u
	}

	logger.Info("Tenant created",
		zap.Uint("tenant_id", tenant.ID),
		zap.String("tenant_code", tenant.Code),
		zap.Bool("restored", restored),
		zap.String("operator", c.GetString("username")),
	)

	Ok(c, gin.H{
		"tenant":           tenant,
		"restored":         restored,
		"admin_user":       adminUser,
		"active_view_hint": "如需在该租户下录入公众号，请先在顶栏切换到该租户视角",
	})
}

// Update 更新租户名称/描述/启停 (仅超级管理员；code 作为唯一标识创建后不可修改)
func (h *TenantHandler) Update(c *gin.Context) {
	if !IsSuperAdmin(c) {
		Fail(c, http.StatusForbidden, "越权操作：仅超级管理员可以修改租户组织")
		return
	}

	var tenant model.Tenant
	if err := database.DB.First(&tenant, c.Param("id")).Error; err != nil {
		Fail(c, http.StatusNotFound, "租户不存在")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}

	updates := map[string]interface{}{}
	if name := strings.TrimSpace(req.Name); name != "" {
		if len([]rune(name)) > 128 {
			Fail(c, http.StatusBadRequest, "租户名称长度不能超过 128 个字符")
			return
		}
		updates["name"] = name
	}
	updates["description"] = strings.TrimSpace(req.Description)

	if req.Status != nil {
		newStatus := 0
		if *req.Status == 1 {
			newStatus = 1
		}
		if newStatus == 0 && tenant.ID == GetHomeTenantID(c) {
			Fail(c, http.StatusBadRequest, "不能停用您本人所属的租户组织，否则将导致无法登录")
			return
		}
		updates["status"] = newStatus
	}

	if err := database.DB.Model(&tenant).Updates(updates).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "更新租户失败: "+err.Error())
		return
	}
	database.DB.First(&tenant, tenant.ID)
	Ok(c, tenant)
}

// Delete 删除租户 (仅超级管理员，且必须为空租户：不存在公众号与用户，防止数据被孤立)
func (h *TenantHandler) Delete(c *gin.Context) {
	if !IsSuperAdmin(c) {
		Fail(c, http.StatusForbidden, "越权操作：仅超级管理员可以删除租户组织")
		return
	}

	var tenant model.Tenant
	if err := database.DB.First(&tenant, c.Param("id")).Error; err != nil {
		Fail(c, http.StatusNotFound, "租户不存在")
		return
	}

	if tenant.ID == GetHomeTenantID(c) {
		Fail(c, http.StatusBadRequest, "不能删除您本人所属的租户组织，请先切换账号后再操作")
		return
	}

	var accCount, userCount int64
	database.DB.Model(&model.WxAccount{}).Where("tenant_id = ?", tenant.ID).Count(&accCount)
	database.DB.Model(&model.User{}).Where("tenant_id = ?", tenant.ID).Count(&userCount)
	if accCount > 0 || userCount > 0 {
		Fail(c, http.StatusBadRequest, fmt.Sprintf("该租户下仍有 %d 个微信账号、%d 个后台用户，请先迁移或删除后再删除租户", accCount, userCount))
		return
	}

	if err := database.DB.Delete(&tenant).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "删除租户失败: "+err.Error())
		return
	}
	logger.Warn("Tenant deleted", zap.Uint("tenant_id", tenant.ID), zap.String("operator", c.GetString("username")))
	Ok(c, gin.H{"id": tenant.ID, "deleted": true})
}
