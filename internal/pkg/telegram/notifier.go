package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Notifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *Notifier) IsEnabled() bool {
	return n.botToken != "" && n.chatID != ""
}

type telegramSendMessagePayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendNotification sends a raw text or markdown message to Telegram Chat
func (n *Notifier) SendNotification(ctx context.Context, message string) error {
	if !n.IsEnabled() {
		log.Println("[Telegram Notifier] Notice: Telegram notifications skipped (TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is empty)")
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	payload := telegramSendMessagePayload{
		ChatID:    n.chatID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create telegram HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	log.Printf("[Telegram Notifier] Notification delivered successfully to Chat ID: %s", n.chatID)
	return nil
}

// SendAlert sends a formatted HTML alert message for system/scraping errors
func (n *Notifier) SendAlert(ctx context.Context, title, severity, details, targetURL string) error {
	severityEmoji := "⚠️"
	if severity == "CRITICAL" || severity == "HIGH" {
		severityEmoji = "🚨"
	} else if severity == "INFO" {
		severityEmoji = "ℹ️"
	}

	message := fmt.Sprintf(
		"<b>%s SOVERA ALERT: %s</b>\n\n"+
			"<b>Severity:</b> %s\n"+
			"<b>Timestamp:</b> %s UTC\n\n"+
			"<b>Details:</b> %s\n",
		severityEmoji,
		title,
		severity,
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		details,
	)

	if targetURL != "" {
		message += fmt.Sprintf("<b>Target URL:</b> <code>%s</code>\n", targetURL)
	}

	return n.SendNotification(ctx, message)
}
