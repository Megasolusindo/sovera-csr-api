package queue

import (
	"context"
	"log"

	"github.com/hibiken/asynq"

	"sovera-core-api/internal/pkg/serper"
	"sovera-core-api/internal/pkg/telegram"
)

const (
	TypeSerperQuotaCheck = "task:check_serper_quota"
	QueueSerperQuota     = "serper-quota-queue"
)

func NewSerperQuotaCheckTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeSerperQuotaCheck, []byte("{}"), asynq.Queue(QueueSerperQuota), asynq.MaxRetry(1)), nil
}

type SerperQuotaWorker struct {
	monitor *serper.SerperMonitor
}

func NewSerperQuotaWorker(apiKey string, notifier *telegram.Notifier) *SerperQuotaWorker {
	mon := serper.InitDefaultMonitor(apiKey, notifier)
	return &SerperQuotaWorker{
		monitor: mon,
	}
}

func (w *SerperQuotaWorker) HandleSerperQuotaCheck(ctx context.Context, t *asynq.Task) error {
	if w.monitor == nil {
		log.Println("[SerperQuotaWorker] Monitor is nil, skipping quota check")
		return nil
	}

	balance, err := w.monitor.CheckAndNotify(ctx)
	if err != nil {
		log.Printf("[SerperQuotaWorker] Error performing Serper quota check: %v", err)
		return err
	}

	log.Printf("[SerperQuotaWorker] Serper quota check completed. Current balance: %d credits", balance)
	return nil
}
