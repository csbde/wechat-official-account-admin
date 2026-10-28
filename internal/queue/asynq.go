package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

const (
	TypeSyncFans         = "wechat:sync_fans"
	TypeDownloadMaterial = "wechat:download_material"
	TypeBatchTagging     = "wechat:batch_tagging"
)

var Client *asynq.Client
var Server *asynq.Server

type SyncFansPayload struct {
	TenantID uint   `json:"tenant_id"`
	AppID    string `json:"appid"`
}

// BatchTaggingPayload 粉丝标签逐条重试任务参数
// 整批 (50 人) 被微信拒绝时派发：逐个 openid 调用，避免一人失败拖垮整批
type BatchTaggingPayload struct {
	TenantID uint     `json:"tenant_id"`
	AppID    string   `json:"appid"`
	TagID    int      `json:"tag_id"`
	Action   string   `json:"action"` // add / remove
	OpenIDs  []string `json:"openids"`
}

func Init(cfg *config.Config) (*asynq.Client, *asynq.Server, error) {
	if cfg.Redis.Addr == "" {
		return nil, nil, fmt.Errorf("redis addr is empty")
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOpt)
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Asynq.Concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	Client = client
	Server = server

	logger.Info("Asynq queue client initialized")
	return Client, Server, nil
}

// EnqueueSyncFans 派发异步任务
func EnqueueSyncFans(tenantID uint, appid string) error {
	if Client == nil {
		return fmt.Errorf("asynq client is nil (redis not configured)")
	}
	payload, _ := json.Marshal(SyncFansPayload{TenantID: tenantID, AppID: appid})
	task := asynq.NewTask(TypeSyncFans, payload)
	info, err := Client.Enqueue(task, asynq.Queue("default"), asynq.MaxRetry(3))
	if err != nil {
		return err
	}
	logger.Info("Enqueued sync fans task", zap.String("task_id", info.ID), zap.String("appid", appid))
	return nil
}

// EnqueueBatchTaggingRetry 派发标签逐条重试任务 (低优先级队列，不挤占粉丝同步)
func EnqueueBatchTaggingRetry(p BatchTaggingPayload) error {
	if Client == nil {
		return fmt.Errorf("asynq client is nil (redis not configured)")
	}
	payload, _ := json.Marshal(p)
	task := asynq.NewTask(TypeBatchTagging, payload)
	info, err := Client.Enqueue(task, asynq.Queue("low"), asynq.MaxRetry(1), asynq.Timeout(30*time.Minute))
	if err != nil {
		return err
	}
	logger.Info("Enqueued fan tag retry task",
		zap.String("task_id", info.ID),
		zap.String("appid", p.AppID),
		zap.Int("tag_id", p.TagID),
		zap.Int("openids", len(p.OpenIDs)),
	)
	return nil
}

func StartWorker(server *asynq.Server, handler asynq.Handler) {
	go func() {
		if err := server.Run(handler); err != nil {
			logger.Error("Asynq worker exited with error", zap.Error(err))
		}
	}()
}

func NewServeMux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeSyncFans, func(ctx context.Context, t *asynq.Task) error {
		var p SyncFansPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		logger.Info("Processing sync fans Asynq task", zap.String("appid", p.AppID))
		return service.DefaultFanSyncManager.DoSyncFansTask(ctx, p.TenantID, p.AppID)
	})
	mux.HandleFunc(TypeBatchTagging, func(ctx context.Context, t *asynq.Task) error {
		var p BatchTaggingPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		logger.Info("Processing fan tag retry Asynq task",
			zap.String("appid", p.AppID), zap.Int("tag_id", p.TagID), zap.Int("openids", len(p.OpenIDs)))
		return service.DefaultFanTagRetryManager.RetryTaggingOneByOne(ctx, p.TenantID, p.AppID, p.TagID, p.Action, p.OpenIDs)
	})
	return mux
}
