package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	wsinfra "github.com/kajve/ws-gateway/src/infrastructure/websocket"
)

// NewRouter arma las rutas del Gateway: /health y el endpoint WebSocket.
func NewRouter(wsHandler *wsinfra.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"ws-gateway"}`))
	})

	r.Get("/ws/lotes/{id}", wsHandler.ServeWS)

	return r
}
