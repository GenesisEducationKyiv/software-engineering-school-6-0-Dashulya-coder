package notification_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/client/notification"
)

func TestRESTClient_ReserveConfirmation(t *testing.T) {
	var gotPath string
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := notification.NewRESTClient(srv.URL)
	err := client.ReserveConfirmation(context.Background(), "s1", "a@b.com", "http://x/c")

	require.NoError(t, err)
	assert.Equal(t, "/v1/confirmations/reserve", gotPath)
	assert.Equal(t, map[string]string{"saga_id": "s1", "email": "a@b.com", "confirm_url": "http://x/c"}, gotBody)
}

func TestRESTClient_ServerErrorIsPropagated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := notification.NewRESTClient(srv.URL)
	err := client.CommitConfirmation(context.Background(), "s1")

	require.Error(t, err)
}
