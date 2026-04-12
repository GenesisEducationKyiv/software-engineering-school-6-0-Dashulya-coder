package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
)

func New(handler *handlers.SubscriptionHandler) http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Post("/subscribe", handler.Subscribe)
		r.Get("/confirm/{token}", handler.Confirm)
		r.Get("/unsubscribe/{token}", handler.Unsubscribe)
		r.Get("/subscriptions", handler.GetSubscriptions)
	})

	r.Handle("/metrics", promhttp.Handler())

	return r
}
