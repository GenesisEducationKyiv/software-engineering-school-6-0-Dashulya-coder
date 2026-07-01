package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const restTimeout = 30 * time.Second

type RESTClient struct {
	baseURL string
	http    *http.Client
}

func NewRESTClient(baseURL string) *RESTClient {
	return &RESTClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: restTimeout},
	}
}

func (c *RESTClient) ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) error {
	return c.post(ctx, "/v1/confirmations/reserve", map[string]string{
		"saga_id":     sagaID,
		"email":       email,
		"confirm_url": confirmURL,
	})
}

func (c *RESTClient) CommitConfirmation(ctx context.Context, sagaID string) error {
	return c.post(ctx, "/v1/confirmations/commit", map[string]string{"saga_id": sagaID})
}

func (c *RESTClient) CancelConfirmation(ctx context.Context, sagaID string) error {
	return c.post(ctx, "/v1/confirmations/cancel", map[string]string{"saga_id": sagaID})
}

func (c *RESTClient) post(ctx context.Context, path string, payload map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", path, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s request: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", path, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close notifier rest response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notifier rest %s: status %d", path, resp.StatusCode)
	}

	return nil
}

func (c *RESTClient) Close() error {
	c.http.CloseIdleConnections()
	return nil
}
