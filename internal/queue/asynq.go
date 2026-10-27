package queue

import (
	"context"
	"encoding/json"
	"fmt"

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
	return mux
}
