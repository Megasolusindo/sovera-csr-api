package queue

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeDispatchCrawling   = "task:dispatch_crawling"
	TypePollPendingTasks   = "task:poll_pending_tasks"
	TypeRawIngestion       = "task:raw_ingestion"
	TypeLLMExtraction      = "task:llm_extraction"
	TypeESGExtraction      = "task:esg_extraction"
	TypeProposalGeneration = "task:proposal_generation"
	TypeIDXSync            = "task:idx_sync"

	QueueDispatchCrawling   = "dispatch-crawling-queue"
	QueuePollPendingTasks   = "poll-pending-tasks-queue"
	QueueRawIngestion       = "raw-ingestion-queue"
	QueueLLMExtraction      = "llm-extraction-queue"
	QueueESGExtraction      = "esg-extraction-queue"
	QueueProposalGeneration = "proposal-generation-queue"
	QueueIDXSync            = "idx-sync-queue"
)

type LLMExtractionPayload struct {
	JobID           string `json:"job_id"`
	TaskID          string `json:"task_id"`
	SourceType      string `json:"source_type"`
	SourceURL       string `json:"source_url"`
	AuthorOrAccount string `json:"author_or_account"`
	PublishedDate   string `json:"published_date"`
	RawText         string `json:"raw_text"`
	MarkdownContent string `json:"markdown_content"`
	ContentHash     string `json:"content_hash"`
}

type ESGExtractionPayload struct {
	JobID           string  `json:"job_id"`
	CompanyID       *string `json:"company_id,omitempty"`
	CompanyName     string  `json:"company_name"`
	IndustrySector  string  `json:"industry_sector"`
	RawText         string  `json:"raw_text"`
	MarkdownContent string  `json:"markdown_content"`
}

func NewDispatchCrawlingTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeDispatchCrawling, []byte("{}"), asynq.Queue(QueueDispatchCrawling), asynq.MaxRetry(3)), nil
}

func NewPollPendingTasksTask() (*asynq.Task, error) {
	return asynq.NewTask(TypePollPendingTasks, []byte("{}"), asynq.Queue(QueuePollPendingTasks), asynq.MaxRetry(3)), nil
}

func NewLLMExtractionTask(payload LLMExtractionPayload) (*asynq.Task, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extraction task payload: %w", err)
	}

	return asynq.NewTask(TypeLLMExtraction, bytes, asynq.Queue(QueueLLMExtraction), asynq.MaxRetry(5)), nil
}

func NewESGExtractionTask(payload ESGExtractionPayload) (*asynq.Task, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ESG extraction task payload: %w", err)
	}

	return asynq.NewTask(TypeESGExtraction, bytes, asynq.Queue(QueueESGExtraction), asynq.MaxRetry(3)), nil
}
