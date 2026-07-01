package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
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
		if req.SagaID == "" || req.Email == "" || req.ConfirmURL == "" {
			writeError(w, http.StatusBadRequest, "missing required fields")
			return
		}

		ok, err := svc.ReserveConfirmation(r.Context(), req.SagaID, req.Email, req.ConfirmURL)
		if err != nil {
			slog.Error("rest reserve confirmation failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to reserve confirmation")
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
		if req.SagaID == "" {
			writeError(w, http.StatusBadRequest, "missing saga_id")
			return
		}

		ok, err := svc.CommitConfirmation(r.Context(), req.SagaID)
		if err != nil {
			slog.Error("rest commit confirmation failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to commit confirmation")
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
		if req.SagaID == "" {
			writeError(w, http.StatusBadRequest, "missing saga_id")
			return
		}

		if err := svc.CancelConfirmation(r.Context(), req.SagaID); err != nil {
			slog.Error("rest cancel confirmation failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to cancel confirmation")
			return
		}

		writeJSON(w, http.StatusOK, okResponse{OK: true})
	}
}

func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
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
