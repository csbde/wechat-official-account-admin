package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"go.uber.org/zap"
)

// ==========================================================
// 粉丝标签「逐条重试」任务
// 背景: 微信 tags/members/batchtagging 是整批语义，一批 50 人里只要有一个
// openid 不属于当前公众号 (已取关/来自别的号)，整批都会失败，看不出到底是谁的问题。
// 因此批量接口在整批被拒后把剩余 openid 交给队列，逐条调用并把失败者点名。
// ==========================================================

// FanTagRetryFailure 单个 openid 的逐条重试失败记录
type FanTagRetryFailure struct {
	OpenID string `json:"openid"`
	Reason string `json:"reason"`
}

// FanTagRetryStatus 逐条重试任务进度 (与粉丝同步任务同样的进程内状态口径)
type FanTagRetryStatus struct {
	AppID      string               `json:"appid"`
	TenantID   uint                 `json:"tenant_id"`
	TagID      int                  `json:"tag_id"`
	TagName    string               `json:"tag_name"`
	Action     string               `json:"action"`
	Status     string               `json:"status"` // idle running completed partial failed
	Total      int                  `json:"total"`
	Done       int                  `json:"done"`
	OK         int                  `json:"ok"`
	Failed     int                  `json:"failed"`
	Percent    int                  `json:"percent"`
	Message    string               `json:"message"`
	Error      string               `json:"error,omitempty"`
	Failures   []FanTagRetryFailure `json:"failures,omitempty"`
	StartTime  time.Time            `json:"start_time"`
	FinishTime time.Time            `json:"finish_time,omitempty"`
}

// maxRecordedFailures 最多留存的失败明细，避免大名单把内存撑爆
const maxRecordedFailures = 50

type FanTagRetryManager struct {
	tasks map[string]*FanTagRetryStatus
	mutex sync.RWMutex
}

var DefaultFanTagRetryManager = &FanTagRetryManager{
	tasks: make(map[string]*FanTagRetryStatus),
}

// GetStatus 获取指定公众号最近的逐条重试进度
func (m *FanTagRetryManager) GetStatus(appid string) *FanTagRetryStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if s, ok := m.tasks[appid]; ok {
		return s
	}
	return &FanTagRetryStatus{AppID: appid, Status: "idle", Message: "暂无标签重试任务"}
}

func (m *FanTagRetryManager) update(appid string, fn func(s *FanTagRetryStatus)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	s, ok := m.tasks[appid]
	if !ok {
		s = &FanTagRetryStatus{AppID: appid, StartTime: time.Now()}
		m.tasks[appid] = s
	}
	fn(s)
}

// RetryTaggingOneByOne 逐个 openid 向微信官方打/取消标签。
// 本地 tag_ids 在批量接口里已经写上 (需求：本地先写上并标记未同步)，
// 这里只负责把微信侧补齐，并把确实失败的 openid 点名列出。
func (m *FanTagRetryManager) RetryTaggingOneByOne(ctx context.Context, tenantID uint, appid string, tagID int, action string, openIDs []string) error {
	m.update(appid, func(s *FanTagRetryStatus) {
		s.TenantID = tenantID
		s.TagID = tagID
		s.Action = action
		s.Status = "running"
		s.Total = len(openIDs)
		s.Done, s.OK, s.Failed, s.Percent = 0, 0, 0, 0
		s.Failures = nil
		s.Error = ""
		s.Message = "正在逐条向微信官方同步标签..."
		s.StartTime = time.Now()
		s.FinishTime = time.Time{}
	})

	fail := func(reason string) error {
		m.update(appid, func(s *FanTagRetryStatus) {
			s.Status = "failed"
			s.Error = reason
			s.Message = "标签逐条重试未能执行: " + reason
			s.FinishTime = time.Now()
		})
		logger.Warn("[粉丝标签逐条重试] 任务终止", zap.String("appid", appid), zap.Int("tag_id", tagID), zap.String("reason", reason))
		return nil
	}

	// 标签必须是微信真实标签，否则逐条重试同样会失败
	var tag model.WxTag
	if err := database.DB.Where("tag_id = ? AND app_id = ? AND tenant_id = ?", tagID, appid, tenantID).First(&tag).Error; err != nil {
		return fail(fmt.Sprintf("标签不存在或不属于当前账号 (tag_id=%d)", tagID))
	}
	if !tag.Synced || IsLocalOnlyFanTagID(tag.TagID) {
		return fail("该标签仅在本地记录，尚未同步到微信官方，请先在「标签管理」点「重新同步到微信」")
	}
	m.update(appid, func(s *FanTagRetryStatus) { s.TagName = tag.Name })

	wxClient, err := NewWxClient(appid)
	if err != nil {
		return fail(err.Error())
	}

	verb := "打标签"
	if action != "add" {
		verb = "取消标签"
	}

	for i, openid := range openIDs {
		var werr error
		if action == "add" {
			werr = wxClient.BatchTaggingMembers(ctx, tag.TagID, []string{openid})
		} else {
			werr = wxClient.BatchUntaggingMembers(ctx, tag.TagID, []string{openid})
		}
		if werr != nil {
			reason := FriendlyWxError(werr).Error()
			m.update(appid, func(s *FanTagRetryStatus) {
				s.Failed++
				if len(s.Failures) < maxRecordedFailures {
					s.Failures = append(s.Failures, FanTagRetryFailure{OpenID: openid, Reason: reason})
				}
			})
			logger.Warn("[粉丝标签逐条重试] 单个 openid 失败",
				zap.String("appid", appid), zap.Int("tag_id", tag.TagID), zap.String("openid", openid), zap.String("error", reason))
		} else {
			m.update(appid, func(s *FanTagRetryStatus) { s.OK++ })
		}

		done := i + 1
		m.update(appid, func(s *FanTagRetryStatus) {
			s.Done = done
			s.Percent = int(float64(done) / float64(maxInt(s.Total, 1)) * 100)
		})

		// 逐条调用容易触发频控，轻微限速
		select {
		case <-ctx.Done():
			m.update(appid, func(s *FanTagRetryStatus) {
				s.Status = "failed"
				s.Error = ctx.Err().Error()
				s.Message = "标签逐条重试已取消"
				s.FinishTime = time.Now()
			})
			return ctx.Err()
		case <-time.After(30 * time.Millisecond):
		}
	}

	var count int64
	database.DB.Model(&model.WxFan{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID).
		Where(FanTagIDsMatchExpr(), FanTagIDsLikeArg(tag.TagID)).Count(&count)
	database.DB.Model(&model.WxTag{}).Where("id = ?", tag.ID).Update("count", count)

	status := "completed"
	msg := fmt.Sprintf("逐条%s完成：成功 %d / %d，当前标签共 %d 人", verb, 0, len(openIDs), count)
	m.update(appid, func(s *FanTagRetryStatus) {
		if s.Failed > 0 {
			status = "partial"
			if s.OK == 0 {
				status = "failed"
			}
		}
		msg = fmt.Sprintf("逐条%s完成：成功 %d，失败 %d，共 %d；当前标签人数 %d", verb, s.OK, s.Failed, s.Total, count)
		s.Status = status
		s.Message = msg
		s.FinishTime = time.Now()
	})
	logger.Info("[粉丝标签逐条重试] 任务结束", zap.String("appid", appid), zap.String("status", status), zap.String("message", msg))
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
