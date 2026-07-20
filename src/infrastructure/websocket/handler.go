//src/infrastructure/websocket/handler.go
// Package websocket es el adaptador de transporte del endpoint
// GET /ws/lotes/{id}: sube la conexión a WebSocket, hace ping/pong, y
// escribe los mensajes que le da application.GatewayService — no valida
// tokens ni consulta la BD directamente, eso vive en application.
package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/kajve/ws-gateway/src/application"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// Handler agrupa las dependencias del endpoint WebSocket.
type Handler struct {
	gateway  *application.GatewayService
	upgrader websocket.Upgrader
}

func NewHandler(gateway *application.GatewayService, allowedOrigin string) *Handler {
	return &Handler{
		gateway: gateway,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if allowedOrigin == "" || allowedOrigin == "*" {
					return true
				}
				return r.Header.Get("Origin") == allowedOrigin
			},
		},
	}
}

// ServeWS maneja GET /ws/lotes/{id}?token=<jwt>.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	loteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid lote id"}`, http.StatusBadRequest)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
		return
	}

	usuarioID, err := h.gateway.Autorizar(r.Context(), token, loteID)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrTokenInvalido):
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		case errors.Is(err, application.ErrLoteNoEncontrado):
			http.Error(w, `{"error":"lote not found"}`, http.StatusNotFound)
		case errors.Is(err, application.ErrNoAutorizado):
			http.Error(w, `{"error":"unauthorized"}`, http.StatusForbidden)
		default:
			log.Printf("ws: error autorizando (lote=%d): %v", loteID, err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: error en el upgrade: %v", err)
		return
	}

	h.serve(conn, usuarioID, loteID)
}

func (h *Handler) serve(conn *websocket.Conn, usuarioID, loteID int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer conn.Close()

	events, unsubscribe, err := h.gateway.Suscribir(ctx, usuarioID)
	if err != nil {
		log.Printf("ws: Redis no disponible, sirviendo solo histórico sin datos en vivo (usuario=%d): %v", usuarioID, err)
		unsubscribe = func() {}
	}
	defer unsubscribe()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				// Antes esto se tragaba el error en silencio, así que no
				// había forma de saber si una desconexión era el cliente
				// cerrando limpio, la red cayéndose, o un timeout de
				// pong. Con esto en el log queda claro cuál fue.
				log.Printf("ws: lectura terminada (usuario=%d, lote=%d): %v", usuarioID, loteID, err)
				return
			}
		}
	}()

	historial := h.gateway.Historial(ctx, loteID, usuarioID)
	if err := h.writeJSON(conn, historial); err != nil {
		log.Printf("ws: error mandando histórico (usuario=%d, lote=%d): %v", usuarioID, loteID, err)
		return
	}

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	inicio := time.Now()
	for {
		select {
		case <-ctx.Done():
			log.Printf("ws: conexión cerrada (usuario=%d, lote=%d, duró %v)", usuarioID, loteID, time.Since(inicio))
			return
		case payload, ok := <-events:
			if !ok {
				log.Printf("ws: canal de eventos cerrado (usuario=%d, lote=%d)", usuarioID, loteID)
				return
			}
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf("ws: error escribiendo lectura (usuario=%d, lote=%d): %v", usuarioID, loteID, err)
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ws: error mandando ping — el cliente probablemente ya no responde (usuario=%d, lote=%d): %v", usuarioID, loteID, err)
				return
			}
		}
	}
}

func (h *Handler) writeJSON(conn *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.TextMessage, b)
}