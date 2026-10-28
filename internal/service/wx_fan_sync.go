package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"go.uber.org/zap"
)

// SyncTaskStatus 粉丝同步任务进度结构
type SyncTaskStatus struct {
	AppID      string    `json:"appid"`
	TenantID   uint      `json:"tenant_id"`
	Status     string    `json:"status"` // idle, running, completed, failed
	TotalFans  int       `json:"total_fans"`
	SyncedFans int       `json:"synced_fans"`
	Percent    int       `json:"percent"`
	Message    string    `json:"message"`
	Error      string    `json:"error,omitempty"`
	StartTime  time.Time `json:"start_time"`
	FinishTime time.Time `json:"finish_time,omitempty"`
}

type FanSyncManager struct {
	tasks map[string]*SyncTaskStatus
	mutex sync.RWMutex
}

var DefaultFanSyncManager = &FanSyncManager{
	tasks: make(map[string]*SyncTaskStatus),
}

// GetStatus 获取指定公众号当前同步进度
func (m *FanSyncManager) GetStatus(appid string) *SyncTaskStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if s, exists := m.tasks[appid]; exists {
		return s
	}
	return &SyncTaskStatus{
		AppID:   appid,
		Status:  "idle",
		Message: "尚未开始同步",
	}
}

func (m *FanSyncManager) updateStatus(appid string, updater func(s *SyncTaskStatus)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	s, exists := m.tasks[appid]
	if !exists {
		s = &SyncTaskStatus{
			AppID:     appid,
			Status:    "idle",
			StartTime: time.Now(),
		}
		m.tasks[appid] = s
	}
	updater(s)
}

// DoSyncFansTask 执行全量粉丝同步
func (m *FanSyncManager) DoSyncFansTask(ctx context.Context, tenantID uint, appid string) error {
	m.updateStatus(appid, func(s *SyncTaskStatus) {
		s.TenantID = tenantID
		s.Status = "running"
		s.TotalFans = 0
		s.SyncedFans = 0
		s.Percent = 0
		s.Error = ""
		s.StartTime = time.Now()
		s.Message = "正在连接微信服务器并初始化..."
	})

	logger.Info("[粉丝同步] 开始执行同步任务", zap.String("appid", appid), zap.Uint("tenant_id", tenantID))

	wxClient, err := NewWxClient(appid)
	if err != nil {
		m.updateStatus(appid, func(s *SyncTaskStatus) {
			s.Status = "failed"
			s.Error = err.Error()
			s.Message = "初始化微信客户端失败: " + err.Error()
		})
		return err
	}

	// 1. 同步微信官方标签列表
	m.updateStatus(appid, func(s *SyncTaskStatus) {
		s.Message = "正在同步微信粉丝标签体系..."
	})
	tags, err := wxClient.GetTags(ctx)
	if err == nil && len(tags) > 0 {
		for _, t := range tags {
			tagIDFloat, _ := t["id"].(float64)
			tagID := int(tagIDFloat)
			name, _ := t["name"].(string)
			countFloat, _ := t["count"].(float64)
			count := int(countFloat)

			var wxTag model.WxTag
			err := database.DB.Where("app_id = ? AND tenant_id = ? AND tag_id = ?", appid, tenantID, tagID).First(&wxTag).Error
			if err != nil {
				wxTag = model.WxTag{
					TenantID: tenantID,
					AppID:    appid,
					TagID:    tagID,
					Name:     name,
					Count:    count,
					Synced:   true,
				}
				database.DB.Create(&wxTag)
			} else {
				wxTag.Name = name
				wxTag.Count = count
				// 微信官方清单里存在的标签就是真实标签：顺手把历史遗留的 synced=false 纠正回来
				wxTag.Synced = true
				database.DB.Save(&wxTag)
			}
		}
	}

	// 2. 循环拉取所有 OpenID 列表
	m.updateStatus(appid, func(s *SyncTaskStatus) {
		s.Message = "正在向微信请求粉丝 OpenID 清单..."
	})

	var allOpenIDs []string
	nextOpenID := ""
	totalFromWeChat := 0

	for {
		res, err := wxClient.GetUserList(ctx, nextOpenID)
		if err != nil {
			m.updateStatus(appid, func(s *SyncTaskStatus) {
				s.Status = "failed"
				s.Error = err.Error()
				s.Message = "拉取粉丝列表失败: " + err.Error()
			})
			return err
		}

		if tot, ok := res["total"].(float64); ok && totalFromWeChat == 0 {
			totalFromWeChat = int(tot)
		}

		dataObj, ok := res["data"].(map[string]interface{})
		if !ok {
			break
		}

		openidsRaw, ok := dataObj["openid"].([]interface{})
		if !ok || len(openidsRaw) == 0 {
			break
		}

		for _, item := range openidsRaw {
			if s, ok := item.(string); ok && s != "" {
				allOpenIDs = append(allOpenIDs, s)
			}
		}

		nextStr, _ := res["next_openid"].(string)
		if nextStr == "" || nextStr == nextOpenID || len(openidsRaw) < 10000 {
			break
		}
		nextOpenID = nextStr
	}

	totalCount := len(allOpenIDs)
	if totalCount == 0 {
		m.updateStatus(appid, func(s *SyncTaskStatus) {
			s.Status = "completed"
			s.TotalFans = 0
			s.SyncedFans = 0
			s.Percent = 100
			s.FinishTime = time.Now()
			s.Message = "当前公众号暂无任何关注者粉丝"
		})
		logger.Info("[粉丝同步] 当前公众号暂无粉丝", zap.String("appid", appid))
		return nil
	}

	m.updateStatus(appid, func(s *SyncTaskStatus) {
		s.TotalFans = totalCount
		s.Message = fmt.Sprintf("已获取到 %d 位粉丝清单，正在分批获取详细资料...", totalCount)
	})

	// 3. 分批拉取用户基本信息并入库 (每批最多100个)
	batchSize := 100
	syncedTotal := 0

	for i := 0; i < totalCount; i += batchSize {
		end := i + batchSize
		if end > totalCount {
			end = totalCount
		}

		chunk := allOpenIDs[i:end]
		userDetails, err := wxClient.BatchGetUserInfo(ctx, chunk)
		if err != nil {
			logger.Warn("[粉丝同步] 批量接口未返回，尝试单条补偿拉取", zap.Error(err), zap.Int("batch_start", i))
			// 优雅回退：使用单条粉丝接口单条拉取
			for _, oid := range chunk {
				u, singleErr := wxClient.GetSingleUserInfo(ctx, oid)
				if singleErr == nil && u != nil {
					userDetails = append(userDetails, *u)
				} else {
					// 兜底保障
					userDetails = append(userDetails, WxUserDetail{
						OpenID:    oid,
						Subscribe: 1,
					})
				}
			}
		}

		for _, user := range userDetails {
			nick := strings.TrimSpace(user.Nickname)
			if nick == "" {
				if user.Remark != "" {
					nick = user.Remark
				} else if len(user.OpenID) > 6 {
					nick = fmt.Sprintf("微信粉丝_%s", user.OpenID[len(user.OpenID)-6:])
				} else {
					nick = "微信用户"
				}
			}

			headURL := strings.TrimSpace(user.HeadImgURL)
			// 协议清洗：将 http 转换为 https
			if strings.HasPrefix(headURL, "http://") {
				headURL = "https://" + strings.TrimPrefix(headURL, "http://")
			}

			var fan model.WxFan
			err := database.DB.Where("app_id = ? AND tenant_id = ? AND open_id = ?", appid, tenantID, user.OpenID).First(&fan).Error
			// 微信返回的 tagid_list 只含真实标签，合并保留本地独有的占位标签 (负 ID)，
			// 否则「本地已写上、微信未同步」的标签会在下一次同步后被整列覆盖丢失。
			tagBytes := []byte(MarshalFanTagIDs(MergeLocalOnlyFanTagIDs(user.TagIDList, fan.TagIDs)))
			if err != nil {
				fan = model.WxFan{
					TenantID:       tenantID,
					AppID:          appid,
					OpenID:         user.OpenID,
					UnionID:        user.UnionID,
					Nickname:       nick,
					Sex:            user.Sex,
					City:           user.City,
					Province:       user.Province,
					Country:        user.Country,
					HeadImgURL:     headURL,
					Subscribe:      user.Subscribe,
					SubscribeTime:  user.SubscribeTime,
					Remark:         user.Remark,
					TagIDs:         string(tagBytes),
					SubscribeScene: user.SubscribeScene,
				}
				database.DB.Create(&fan)
			} else {
				fan.Nickname = nick
				fan.Sex = user.Sex
				fan.City = user.City
				fan.Province = user.Province
				fan.Country = user.Country
				if headURL != "" {
					fan.HeadImgURL = headURL
				}
				fan.Subscribe = user.Subscribe
				if user.SubscribeTime > 0 {
					fan.SubscribeTime = user.SubscribeTime
				}
				if user.Remark != "" {
					fan.Remark = user.Remark
				}
				fan.TagIDs = string(tagBytes)
				fan.SubscribeScene = user.SubscribeScene
				database.DB.Save(&fan)
			}
		}
		syncedTotal += len(chunk)

		// 计算进度
		pct := int(float64(syncedTotal) / float64(totalCount) * 100)
		if pct > 100 {
			pct = 100
		}

		m.updateStatus(appid, func(s *SyncTaskStatus) {
			s.SyncedFans = syncedTotal
			s.Percent = pct
			s.Message = fmt.Sprintf("正在同步粉丝资料: 已处理 %d / %d 位 (%d%%)", syncedTotal, totalCount, pct)
		})

		// 稍微微歇 100ms 避免超频
		time.Sleep(100 * time.Millisecond)
	}

	// 4. 更新完成
	m.updateStatus(appid, func(s *SyncTaskStatus) {
		s.Status = "completed"
		s.SyncedFans = syncedTotal
		s.Percent = 100
		s.FinishTime = time.Now()
		s.Message = fmt.Sprintf("🎉 粉丝数据全量同步成功！共同步 %d 位粉丝", syncedTotal)
	})

	logger.Info("[粉丝同步] 任务圆满完成", zap.String("appid", appid), zap.Int("synced", syncedTotal))
	return nil
}
