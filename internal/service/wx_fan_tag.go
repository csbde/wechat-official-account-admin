package service

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/model"
	"gorm.io/gorm"
)

// ==========================================================
// 粉丝标签 (wx_fans.tag_ids) 的统一读写工具
// 存储格式与微信一致：紧凑 JSON 数组文本，例如 "[1,2,11]"
// 「仅本地」标签 (微信侧还没建成功) 用负的本地主键作占位 ID，绝不会与微信真实 tag_id 冲突
// ==========================================================

// FanTagIDsColumn 粉丝标签列名 (对应 model.WxFan.TagIDs)
const FanTagIDsColumn = "tag_ids"

// FanTagIDsMatchExpr 生成「tag_ids 精确包含指定标签 ID」的 WHERE 片段 (兼容 SQLite 与 PostgreSQL)。
// 把 "[1,2,11]" 先规范成 ",1,2,11,"，再用 ",<id>," 匹配；
// 直接 LIKE '%1%' 会把 11/101 一起命中，筛选与人数统计都会错。
func FanTagIDsMatchExpr() string {
	return "',' || REPLACE(REPLACE(REPLACE(" + FanTagIDsColumn + ", '[', ''), ']', ''), ' ', '') || ',' LIKE ?"
}

// FanTagIDsLikeArg 生成 LIKE 匹配串 "%,<id>,%" (带通配符，配合 FanTagIDsMatchExpr 使用)
func FanTagIDsLikeArg(tagID int) string {
	return "%," + strconv.Itoa(tagID) + ",%"
}

// ParseFanTagIDs 解析粉丝标签列表，容忍空值与脏数据
func ParseFanTagIDs(raw string) []int {
	ids := make([]int, 0, 8)
	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return ids
	}
	var list []int
	if err := json.Unmarshal([]byte(s), &list); err == nil {
		ids = append(ids, list...)
	}
	return UniquePositiveAndNegativeInts(ids)
}

// MarshalFanTagIDs 序列化为微信风格的紧凑 JSON 数组文本
func MarshalFanTagIDs(ids []int) string {
	clean := UniquePositiveAndNegativeInts(ids)
	if len(clean) == 0 {
		return "[]"
	}
	b, err := json.Marshal(clean)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// LocalOnlyFanTagID 「仅本地」标签的占位 ID：取负的本地主键
func LocalOnlyFanTagID(id uint) int {
	return -int(id)
}

// IsLocalOnlyFanTagID 判断标签 ID 是否属于仅本地占位 ID
func IsLocalOnlyFanTagID(tagID int) bool {
	return tagID < 0
}

// MergeLocalOnlyFanTagIDs 合并微信返回的真实标签与本地独有的占位标签。
// 全量粉丝同步时微信返回的 tagid_list 不含本地标签，直接覆盖会把刚「打上但还没同步成功」的标签抹掉。
func MergeLocalOnlyFanTagIDs(remote []int, existingRaw string) []int {
	out := UniquePositiveAndNegativeInts(remote)
	seen := make(map[int]bool, len(out))
	for _, v := range out {
		seen[v] = true
	}
	for _, v := range ParseFanTagIDs(existingRaw) {
		if v < 0 && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// UniquePositiveAndNegativeInts 去重并稳定排序 (负数占位 ID 排在前面，便于人工核对)
func UniquePositiveAndNegativeInts(items []int) []int {
	seen := make(map[int]bool, len(items))
	out := make([]int, 0, len(items))
	for _, v := range items {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// RemapFanTagID 把粉丝身上的旧标签 ID 换成新 ID；newID <= 0 表示直接移除 (用于删除标签后清理残留)。
// 采用 id 游标分页，避免大账号一次性载入全量粉丝。返回处理的粉丝条数。
func RemapFanTagID(tx *gorm.DB, appid string, tenantID uint, oldID, newID int) int64 {
	if oldID == 0 {
		return 0
	}
	var changed int64
	after := uint(0)
	for {
		var rows []model.WxFan
		err := tx.Model(&model.WxFan{}).
			Select("id", "tag_ids").
			Where("app_id = ? AND tenant_id = ? AND id > ?", appid, tenantID, after).
			Where(FanTagIDsMatchExpr(), FanTagIDsLikeArg(oldID)).
			Order("id asc").
			Limit(500).
			Find(&rows).Error
		if err != nil || len(rows) == 0 {
			return changed
		}
		for _, r := range rows {
			ids := ParseFanTagIDs(r.TagIDs)
			next := make([]int, 0, len(ids))
			hit := false
			for _, v := range ids {
				if v == oldID {
					hit = true
					if newID > 0 {
						next = append(next, newID)
					}
					continue
				}
				next = append(next, v)
			}
			if !hit {
				continue
			}
			if err := tx.Model(&model.WxFan{}).Where("id = ?", r.ID).
				Update("tag_ids", MarshalFanTagIDs(next)).Error; err == nil {
				changed++
			}
		}
		after = rows[len(rows)-1].ID
		if len(rows) < 500 {
			return changed
		}
	}
}
