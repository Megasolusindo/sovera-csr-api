package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelegramNotifier_IsEnabled(t *testing.T) {
	n1 := NewNotifier("", "")
	assert.False(t, n1.IsEnabled())

	n2 := NewNotifier("token123", "chat456")
	assert.True(t, n2.IsEnabled())
}

func TestTelegramNotifier_SendNotification_Disabled(t *testing.T) {
	n := NewNotifier("", "")
	err := n.SendNotification(context.Background(), "Test message")
	assert.NoError(t, err)
}
