package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/metrics"
)

func New(handler *handlers.SubscriptionHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(metrics.Handler)

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "web/index.html")
	})
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/subscribe", handler.Subscribe)
		r.Get("/confirm/{token}", handler.Confirm)
		r.Get("/unsubscribe/{token}", handler.Unsubscribe)
		r.Get("/subscriptions", handler.GetSubscriptions)
	})

	r.Handle("/metrics", promhttp.Handler())

	return r
}
