//src/infrastructure/httpapi/resumen_handler.go
package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kajve/ws-gateway/src/application"
)

// ResumenHandler expone GET /lotes/{id}/resumen: agregados (min/prom/max)
// de todas las lecturas del lote, calculados directo sobre Postgres desde
// este servicio — no depende de api-mobile. Pensado para alimentar la
// vista de Monitoreo con algo que Tiempo Real y Sensores no dan: el
// panorama completo del lote, no solo lo último o si el hardware sigue
// conectado.
type ResumenHandler struct {
	gateway *application.GatewayService
}

func NewResumenHandler(gateway *application.GatewayService) *ResumenHandler {
	return &ResumenHandler{gateway: gateway}
}

// ServeHTTP maneja GET /lotes/{id}/resumen con `Authorization: Bearer <jwt>`.
// Autorizar ya valida el token Y confirma que el lote sea del usuario del
// token en una sola llamada (mismo método que usa el WebSocket).
func (h *ResumenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lote id")
		return
	}

	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	usuarioID, err := h.gateway.Autorizar(r.Context(), token, loteID)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrTokenInvalido):
			writeError(w, http.StatusUnauthorized, "invalid token")
		case errors.Is(err, application.ErrLoteNoEncontrado):
			writeError(w, http.StatusNotFound, "lote not found")
		case errors.Is(err, application.ErrNoAutorizado):
			writeError(w, http.StatusForbidden, "unauthorized")
		default:
			log.Printf("httpapi: error autorizando resumen (lote=%d): %v", loteID, err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	resumen, err := h.gateway.Resumen(r.Context(), loteID, usuarioID)
	if err != nil {
		log.Printf("httpapi: error calculando resumen (lote=%d): %v", loteID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, resumen)
}
