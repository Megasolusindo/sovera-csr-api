package main

import (
	"log"

	"github.com/hibiken/asynq"
)

func main() {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	defer client.Close()

	for i := 0; i < 5; i++ {
		task := asynq.NewTask("task:url_health_check", []byte("{}"), asynq.Queue("url-health-check-queue"), asynq.MaxRetry(1))
		info, err := client.Enqueue(task)
		if err != nil {
			log.Fatalf("Could not enqueue task: %v", err)
		}
		log.Printf("Successfully enqueued URL health check task ID: %s", info.ID)
	}
}
