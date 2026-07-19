// Package httpapi contiene los endpoints REST normales del Gateway (no
// WebSocket) — separado de infrastructure/websocket porque la
// autenticación aquí va por header Authorization, no por query param.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/kajve/ws-gateway/src/application"
	"github.com/kajve/ws-gateway/src/domain/entities"
)

// SensorStatusHandler expone GET /sensores/estado.
type SensorStatusHandler struct {
	gateway *application.GatewayService
	status  *application.SensorStatusService
}

func NewSensorStatusHandler(gateway *application.GatewayService, status *application.SensorStatusService) *SensorStatusHandler {
	return &SensorStatusHandler{gateway: gateway, status: status}
}

// ServeHTTP maneja GET /sensores/estado con `Authorization: Bearer <jwt>`,
// el mismo esquema que usa el ApiClient de Flutter para el resto de las
// llamadas REST (a diferencia del WebSocket, aquí sí hay headers).
func (h *SensorStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	usuarioID, err := h.gateway.ValidarToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	estados, err := h.status.Estado(r.Context(), usuarioID)
	if err != nil {
		log.Printf("httpapi: error consultando estado de sensores (usuario=%d): %v", usuarioID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if estados == nil {
		estados = []entities.EstadoSensor{}
	}

	writeJSON(w, http.StatusOK, entities.EstadoSensoresResponse{Sensores: estados})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimPrefix(h, prefix)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
