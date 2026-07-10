package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"

	"github.com/google/uuid"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
)

type Service interface {
	ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) (bool, error)
	CommitConfirmation(ctx context.Context, sagaID string) (bool, error)
	CancelConfirmation(ctx context.Context, sagaID string) error
}

type reserveRequest struct {
	SagaID     string `json:"saga_id"`
	Email      string `json:"email"`
	ConfirmURL string `json:"confirm_url"`
}

type sagaRequest struct {
	SagaID string `json:"saga_id"`
}

type okResponse struct {
	OK bool `json:"ok"`
}

func NewHandler(svc Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/confirmations/reserve", reserveHandler(svc))
	mux.HandleFunc("POST /v1/confirmations/commit", commitHandler(svc))
	mux.HandleFunc("POST /v1/confirmations/cancel", cancelHandler(svc))
	return mux
}

func reserveHandler(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req reserveRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := uuid.Validate(req.SagaID); err != nil {
			writeError(w, http.StatusBadRequest, "invalid saga_id")
			return
		}
		if _, err := mail.ParseAddress(req.Email); err != nil {
			writeError(w, http.StatusBadRequest, "invalid email")
			return
		}
		if !validURL(req.ConfirmURL) {
			writeError(w, http.StatusBadRequest, "invalid confirm_url")
			return
		}

		ok, err := svc.ReserveConfirmation(r.Context(), req.SagaID, req.Email, req.ConfirmURL)
		if err != nil {
			writeServiceError(w, "reserve confirmation", err)
			return
		}

		writeJSON(w, http.StatusOK, okResponse{OK: ok})
	}
}

func commitHandler(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req sagaRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := uuid.Validate(req.SagaID); err != nil {
			writeError(w, http.StatusBadRequest, "invalid saga_id")
			return
		}

		ok, err := svc.CommitConfirmation(r.Context(), req.SagaID)
		if err != nil {
			writeServiceError(w, "commit confirmation", err)
			return
		}

		writeJSON(w, http.StatusOK, okResponse{OK: ok})
	}
}

func cancelHandler(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req sagaRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := uuid.Validate(req.SagaID); err != nil {
			writeError(w, http.StatusBadRequest, "invalid saga_id")
			return
		}

		if err := svc.CancelConfirmation(r.Context(), req.SagaID); err != nil {
			writeServiceError(w, "cancel confirmation", err)
			return
		}

		writeJSON(w, http.StatusOK, okResponse{OK: true})
	}
}

func validURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeServiceError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, delivery.ErrNotReserved):
		writeError(w, http.StatusConflict, "confirmation not reserved")
	case errors.Is(err, delivery.ErrCanceled):
		writeError(w, http.StatusConflict, "confirmation already canceled")
	default:
		slog.Error("rest "+op+" failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to "+op)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("rest write response failed", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
