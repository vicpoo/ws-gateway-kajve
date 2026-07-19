// Package application contiene la orquestación del hexágono: no sabe nada
// de HTTP ni de WebSockets (eso vive en infrastructure/websocket), solo
// conoce los puertos definidos en domain. Mismo criterio que
// ingesta-iot/application/ingesta_service.go.
package application

import (
	"context"
	"errors"

	"github.com/kajve/ws-gateway/src/domain"
	"github.com/kajve/ws-gateway/src/domain/entities"
	"github.com/kajve/ws-gateway/src/infrastructure/auth"
)

// Errores de autorización, para que infrastructure/websocket decida el
// código HTTP correcto sin tener que comparar strings.
var (
	ErrTokenInvalido    = errors.New("token inválido")
	ErrLoteNoEncontrado = errors.New("lote not found")
	ErrNoAutorizado     = errors.New("unauthorized")
)

// GatewayService orquesta la autorización de una sesión de WebSocket y la
// preparación de sus datos: validar el JWT, confirmar que el lote sea del
// usuario del token, armar el histórico inicial y suscribirse a los
// eventos en vivo.
type GatewayService struct {
	lotes      domain.LoteRepository
	lecturas   domain.LecturaRepository
	events     domain.EventSubscriber
	validator  *auth.Validator
	historyLen int
}

func NewGatewayService(
	lotes domain.LoteRepository,
	lecturas domain.LecturaRepository,
	events domain.EventSubscriber,
	validator *auth.Validator,
	historyLen int,
) *GatewayService {
	return &GatewayService{
		lotes:      lotes,
		lecturas:   lecturas,
		events:     events,
		validator:  validator,
		historyLen: historyLen,
	}
}

// Autorizar valida el token y confirma que el lote pertenezca al usuario
// del token. Devuelve el id_usuario ya confirmado.
func (s *GatewayService) Autorizar(ctx context.Context, token string, loteID int) (usuarioID int, err error) {
	claims, err := s.validator.Validate(token)
	if err != nil {
		return 0, ErrTokenInvalido
	}

	lote, err := s.lotes.GetByID(ctx, loteID)
	if err != nil {
		return 0, err
	}
	if lote == nil {
		return 0, ErrLoteNoEncontrado
	}
	if lote.UsuarioID != claims.UserID {
		return 0, ErrNoAutorizado
	}
	return claims.UserID, nil
}

// Historial arma el mensaje de histórico inicial de un lote, para que la
// gráfica no arranque vacía. Un error de BD aquí no es fatal: el stream en
// vivo sigue funcionando igual, solo se manda un histórico vacío.
func (s *GatewayService) Historial(ctx context.Context, loteID, usuarioID int) entities.HistorialMessage {
	puntos, err := s.lecturas.GetUltimas(ctx, loteID, usuarioID, s.historyLen)
	if err != nil {
		puntos = nil
	}
	return entities.HistorialMessage{
		Tipo:   "historial",
		LoteID: loteID,
		Puntos: puntos,
	}
}

// Suscribir se suscribe al canal de eventos en vivo del usuario (Redis
// Pub/Sub, canal user:<id>, publicado por ingesta-iot).
func (s *GatewayService) Suscribir(ctx context.Context, usuarioID int) (<-chan []byte, func(), error) {
	return s.events.Subscribe(ctx, usuarioID)
}