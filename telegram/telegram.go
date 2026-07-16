// Package telegram provides a client for the Telegram Bot API sendMessage method.
//
// This is kept simple (GET request with query params) for clarity.
// Production use would use POST with JSON body for longer messages.
package telegram

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Config holds Telegram Bot credentials.
type Config struct {
	BotToken string
	ChatID   string
}

// Client sends messages via the Telegram Bot API.
type Client struct {
	cfg    Config
	http   *http.Client
}

// NewClient creates a Telegram client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SendMessage sends a text message to the configured chat.
func (c *Client) SendMessage(text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.cfg.BotToken)

	params := url.Values{}
	params.Set("chat_id", c.cfg.ChatID)
	params.Set("text", text)

	resp, err := c.http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
