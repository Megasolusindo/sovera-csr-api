package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/hibiken/asynq"
)

type OpenClawWorker struct{}

func NewOpenClawWorker() *OpenClawWorker {
	return &OpenClawWorker{}
}

func (w *OpenClawWorker) HandleOpenClawResearchTask(ctx context.Context, t *asynq.Task) error {
	var payload map[string]string
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal OpenClaw research payload: %w", err)
	}

	companyName := payload["company_name"]
	if companyName == "" {
		companyName = "PT Pertamina Patra Niaga"
	}

	log.Printf("[OpenClaw Worker] Triggering OpenClaw AI Research Agent for target: %s", companyName)

	// Execute OpenClaw CLI in container
	cmd := exec.CommandContext(ctx, "node", "/app/agent.js", fmt.Sprintf("--company=%s", companyName))
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("[OpenClaw Worker Warning] Agent run returned error: %v. Output:\n%s", err, string(output))
		return fmt.Errorf("OpenClaw agent execution failed: %w", err)
	}

	log.Printf("[OpenClaw Worker Success] Agent run finished cleanly. Output:\n%s", string(output))
	return nil
}
