package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/kajve/ws-gateway/src/infrastructure/httpapi"
	wsinfra "github.com/kajve/ws-gateway/src/infrastructure/websocket"
)

// NewRouter arma las rutas del Gateway: /health, el endpoint WebSocket, y
// los endpoints REST auxiliares (por ahora, estado de sensores).
func NewRouter(wsHandler *wsinfra.Handler, sensorStatusHandler *httpapi.SensorStatusHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"ws-gateway"}`))
	})

	r.Get("/ws/lotes/{id}", wsHandler.ServeWS)
	r.Get("/sensores/estado", sensorStatusHandler.ServeHTTP)

	return r
}
